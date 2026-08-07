package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"hyperfocus/internal/entity"
)

// UpsertVod inserts or updates a resolved VOD record.
func (r *Repository) UpsertVod(ctx context.Context, v entity.Vod) error {
	var dur *int
	if v.DurationSeconds != nil {
		d := *v.DurationSeconds
		dur = &d
	}
	_, err := r.db(ctx).Exec(ctx, `
INSERT INTO vods (vod_id, streamer_id, stream_id, started_at, duration_seconds, url, thumbnail_url)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (vod_id) DO UPDATE SET
    stream_id        = EXCLUDED.stream_id,
    started_at       = EXCLUDED.started_at,
    duration_seconds = EXCLUDED.duration_seconds,
    url              = EXCLUDED.url,
    thumbnail_url    = EXCLUDED.thumbnail_url;`,
		v.VodID, v.StreamerID, v.StreamID, v.StartedAt, dur, v.URL, v.ThumbnailURL)
	return err
}

// DeleteVodsBefore removes VOD records older than cutoff.
func (r *Repository) DeleteVodsBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `DELETE FROM vods WHERE started_at < $1;`, cutoff)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// GetVod returns a VOD by its ID.
func (r *Repository) GetVod(ctx context.Context, vodID string) (entity.Vod, error) {
	row := r.db(ctx).QueryRow(ctx, `
SELECT vod_id, streamer_id, stream_id, started_at, duration_seconds, url, thumbnail_url
FROM vods WHERE vod_id = $1;`, vodID)

	var v entity.Vod
	err := row.Scan(&v.VodID, &v.StreamerID, &v.StreamID, &v.StartedAt, &v.DurationSeconds, &v.URL, &v.ThumbnailURL)
	if errors.Is(err, pgx.ErrNoRows) {
		return entity.Vod{}, ErrNotFound
	}
	if err != nil {
		return entity.Vod{}, err
	}
	return v, nil
}
