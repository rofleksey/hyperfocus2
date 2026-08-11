package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"hyperfocus/internal/entity"
)

func (r *Repository) InsertSubscriber(ctx context.Context, sub entity.NotificationSubscriber) (int64, error) {
	var id int64
	err := r.db(ctx).QueryRow(ctx, `
INSERT INTO notification_subscribers (twitch_login, twitch_user_id, steam_url, steam_id, status)
VALUES ($1, $2, $3, $4, 'pending')
RETURNING id;`,
		sub.TwitchLogin, sub.TwitchUserID, sub.SteamURL, sub.SteamID,
	).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (r *Repository) GetSubscriberByTwitch(ctx context.Context, twitchLogin string) (*entity.NotificationSubscriber, error) {
	var s entity.NotificationSubscriber
	err := r.db(ctx).QueryRow(ctx, `
SELECT id, twitch_login, twitch_user_id, steam_url, steam_id, status, created_at, verified_at
FROM notification_subscribers WHERE twitch_login = $1;`, twitchLogin).Scan(
		&s.ID, &s.TwitchLogin, &s.TwitchUserID, &s.SteamURL, &s.SteamID,
		&s.Status, &s.CreatedAt, &s.VerifiedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

func (r *Repository) UpdateSubscriberStatus(ctx context.Context, subscriberID int64, status string) error {
	_, err := r.db(ctx).Exec(ctx, `
UPDATE notification_subscribers SET status = $1, verified_at = CASE WHEN $1 = 'active' THEN now() ELSE verified_at END
WHERE id = $2;`, status, subscriberID)
	return err
}

func (r *Repository) DeleteSubscriber(ctx context.Context, subscriberID int64) error {
	_, err := r.db(ctx).Exec(ctx, `DELETE FROM notification_subscribers WHERE id = $1;`, subscriberID)
	return err
}

func (r *Repository) DeletePendingExpired(ctx context.Context) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `
DELETE FROM notification_subscribers WHERE status = 'pending' AND created_at < now() - interval '24 hours';`)
	return tag.RowsAffected(), err
}

func (r *Repository) ActiveSubscribers(ctx context.Context) ([]entity.NotificationSubscriber, error) {
	rows, err := r.db(ctx).Query(ctx, `
SELECT id, twitch_login, twitch_user_id, steam_url, steam_id, status, created_at, verified_at
FROM notification_subscribers WHERE status = 'active'
ORDER BY id;`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []entity.NotificationSubscriber
	for rows.Next() {
		var s entity.NotificationSubscriber
		if err := rows.Scan(&s.ID, &s.TwitchLogin, &s.TwitchUserID, &s.SteamURL, &s.SteamID,
			&s.Status, &s.CreatedAt, &s.VerifiedAt); err != nil {
			return nil, err
		}
		subs = append(subs, s)
	}
	return subs, rows.Err()
}

func (r *Repository) ActiveSubscriberChannels(ctx context.Context) ([]string, error) {
	rows, err := r.db(ctx).Query(ctx, `
SELECT twitch_login FROM notification_subscribers WHERE status = 'active'
UNION
SELECT twitch_login FROM notification_subscribers WHERE status = 'pending';`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var ch string
		if err := rows.Scan(&ch); err != nil {
			return nil, err
		}
		out = append(out, ch)
	}
	return out, rows.Err()
}

func (r *Repository) RecentNotification(ctx context.Context, subscriberID int64, detectedName string, cooldown time.Duration) (bool, error) {
	var exists bool
	err := r.db(ctx).QueryRow(ctx, `
SELECT EXISTS(
  SELECT 1 FROM notification_log
  WHERE subscriber_id = $1 AND detected_name = $2 AND sent_at > now() - $3::interval
);`, subscriberID, detectedName, formatInterval(cooldown)).Scan(&exists)
	return exists, err
}

func (r *Repository) LogNotification(ctx context.Context, subscriberID int64, detectedName string, score float64, snapshotID int64, sourceStreamerID string) error {
	_, err := r.db(ctx).Exec(ctx, `
INSERT INTO notification_log (subscriber_id, detected_name, match_score, snapshot_id, source_streamer_id)
VALUES ($1, $2, $3, $4, $5);`,
		subscriberID, detectedName, score, snapshotID, sourceStreamerID,
	)
	return err
}

func formatInterval(d time.Duration) string {
	minutes := int(d.Minutes())
	if minutes <= 0 {
		minutes = 1
	}
	return fmt.Sprintf("%d minutes", minutes)
}
