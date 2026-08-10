-- 0002_duration: add duration_seconds column to snapshots for cycle timing.
ALTER TABLE snapshots ADD COLUMN IF NOT EXISTS duration_seconds DOUBLE PRECISION NOT NULL DEFAULT 0;
