CREATE TABLE IF NOT EXISTS notification_subscribers (
    id              BIGSERIAL PRIMARY KEY,
    twitch_login    TEXT NOT NULL UNIQUE,
    twitch_user_id  TEXT NOT NULL,
    steam_url       TEXT NOT NULL,
    steam_id        TEXT NOT NULL,
    steam_name      TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'pending',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    verified_at     TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS subscriber_names (
    id              BIGSERIAL PRIMARY KEY,
    subscriber_id   BIGINT NOT NULL REFERENCES notification_subscribers(id) ON DELETE CASCADE,
    in_game_name    TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(subscriber_id, in_game_name)
);

CREATE TABLE IF NOT EXISTS notification_log (
    id                 BIGSERIAL PRIMARY KEY,
    subscriber_id      BIGINT NOT NULL REFERENCES notification_subscribers(id),
    detected_name      TEXT NOT NULL,
    match_score        REAL NOT NULL,
    snapshot_id        BIGINT NOT NULL REFERENCES snapshots(id),
    source_streamer_id TEXT NOT NULL,
    sent_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_notif_dedup ON notification_log(subscriber_id, detected_name, sent_at);
CREATE INDEX IF NOT EXISTS idx_notif_sub_pending ON notification_subscribers(created_at) WHERE status = 'pending';
