
-- +migrate Down
DROP INDEX IF EXISTS idx_media_files_project_status;
DROP INDEX IF EXISTS idx_answer_records_client_generated_id;
DROP INDEX IF EXISTS idx_answer_records_project_participant;

DROP TABLE IF EXISTS media_files;
DROP TABLE IF EXISTS answer_records;
