package postgres

import (
	"context"
	"time"

	"hyperfocus/internal/entity"
)

// EnsureOpenSession returns the session id for the given Twitch stream id,
// creating it (open, no end) if it does not yet exist. If a session for that
// stream id exists but was closed (e.g. the stream briefly dropped out of a
// poll cycle, or the Twitch API returned a stale entry), it is reopened: the
// row update takes a lock and clears ended_at, so the hourly prune — which
// only deletes sessions with ended_at IS NOT NULL — can never remove a
// session between the poller capturing its id and inserting samples minutes
// later (a race that used to abort whole snapshots with FK violations).
// Also returns any previously-resolved vod_id so callers can skip redundant
// VOD lookups.
func (r *Repository) EnsureOpenSession(ctx context.Context, streamerID, twitchStreamID string, startedAt time.Time) (int64, *string, error) {
	var id int64
	var vodID *string
	err := r.db(ctx).QueryRow(ctx, `
INSERT INTO stream_sessions (streamer_id, twitch_stream_id, started_at, ended_at)
VALUES ($1, $2, $3, NULL)
ON CONFLICT (twitch_stream_id) DO UPDATE SET started_at = stream_sessions.started_at, ended_at = NULL
RETURNING id, vod_id;`, streamerID, twitchStreamID, startedAt).Scan(&id, &vodID)
	return id, vodID, err
}

// CloseUnseenSessions ends any still-open session whose Twitch stream id is not
// in seenIDs. Returns the number of sessions closed.
func (r *Repository) CloseUnseenSessions(ctx context.Context, seenIDs []string, now time.Time) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `
UPDATE stream_sessions SET ended_at = $1
WHERE ended_at IS NULL
  AND twitch_stream_id <> ALL ($2::text[]);`, now, seenIDs)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// DeleteSessionsEndedBefore removes closed sessions older than cutoff that have
// no remaining samples (samples are removed first via snapshot pruning).
func (r *Repository) DeleteSessionsEndedBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `
DELETE FROM stream_sessions
WHERE ended_at IS NOT NULL AND ended_at < $1
  AND id NOT IN (SELECT session_id FROM stream_samples);`, cutoff)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// ListSessionsForStreamer returns recent sessions for a streamer, newest first.
func (r *Repository) ListSessionsForStreamer(ctx context.Context, streamerID string, limit int) ([]entity.SessionDetail, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.db(ctx).Query(ctx, `
SELECT sess.id, sess.streamer_id, sess.twitch_stream_id, sess.started_at, sess.ended_at,
       sess.vod_id, sess.vod_created_at, st.login, st.display_name, st.profile_image_url
FROM stream_sessions sess
JOIN streamers st ON st.twitch_user_id = sess.streamer_id
WHERE sess.streamer_id = $1
ORDER BY sess.started_at DESC LIMIT $2;`, streamerID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []entity.SessionDetail
	for rows.Next() {
		var d entity.SessionDetail
		if err := rows.Scan(&d.ID, &d.StreamerID, &d.TwitchStreamID, &d.StartedAt, &d.EndedAt,
			&d.VodID, &d.VodCreatedAt, &d.Login, &d.DisplayName, &d.ProfileImageURL); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}
