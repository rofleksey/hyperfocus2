// Package resolvevod is the usecase that attaches vod_ids to live stream
// sessions. Because Twitch does not embed a vod_id in the live stream object,
// we query each broadcaster's archive videos (created in real time while live)
// and match by stream_id. Once resolved, sample vod offsets are recomputed.
package resolvevod

import (
	"context"
	"log/slog"
	"time"

	"github.com/samber/oops"

	"hyperfocus/internal/config"
	"hyperfocus/internal/entity"
	"hyperfocus/internal/pkg/clock"
)

// VideosGateway is the port for fetching a broadcaster's archive videos.
type VideosGateway interface {
	GetVideosByUser(ctx context.Context, userID string) ([]entity.Video, error)
}

// Repository is the persistence port needed by the resolve usecase.
type Repository interface {
	SessionsMissingVod(ctx context.Context, cutoff time.Time) ([]entity.StreamSession, error)
	SetSessionVod(ctx context.Context, sessionID int64, vodID string, vodCreatedAt time.Time) error
	RecomputeSampleOffsets(ctx context.Context, sessionID int64) error
	UpsertVod(ctx context.Context, v entity.Vod) error
}

// Deps holds the collaborators for New.
type Deps struct {
	Clock   clock.Clock
	Logger  *slog.Logger
	Gateway VideosGateway
	Repo    Repository
	Config  config.Vod
}

// Resolver runs the vod-resolution loop.
type Resolver struct {
	clock   clock.Clock
	log     *slog.Logger
	gateway VideosGateway
	repo    Repository
	cfg     config.Vod
}

// New builds a Resolver.
func New(d Deps) *Resolver {
	if d.Clock == nil {
		d.Clock = clock.System()
	}
	return &Resolver{clock: d.Clock, log: d.Logger, gateway: d.Gateway, repo: d.Repo, cfg: d.Config}
}

// Run resolves missing vods forever until ctx is cancelled.
func (r *Resolver) Run(ctx context.Context) {
	interval := r.cfg.ResolveInterval.Std()
	if interval <= 0 {
		interval = 2 * time.Minute
	}
	r.log.Info("vod resolver loop starting", slog.Duration("interval", interval))
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			r.tick(ctx)
		}
	}
}

func (r *Resolver) tick(ctx context.Context) {
	if err := r.doResolve(ctx); err != nil {
		r.log.Error("vod resolve cycle failed", slog.Any("error", err))
	}
}

func (r *Resolver) doResolve(ctx context.Context) error {
	cutoff := r.clock.Now().UTC().Add(-48 * time.Hour)
	sessions, err := r.repo.SessionsMissingVod(ctx, cutoff)
	if err != nil {
		return oops.Wrap(err)
	}
	if len(sessions) == 0 {
		r.log.Debug("vod resolve: no sessions missing vods")
		return nil
	}
	r.log.Debug("vod resolve: candidates", slog.Int("count", len(sessions)))

	seen := make(map[string]struct{}, len(sessions)/4)
	resolved := 0
	const maxStreamersPerCycle = 100
	for _, s := range sessions {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if _, ok := seen[s.StreamerID]; ok {
			continue
		}
		if len(seen) >= maxStreamersPerCycle {
			r.log.Debug("vod resolve: hit streamer cap, deferring rest",
				slog.Int("processed", len(seen)),
				slog.Int("remaining", len(sessions)-len(seen)))
			break
		}
		seen[s.StreamerID] = struct{}{}

		r.log.Debug("vod resolve: fetching videos",
			slog.Int64("session_id", s.ID),
			slog.String("streamer_id", s.StreamerID),
			slog.String("twitch_stream_id", s.TwitchStreamID))

		vctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		videos, err := r.gateway.GetVideosByUser(vctx, s.StreamerID)
		cancel()
		if err != nil {
			r.log.Warn("vod resolve: get videos failed",
				slog.String("streamer", s.StreamerID), slog.Any("error", err))
			continue
		}
		r.log.Debug("vod resolve: got videos",
			slog.String("streamer", s.StreamerID),
			slog.Int("count", len(videos)))

		for _, s2 := range sessions {
			if s2.StreamerID != s.StreamerID || s2.VodID != nil {
				continue
			}
			video := matchVideo(s2, videos)
			if video == nil {
				r.log.Debug("vod resolve: no match for session",
					slog.Int64("session_id", s2.ID),
					slog.String("twitch_stream_id", s2.TwitchStreamID),
					slog.Time("session_started", s2.StartedAt),
					slog.Int("videos_checked", len(videos)))
				continue
			}
			r.log.Debug("vod resolve: matched",
				slog.Int64("session_id", s2.ID),
				slog.String("vod_id", video.VodID),
				slog.String("vod_stream_id", strPtr(video.StreamID)),
				slog.Time("vod_created", video.CreatedAt))
			if err := r.apply(ctx, s2, *video); err != nil {
				r.log.Warn("vod resolve: apply failed", slog.Int64("session", s2.ID), slog.Any("error", err))
				continue
			}
			resolved++
		}
		time.Sleep(300 * time.Millisecond)
	}

	if resolved > 0 {
		r.log.Info("vod resolve cycle complete", slog.Int("resolved", resolved), slog.Int("candidates", len(sessions)))
	}
	return nil
}

func strPtr(s *string) string {
	if s == nil {
		return "<nil>"
	}
	return *s
}

func (r *Resolver) apply(ctx context.Context, s entity.StreamSession, v entity.Video) error {
	durSec := int(v.Duration.Seconds())
	vod := entity.Vod{
		VodID:           v.VodID,
		StreamerID:      s.StreamerID,
		StreamID:        &s.TwitchStreamID,
		StartedAt:       v.CreatedAt,
		DurationSeconds: &durSec,
		URL:             v.URL,
		ThumbnailURL:    v.Thumbnail,
	}
	if err := r.repo.UpsertVod(ctx, vod); err != nil {
		return oops.Wrap(err)
	}
	if err := r.repo.SetSessionVod(ctx, s.ID, v.VodID, v.CreatedAt); err != nil {
		return oops.Wrap(err)
	}
	if err := r.repo.RecomputeSampleOffsets(ctx, s.ID); err != nil {
		return oops.Wrap(err)
	}
	return nil
}

// matchVideo finds the archive video corresponding to a session. Prefers an exact
// stream_id match; falls back to a video created near the session start or end.
func matchVideo(s entity.StreamSession, videos []entity.Video) *entity.Video {
	for i := range videos {
		if videos[i].StreamID != nil && *videos[i].StreamID == s.TwitchStreamID {
			return &videos[i]
		}
	}
	var best *entity.Video
	bestScore := time.Hour
	for i := range videos {
		v := videos[i]
		d1 := absDuration(v.CreatedAt.Sub(s.StartedAt))
		if d1 < bestScore {
			bestScore = d1
			best = &videos[i]
		}
		if s.EndedAt != nil {
			d2 := absDuration(v.CreatedAt.Sub(*s.EndedAt))
			if d2 < bestScore {
				bestScore = d2
				best = &videos[i]
			}
		}
	}
	if best != nil && bestScore <= 15*time.Minute {
		return best
	}
	return nil
}

func absDuration(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}
