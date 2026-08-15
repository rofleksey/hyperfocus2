-- 0002_steam_name: store the latest resolved Steam persona name per subscriber
-- so it can be used as a fallback when the Steam API is unreachable.

ALTER TABLE notification_subscribers
    ADD COLUMN IF NOT EXISTS steam_name TEXT NOT NULL DEFAULT '';
