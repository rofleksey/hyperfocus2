package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"hyperfocus/internal/entity"
)

// InsertSample records one stream's data for a snapshot. It upserts on the
// (snapshot, session) primary key so a repeated sample is idempotent.
func (r *Repository) InsertSample(ctx context.Context, s entity.StreamSample) error {
	if s.Tags == nil {
		s.Tags = []string{}
	}
	if s.SurvivorNames == nil {
		s.SurvivorNames = []string{}
	}
	_, err := r.db(ctx).Exec(ctx, `
INSERT INTO stream_samples
    (snapshot_id, session_id, streamer_id, viewer_count, title, language, tags, started_at, vod_offset_seconds, preview_filename, thumb_filename, survivor_names)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
ON CONFLICT (snapshot_id, session_id) DO UPDATE SET
    viewer_count       = EXCLUDED.viewer_count,
    title              = EXCLUDED.title,
    language           = EXCLUDED.language,
    tags               = EXCLUDED.tags,
    preview_filename   = EXCLUDED.preview_filename,
    thumb_filename     = EXCLUDED.thumb_filename,
    survivor_names     = EXCLUDED.survivor_names;`,
		s.SnapshotID, s.SessionID, s.StreamerID, s.ViewerCount, s.Title, s.Language, s.Tags,
		s.StartedAt, s.VodOffsetSeconds, s.PreviewFilename, s.ThumbFilename, s.SurvivorNames)
	return err
}

// FindSamples returns the samples for a snapshot joined with streamer display
// fields, optionally filtered by name (login or display_name), language, and
// vod presence, and ordered by a whitelisted sort key. If limit <= 0, ALL
// matching rows are returned.
func (r *Repository) FindSamples(ctx context.Context, snapshotID int64, query string, language string, vod string, sort string, dir string, limit int, offset int) ([]entity.SampleDetail, error) {
	baseSelect := `
SELECT s.snapshot_id, s.session_id, s.streamer_id, s.viewer_count, s.title, s.language,
       s.tags, s.started_at, s.vod_offset_seconds, s.preview_filename, s.thumb_filename, s.survivor_names,
       st.login, st.display_name, st.profile_image_url, sess.vod_id
FROM stream_samples s
JOIN streamers st ON st.twitch_user_id = s.streamer_id
JOIN stream_sessions sess ON sess.id = s.session_id
WHERE s.snapshot_id = $1
  AND ($2::text = '' OR st.login ILIKE '%' || $2 || '%' OR st.display_name ILIKE '%' || $2 || '%')
  AND ($3::text = '' OR s.language = $3)
  AND ($4::text = 'all' OR $4::text = '' OR ($4::text = 'has' AND sess.vod_id IS NOT NULL) OR ($4::text = 'no' AND sess.vod_id IS NULL))
ORDER BY ` + buildOrderBy(sort, dir) + ` NULLS LAST`

	// No limit by default: every stream in the moment is returned.
	if limit <= 0 {
		rows, err := r.db(ctx).Query(ctx, baseSelect+`;`, snapshotID, query, language, vod)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		return scanSampleDetails(rows)
	}

	rows, err := r.db(ctx).Query(ctx, baseSelect+`
LIMIT $5 OFFSET $6;`, snapshotID, query, language, vod, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSampleDetails(rows)
}

// FindSampleByStreamer returns a single sample for a streamer at a specific snapshot.
func (r *Repository) FindSampleByStreamer(ctx context.Context, snapshotID int64, streamerID string) (*entity.SampleDetail, error) {
	row := r.db(ctx).QueryRow(ctx, `
SELECT s.snapshot_id, s.session_id, s.streamer_id, s.viewer_count, s.title, s.language,
       s.tags, s.started_at, s.vod_offset_seconds, s.preview_filename, s.thumb_filename, s.survivor_names,
       st.login, st.display_name, st.profile_image_url, sess.vod_id
FROM stream_samples s
JOIN streamers st ON st.twitch_user_id = s.streamer_id
JOIN stream_sessions sess ON sess.id = s.session_id
WHERE s.snapshot_id = $1 AND s.streamer_id = $2
LIMIT 1;`, snapshotID, streamerID)

	var d entity.SampleDetail
	if err := row.Scan(
		&d.SnapshotID, &d.SessionID, &d.StreamerID, &d.ViewerCount, &d.Title, &d.Language,
		&d.Tags, &d.StartedAt, &d.VodOffsetSeconds, &d.PreviewFilename, &d.ThumbFilename, &d.SurvivorNames,
		&d.Login, &d.DisplayName, &d.ProfileImageURL, &d.VodID,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &d, nil
}

func scanSampleDetails(rows pgx.Rows) ([]entity.SampleDetail, error) {
	var out []entity.SampleDetail
	for rows.Next() {
		var d entity.SampleDetail
		if err := rows.Scan(
			&d.SnapshotID, &d.SessionID, &d.StreamerID, &d.ViewerCount, &d.Title, &d.Language,
			&d.Tags, &d.StartedAt, &d.VodOffsetSeconds, &d.PreviewFilename, &d.ThumbFilename, &d.SurvivorNames,
			&d.Login, &d.DisplayName, &d.ProfileImageURL, &d.VodID,
		); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// buildOrderBy maps a UI sort key + direction to a safe SQL ORDER BY clause.
func buildOrderBy(sort, dir string) string {
	col := "viewer_count"
	switch sort {
	case "viewers":
		col = "viewer_count"
	case "name":
		col = "display_name"
	case "started":
		col = "started_at"
	case "login":
		col = "login"
	}
	direction := "DESC"
	if dir == "asc" {
		direction = "ASC"
	}
	// name/login are more natural ascending; default flip handled by caller value.
	return col + " " + direction
}

// SamplePreviewFilenames returns preview filenames for a snapshot (unused now,
// reserved for explicit file pruning if mtime sweep is ever insufficient).
func (r *Repository) SamplePreviewFilenames(ctx context.Context, snapshotID int64) ([]string, error) {
	rows, err := r.db(ctx).Query(ctx, `
SELECT preview_filename FROM stream_samples
WHERE snapshot_id = $1 AND preview_filename IS NOT NULL;`, snapshotID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var fn string
		if err := rows.Scan(&fn); err != nil {
			return nil, err
		}
		out = append(out, fn)
	}
	return out, rows.Err()
}
