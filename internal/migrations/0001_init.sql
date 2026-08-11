-- 0001_init: initial schema for the DBD live stream history tracker.

CREATE TABLE IF NOT EXISTS streamers (
  twitch_user_id    TEXT PRIMARY KEY,
  login             TEXT NOT NULL,
  display_name      TEXT NOT NULL,
  profile_image_url TEXT NOT NULL DEFAULT '',
  last_seen         TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS snapshots (
  id                 BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  taken_at           TIMESTAMPTZ NOT NULL,
  source             TEXT NOT NULL DEFAULT 'twitch',
  stream_count       INT NOT NULL,
  duration_seconds   DOUBLE PRECISION NOT NULL DEFAULT 0,
  disk_usage_bytes   BIGINT NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS stream_sessions (
  id               BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  streamer_id      TEXT NOT NULL REFERENCES streamers(twitch_user_id) ON DELETE CASCADE,
  twitch_stream_id TEXT NOT NULL UNIQUE,
  started_at       TIMESTAMPTZ NOT NULL,
  ended_at         TIMESTAMPTZ,
  vod_id           TEXT,
  vod_created_at   TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS stream_samples (
  snapshot_id        BIGINT NOT NULL REFERENCES snapshots(id) ON DELETE CASCADE,
  session_id         BIGINT NOT NULL REFERENCES stream_sessions(id) ON DELETE CASCADE,
  streamer_id        TEXT NOT NULL REFERENCES streamers(twitch_user_id) ON DELETE CASCADE,
  viewer_count       INT NOT NULL,
  title              TEXT NOT NULL,
  language           TEXT NOT NULL DEFAULT '',
  tags               TEXT[] NOT NULL DEFAULT '{}',
  started_at         TIMESTAMPTZ NOT NULL,
  vod_offset_seconds INT,
  preview_filename   TEXT,
  thumb_filename     TEXT,
  survivor_names     TEXT[] NOT NULL DEFAULT '{}',
  PRIMARY KEY (snapshot_id, session_id)
);

CREATE TABLE IF NOT EXISTS notification_subscribers (
    id              BIGSERIAL PRIMARY KEY,
    twitch_login    TEXT NOT NULL UNIQUE,
    twitch_user_id  TEXT NOT NULL,
    steam_url       TEXT NOT NULL,
    steam_id        TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'pending',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    verified_at     TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS notification_log (
    id                 BIGSERIAL PRIMARY KEY,
    subscriber_id      BIGINT NOT NULL REFERENCES notification_subscribers(id) ON DELETE CASCADE,
    detected_name      TEXT NOT NULL,
    match_score        REAL NOT NULL,
    snapshot_id        BIGINT NOT NULL REFERENCES snapshots(id) ON DELETE CASCADE,
    source_streamer_id TEXT NOT NULL,
    sent_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_snapshots_taken_at   ON snapshots(taken_at);
CREATE INDEX IF NOT EXISTS idx_samples_snapshot     ON stream_samples(snapshot_id);
CREATE INDEX IF NOT EXISTS idx_samples_streamer     ON stream_samples(streamer_id);
CREATE INDEX IF NOT EXISTS idx_samples_survivors    ON stream_samples USING GIN (survivor_names);
CREATE INDEX IF NOT EXISTS idx_sessions_streamer_st ON stream_sessions(streamer_id, started_at);
CREATE INDEX IF NOT EXISTS idx_sessions_ended       ON stream_sessions(ended_at);
CREATE INDEX IF NOT EXISTS idx_streamers_last_seen  ON streamers(last_seen);

CREATE INDEX IF NOT EXISTS idx_notif_dedup       ON notification_log(subscriber_id, detected_name, sent_at);
CREATE INDEX IF NOT EXISTS idx_notif_log_snapshot ON notification_log(snapshot_id);
CREATE INDEX IF NOT EXISTS idx_notif_log_source   ON notification_log(source_streamer_id);
CREATE INDEX IF NOT EXISTS idx_notif_sub_pending  ON notification_subscribers(created_at) WHERE status = 'pending';
CREATE INDEX IF NOT EXISTS idx_notif_sub_active   ON notification_subscribers(status) WHERE status = 'active';
