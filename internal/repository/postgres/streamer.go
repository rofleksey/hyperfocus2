package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"hyperfocus/internal/entity"
)

// ErrNotFound is re-exported from entity for adapter convenience.
var ErrNotFound = entity.ErrNotFound

const upsertStreamerSQL = `
INSERT INTO streamers (twitch_user_id, login, display_name, profile_image_url, last_seen)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (twitch_user_id) DO UPDATE SET
    login            = EXCLUDED.login,
    display_name     = EXCLUDED.display_name,
    profile_image_url = EXCLUDED.profile_image_url,
    last_seen        = EXCLUDED.last_seen;`

// UpsertStreamer inserts or updates a streamer, refreshing last_seen.
func (r *Repository) UpsertStreamer(ctx context.Context, s entity.Streamer) error {
	_, err := r.db(ctx).Exec(ctx, upsertStreamerSQL,
		s.TwitchUserID, s.Login, s.DisplayName, s.ProfileImageURL, s.LastSeen)
	return err
}

// GetStreamer fetches a single streamer by Twitch user id.
func (r *Repository) GetStreamer(ctx context.Context, id string) (entity.Streamer, error) {
	row := r.db(ctx).QueryRow(ctx, `
SELECT twitch_user_id, login, display_name, profile_image_url, last_seen
FROM streamers WHERE twitch_user_id = $1;`, id)
	var s entity.Streamer
	if err := row.Scan(&s.TwitchUserID, &s.Login, &s.DisplayName, &s.ProfileImageURL, &s.LastSeen); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entity.Streamer{}, ErrNotFound
		}
		return entity.Streamer{}, err
	}
	return s, nil
}

// ListStreamers returns streamers whose login/display_name match q (if non-empty).
func (r *Repository) ListStreamers(ctx context.Context, q string, limit int) ([]entity.Streamer, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.db(ctx).Query(ctx, `
SELECT twitch_user_id, login, display_name, profile_image_url, last_seen
FROM streamers
WHERE $2::text = '' OR login ILIKE '%' || $2 || '%' OR display_name ILIKE '%' || $2 || '%'
ORDER BY display_name
LIMIT $1;`, limit, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []entity.Streamer
	for rows.Next() {
		var s entity.Streamer
		if err := rows.Scan(&s.TwitchUserID, &s.Login, &s.DisplayName, &s.ProfileImageURL, &s.LastSeen); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// DeleteOrphanStreamers removes streamers older than cutoff that are no longer
// referenced by any sample, session or vod.
func (r *Repository) DeleteOrphanStreamers(ctx context.Context, cutoff time.Time) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `
DELETE FROM streamers
WHERE last_seen < $1
  AND twitch_user_id NOT IN (
      SELECT streamer_id FROM stream_samples
      UNION SELECT streamer_id FROM stream_sessions
      UNION SELECT streamer_id FROM vods
  );`, cutoff)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// ListStreamerIDsSince returns distinct streamer ids with activity since cutoff.
func (r *Repository) ListStreamerIDsSince(ctx context.Context, cutoff time.Time) ([]string, error) {
	rows, err := r.db(ctx).Query(ctx, `
SELECT DISTINCT streamer_id FROM stream_samples WHERE snapshot_id IN (
    SELECT id FROM snapshots WHERE taken_at >= $1
);`, cutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}
