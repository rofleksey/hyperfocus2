ALTER TABLE notification_log
  DROP CONSTRAINT IF EXISTS notification_log_subscriber_id_fkey,
  ADD CONSTRAINT notification_log_subscriber_id_fkey
    FOREIGN KEY (subscriber_id) REFERENCES notification_subscribers(id) ON DELETE CASCADE;

ALTER TABLE notification_log
  DROP CONSTRAINT IF EXISTS notification_log_snapshot_id_fkey,
  ADD CONSTRAINT notification_log_snapshot_id_fkey
    FOREIGN KEY (snapshot_id) REFERENCES snapshots(id) ON DELETE CASCADE;
