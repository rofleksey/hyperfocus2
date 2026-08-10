// Package moments is the read-side usecase powering the "who was online at time
// T" view. It selects the nearest snapshot at or before T and returns that
// moment's stream samples, filtered and sorted, each carrying its own data.
package moments

import (
	"context"
	"errors"
	"log/slog"
	"sort"
	"time"

	"github.com/samber/oops"

	"hyperfocus/internal/entity"
	"hyperfocus/internal/pkg/fuzzy"
)

// Repository is the read port needed by the moments usecase.
type Repository interface {
	SnapshotAtOrBefore(ctx context.Context, t time.Time) (entity.Snapshot, error)
	SnapshotAtOrAfter(ctx context.Context, t time.Time) (entity.Snapshot, error)
	FindSamples(ctx context.Context, snapshotID int64, query string, language string, vod string, sort string, dir string, limit int, offset int) ([]entity.SampleDetail, error)
	ListSnapshots(ctx context.Context, from, to *time.Time, limit int) ([]entity.Snapshot, error)
}

// Params describes a "who was online" query.
type Params struct {
	At       time.Time
	Query    string // streamer name (login/display_name) ILIKE filter
	Survivor string // survivor-name fuzzy search; when set, overrides Sort/Dir
	Language string
	Sort     string // viewers | name | started | login
	Dir      string // asc | desc
	Limit    int
	Offset   int
}

// MomentResult is a single moment's answer.
type MomentResult struct {
	RequestedAt time.Time
	HasData     bool
	Snapshot    entity.Snapshot
	Streams     []entity.SampleDetail
}

// Deps holds collaborators for New.
type Deps struct {
	Logger *slog.Logger
	Repo   Repository
}

// Service answers moment queries.
type Service struct {
	log  *slog.Logger
	repo Repository
}

// New builds a Service.
func New(d Deps) *Service {
	return &Service{log: d.Logger, repo: d.Repo}
}

// MomentAt returns the streams live closest to At, filtered/sorted.
func (s *Service) MomentAt(ctx context.Context, p Params) (MomentResult, error) {
	if p.At.IsZero() {
		p.At = time.Now().UTC()
	}
	res := MomentResult{RequestedAt: p.At}

	s.log.Debug("moment: query",
		slog.Time("at", p.At),
		slog.String("q", p.Query),
		slog.String("survivor", p.Survivor),
		slog.String("lang", p.Language),
		slog.String("sort", p.Sort),
		slog.String("dir", p.Dir))

	before, errBefore := s.repo.SnapshotAtOrBefore(ctx, p.At)
	after, errAfter := s.repo.SnapshotAtOrAfter(ctx, p.At)

	var snap entity.Snapshot
	switch {
	case errBefore == nil && errAfter == nil:
		if p.At.Sub(before.TakenAt) <= after.TakenAt.Sub(p.At) {
			snap = before
		} else {
			snap = after
		}
		s.log.Debug("moment: found two snapshots",
			slog.Int64("before_id", before.ID),
			slog.Time("before_at", before.TakenAt),
			slog.Int64("after_id", after.ID),
			slog.Time("after_at", after.TakenAt),
			slog.Int64("chosen", snap.ID))
	case errBefore == nil:
		snap = before
		s.log.Debug("moment: found snapshot (before)", slog.Int64("id", snap.ID), slog.Time("taken_at", snap.TakenAt))
	case errAfter == nil:
		snap = after
		s.log.Debug("moment: found snapshot (after)", slog.Int64("id", snap.ID), slog.Time("taken_at", snap.TakenAt))
	default:
		s.log.Debug("moment: no snapshot found")
		if errors.Is(errBefore, entity.ErrNotFound) || errors.Is(errAfter, entity.ErrNotFound) {
			return res, nil
		}
		return res, oops.Wrap(errBefore)
	}
	res.Snapshot = snap
	res.HasData = true

	// For survivor search the SQL layer must return all rows so ranking is
	// correct — pagination (limit/offset) is applied in Go after ranking.
	findLimit := p.Limit
	findOffset := p.Offset
	if p.Survivor != "" {
		findLimit = 0
		findOffset = 0
	}
	samples, err := s.repo.FindSamples(ctx, snap.ID, p.Query, p.Language, "all", p.Sort, p.Dir, findLimit, findOffset)
	if err != nil {
		return res, oops.Wrap(err)
	}
	// When a survivor-name search is active it becomes the primary ranking: the
	// user-selected Sort/Dir is ignored and results are ordered by fuzzy score
	// (descending). The matcher is intentionally loose so many results surface.
	if p.Survivor != "" {
		samples = rankBySurvivor(p.Survivor, samples)
		// Apply offset/limit after ranking (all results were fetched from SQL).
		if p.Offset > 0 && p.Offset < len(samples) {
			samples = samples[p.Offset:]
		} else if p.Offset >= len(samples) {
			samples = nil
		}
		if p.Limit > 0 && p.Limit < len(samples) {
			samples = samples[:p.Limit]
		}
	}
	res.Streams = samples
	s.log.Debug("moment: result", slog.Int("samples", len(samples)))
	return res, nil
}

// Snapshots lists available moments within [from, to].
func (s *Service) Snapshots(ctx context.Context, from, to *time.Time, limit int) ([]entity.Snapshot, error) {
	return s.repo.ListSnapshots(ctx, from, to, limit)
}

// rankBySurvivor scores each sample against the survivor query (best fuzzy
// match across its OCR'd survivor names, with a small streamer login/display
// tiebreak so the result still feels right), drops anything below the loose
// threshold, and sorts the rest by score descending. The matched score is
// attached to each returned SampleDetail so the handler can surface it.
func rankBySurvivor(query string, samples []entity.SampleDetail) []entity.SampleDetail {
	type scored struct {
		s    entity.SampleDetail
		best float64
	}
	out := make([]scored, 0, len(samples))
	for _, s := range samples {
		best := fuzzy.BestScore(query, s.SurvivorNames)
		if best < fuzzy.Threshold {
			continue
		}
		out = append(out, scored{s: s, best: best})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].best != out[j].best {
			return out[i].best > out[j].best
		}
		// Stable secondary ordering by viewer count for consistent display.
		return out[i].s.ViewerCount > out[j].s.ViewerCount
	})
	res := make([]entity.SampleDetail, len(out))
	for i, o := range out {
		score := o.best
		o.s.FuzzyScore = &score
		res[i] = o.s
	}
	return res
}
