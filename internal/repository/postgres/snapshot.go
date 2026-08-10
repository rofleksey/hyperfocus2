package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"hyperfocus/internal/entity"
)

// SnapshotStats returns the last N snapshots with aggregated sample counts.
func (r *Repository) SnapshotStats(ctx context.Context, n int) ([]entity.SnapshotStat, error) {
	if n < 1 {
		n = 1
	}
	if n > 500 {
		n = 500
	}
	rows, err := r.db(ctx).Query(ctx, `
SELECT s.id, s.taken_at, s.stream_count,
       COUNT(ss.snapshot_id) FILTER (WHERE ss.preview_filename IS NOT NULL) AS previews_ok,
       COUNT(ss.snapshot_id) FILTER (WHERE cardinality(ss.survivor_names) > 0) AS ocr_ok,
       COUNT(ss.snapshot_id) AS total
FROM (
  SELECT id, taken_at, stream_count FROM snapshots
  ORDER BY taken_at DESC LIMIT $1
) s
LEFT JOIN stream_samples ss ON ss.snapshot_id = s.id
GROUP BY s.id, s.taken_at, s.stream_count
ORDER BY s.taken_at ASC;`, n)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []entity.SnapshotStat
	for rows.Next() {
		var st entity.SnapshotStat
		if err := rows.Scan(&st.ID, &st.TakenAt, &st.StreamCount, &st.PreviewOK, &st.OCROK, &st.Total); err != nil {
			return nil, err
		}
		out = append(out, st)
	}
	return out, rows.Err()
}

// InsertSnapshot creates a snapshot row and returns its id.
func (r *Repository) InsertSnapshot(ctx context.Context, takenAt time.Time, source string, count int) (int64, error) {
	var id int64
	err := r.db(ctx).QueryRow(ctx, `
INSERT INTO snapshots (taken_at, source, stream_count) VALUES ($1, $2, $3) RETURNING id;`,
		takenAt, source, count).Scan(&id)
	return id, err
}

// SnapshotAtOrBefore returns the most recent snapshot at or before t.
func (r *Repository) SnapshotAtOrBefore(ctx context.Context, t time.Time) (entity.Snapshot, error) {
	row := r.db(ctx).QueryRow(ctx, `
SELECT id, taken_at, source, stream_count FROM snapshots
WHERE taken_at <= $1 ORDER BY taken_at DESC LIMIT 1;`, t)
	var s entity.Snapshot
	if err := row.Scan(&s.ID, &s.TakenAt, &s.Source, &s.StreamCount); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entity.Snapshot{}, ErrNotFound
		}
		return entity.Snapshot{}, err
	}
	return s, nil
}

// SnapshotAtOrAfter returns the earliest snapshot at or after t.
func (r *Repository) SnapshotAtOrAfter(ctx context.Context, t time.Time) (entity.Snapshot, error) {
	row := r.db(ctx).QueryRow(ctx, `
SELECT id, taken_at, source, stream_count FROM snapshots
WHERE taken_at >= $1 ORDER BY taken_at ASC LIMIT 1;`, t)
	var s entity.Snapshot
	if err := row.Scan(&s.ID, &s.TakenAt, &s.Source, &s.StreamCount); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entity.Snapshot{}, ErrNotFound
		}
		return entity.Snapshot{}, err
	}
	return s, nil
}

// ListSnapshots returns snapshots within [from,to] (nil bounds unbounded).
func (r *Repository) ListSnapshots(ctx context.Context, from, to *time.Time, limit int) ([]entity.Snapshot, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.db(ctx).Query(ctx, `
SELECT id, taken_at, source, stream_count FROM snapshots
WHERE ($1::timestamptz IS NULL OR taken_at >= $1)
  AND ($2::timestamptz IS NULL OR taken_at <= $2)
ORDER BY taken_at DESC LIMIT $3;`, from, to, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []entity.Snapshot
	for rows.Next() {
		var s entity.Snapshot
		if err := rows.Scan(&s.ID, &s.TakenAt, &s.Source, &s.StreamCount); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// DeleteSnapshotsBefore removes snapshots older than cutoff (cascades samples).
func (r *Repository) DeleteSnapshotsBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `DELETE FROM snapshots WHERE taken_at < $1;`, cutoff)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
