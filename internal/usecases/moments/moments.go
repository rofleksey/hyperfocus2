// Package moments is the read-side usecase powering the "who was online at time
// T" view. It selects the nearest snapshot at or before T and returns that
// moment's stream samples, filtered and sorted, each carrying its own data.
package moments

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/samber/oops"

	"hyperfocus/internal/entity"
)

// Repository is the read port needed by the moments usecase.
type Repository interface {
	SnapshotAtOrBefore(ctx context.Context, t time.Time) (entity.Snapshot, error)
	SnapshotAtOrAfter(ctx context.Context, t time.Time) (entity.Snapshot, error)
	FindSamples(ctx context.Context, snapshotID int64, query string, language string, vod string, sort string, dir string, limit int) ([]entity.SampleDetail, error)
	ListSnapshots(ctx context.Context, from, to *time.Time, limit int) ([]entity.Snapshot, error)
}

// Params describes a "who was online" query.
type Params struct {
	At       time.Time
	Query    string
	Language string
	HasVod   bool
	Vod      string // "all", "has", "no" — replaces HasVod
	Sort     string // viewers | name | started | login
	Dir      string // asc | desc
	Limit    int
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
		slog.String("lang", p.Language),
		slog.Bool("has_vod", p.HasVod),
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

	samples, err := s.repo.FindSamples(ctx, snap.ID, p.Query, p.Language, p.Vod, p.Sort, p.Dir, p.Limit)
	if err != nil {
		return res, oops.Wrap(err)
	}
	res.Streams = samples
	s.log.Debug("moment: result", slog.Int("samples", len(samples)))
	return res, nil
}

// Snapshots lists available moments within [from, to].
func (s *Service) Snapshots(ctx context.Context, from, to *time.Time, limit int) ([]entity.Snapshot, error) {
	return s.repo.ListSnapshots(ctx, from, to, limit)
}
