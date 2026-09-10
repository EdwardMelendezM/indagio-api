-- +migrate Down
DROP INDEX IF EXISTS idx_participants_external_identifier;
DROP INDEX IF EXISTS idx_participants_code;
DROP INDEX IF EXISTS idx_participants_project_status;
DROP TABLE IF EXISTS participants;
