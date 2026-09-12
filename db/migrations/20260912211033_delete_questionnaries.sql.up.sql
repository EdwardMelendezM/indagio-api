-- +migrate Up
ALTER TABLE answer_records DROP CONSTRAINT IF EXISTS answer_records_questionnaire_id_fkey;
ALTER TABLE answer_records DROP COLUMN IF EXISTS questionnaire_id;

DROP INDEX IF EXISTS idx_questionnaires_project_status;
DROP TABLE IF EXISTS questionnaires;
