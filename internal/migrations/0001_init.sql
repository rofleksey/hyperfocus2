-- 0001_init: initial schema for the DBD live stream history tracker.

CREATE TABLE IF NOT EXISTS schema_version (
  version    INT PRIMARY KEY,
  applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS streamers (
  twitch_user_id    TEXT PRIMARY KEY,
  login             TEXT NOT NULL,
  display_name      TEXT NOT NULL,
  profile_image_url TEXT NOT NULL DEFAULT '',
  last_seen         TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS snapshots (
  id           BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  taken_at     TIMESTAMPTZ NOT NULL,
  source       TEXT NOT NULL DEFAULT 'twitch',
  stream_count INT NOT NULL
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
  PRIMARY KEY (snapshot_id, session_id)
);

CREATE TABLE IF NOT EXISTS vods (
  vod_id           TEXT PRIMARY KEY,
  streamer_id      TEXT NOT NULL REFERENCES streamers(twitch_user_id) ON DELETE CASCADE,
  stream_id        TEXT,
  started_at       TIMESTAMPTZ NOT NULL,
  duration_seconds INT,
  url              TEXT NOT NULL DEFAULT '',
  thumbnail_url    TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_snapshots_taken_at   ON snapshots(taken_at);
CREATE INDEX IF NOT EXISTS idx_samples_snapshot     ON stream_samples(snapshot_id);
CREATE INDEX IF NOT EXISTS idx_samples_streamer     ON stream_samples(streamer_id);
CREATE INDEX IF NOT EXISTS idx_sessions_streamer_st ON stream_sessions(streamer_id, started_at);
CREATE INDEX IF NOT EXISTS idx_sessions_ended       ON stream_sessions(ended_at);
CREATE INDEX IF NOT EXISTS idx_streamers_last_seen  ON streamers(last_seen);
CREATE INDEX IF NOT EXISTS idx_vods_started         ON vods(started_at);
