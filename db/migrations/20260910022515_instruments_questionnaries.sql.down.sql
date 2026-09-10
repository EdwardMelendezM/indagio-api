-- +migrate Down
DROP INDEX IF EXISTS idx_questionnaires_project_status;
DROP INDEX IF EXISTS idx_instruments_kind;
DROP INDEX IF EXISTS idx_instruments_project_status;

DROP TABLE IF EXISTS questionnaires;
DROP TABLE IF EXISTS instruments;
