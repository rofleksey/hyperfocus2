// Package prune is the usecase enforcing the configured data retention window
// (default 72 hours) across snapshots, sessions, vods, orphan streamers and the
// on-disk preview image files.
package prune

import (
	"context"
	"log/slog"
	"time"

	"hyperfocus/internal/config"
	"hyperfocus/internal/pkg/clock"
)

// Repository is the persistence port for deletion.
type Repository interface {
	DeleteSnapshotsBefore(ctx context.Context, cutoff time.Time) (int64, error)
	DeleteSessionsEndedBefore(ctx context.Context, cutoff time.Time) (int64, error)
	DeleteOrphanStreamers(ctx context.Context, cutoff time.Time) (int64, error)
}

// PreviewStore sweeps expired image files.
type PreviewStore interface {
	Sweep(ctx context.Context, olderThan time.Time) (int, error)
}

// Deps holds collaborators for New.
type Deps struct {
	Clock   clock.Clock
	Logger  *slog.Logger
	Repo    Repository
	Preview PreviewStore
	Config  config.Prune
}

// Pruner runs the retention loop.
type Pruner struct {
	clock clock.Clock
	log   *slog.Logger
	repo  Repository
	prev  PreviewStore
	cfg   config.Prune
}

// New builds a Pruner.
func New(d Deps) *Pruner {
	if d.Clock == nil {
		d.Clock = clock.System()
	}
	return &Pruner{clock: d.Clock, log: d.Logger, repo: d.Repo, prev: d.Preview, cfg: d.Config}
}

// Run prunes once immediately at startup (so restarting doesn't postpone
// cleanup) and then by interval, forever, until ctx is cancelled.
func (p *Pruner) Run(ctx context.Context) {
	interval := p.cfg.Interval.Std()
	if interval <= 0 {
		interval = time.Hour
	}
	hours := p.cfg.Hours
	if hours <= 0 {
		hours = 72
	}
	p.log.Info("prune loop starting",
		slog.Duration("interval", interval), slog.Int("retention_hours", hours))

	p.tick(ctx, hours)

	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			p.tick(ctx, hours)
		}
	}
}

func (p *Pruner) tick(ctx context.Context, hours int) {
	if err := p.doPrune(ctx, hours); err != nil {
		p.log.Error("prune cycle failed", slog.Any("error", err))
	}
}

func (p *Pruner) doPrune(ctx context.Context, hours int) error {
	cutoff := p.clock.Now().UTC().Add(-time.Duration(hours) * time.Hour)

	// Order matters: snapshots first (cascades samples), then sessions, vods,
	// orphan streamers, and finally the preview files on disk.
	snaps, err := p.repo.DeleteSnapshotsBefore(ctx, cutoff)
	if err != nil {
		return err
	}
	sess, err := p.repo.DeleteSessionsEndedBefore(ctx, cutoff)
	if err != nil {
		return err
	}
	streamers, err := p.repo.DeleteOrphanStreamers(ctx, cutoff)
	if err != nil {
		return err
	}
	swept, err := p.prev.Sweep(ctx, cutoff)
	if err != nil {
		return err
	}

	if snaps+sess+streamers > 0 || swept > 0 {
		p.log.Info("prune cycle complete",
			slog.Int64("snapshots", snaps),
			slog.Int64("sessions", sess),
			slog.Int64("streamers", streamers),
			slog.Int("previews", swept),
		)
	}
	return nil
}
