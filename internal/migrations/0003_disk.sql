-- 0003_disk: add disk_usage_bytes column to snapshots.
ALTER TABLE snapshots ADD COLUMN IF NOT EXISTS disk_usage_bytes BIGINT NOT NULL DEFAULT 0;
