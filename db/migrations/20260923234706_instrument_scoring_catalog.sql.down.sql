-- +migrate Down

DROP INDEX IF EXISTS idx_instrument_score_bands_instrument_version;
DROP INDEX IF EXISTS idx_instrument_scoring_rules_instrument_version;
DROP INDEX IF EXISTS idx_instrument_subscales_instrument_version;
DROP INDEX IF EXISTS idx_instrument_items_subscale;
DROP INDEX IF EXISTS idx_instrument_items_instrument_version;

DROP TABLE IF EXISTS instrument_score_bands;
DROP TABLE IF EXISTS instrument_scoring_rules;
DROP TABLE IF EXISTS instrument_subscales;
DROP TABLE IF EXISTS instrument_items;