
-- +migrate Down
DROP INDEX IF EXISTS idx_sync_events_batch;
DROP INDEX IF EXISTS idx_sync_batches_project_participant_status;
DROP TABLE IF EXISTS sync_events;
DROP TABLE IF EXISTS sync_batches;
