-- +migrate Down

DROP INDEX IF EXISTS idx_instrument_scores_administration;
DROP INDEX IF EXISTS idx_answer_records_administration;
DROP INDEX IF EXISTS idx_instrument_administrations_client_generated_id;
DROP INDEX IF EXISTS idx_instrument_administrations_instrument_status;
DROP INDEX IF EXISTS idx_instrument_administrations_project_participant;

DROP TABLE IF EXISTS instrument_scores;

ALTER TABLE answer_records DROP CONSTRAINT IF EXISTS answer_records_administration_id_fkey;
ALTER TABLE answer_records DROP COLUMN IF EXISTS administration_id;

DROP TABLE IF EXISTS instrument_administrations;
