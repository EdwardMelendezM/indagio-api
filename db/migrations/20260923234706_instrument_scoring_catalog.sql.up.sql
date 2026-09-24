-- +migrate Up

-- Catálogo de ítems por instrumento y versión.
-- No se usa FK compuesta a instruments(id, version) a propósito:
-- instruments.version es un puntero a la versión "vigente" (mutable),
-- pero las filas de versiones anteriores en estas tablas catálogo deben
-- seguir existiendo y ser consultables para poder recalcular/auditar
-- aplicaciones (instrument_administrations) hechas con versiones viejas.
CREATE TABLE instrument_items (
                                  id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
                                  instrument_id       UUID NOT NULL REFERENCES instruments(id) ON DELETE CASCADE,
                                  instrument_version  INTEGER NOT NULL,
                                  question_key        VARCHAR(120) NOT NULL,
                                  subscale_key        VARCHAR(64) NULL,
                                  order_index         INTEGER NOT NULL DEFAULT 0,
                                  item_type           VARCHAR(32) NOT NULL DEFAULT 'likert',
                                  reverse_scored      BOOLEAN NOT NULL DEFAULT FALSE,
                                  weight              NUMERIC(10,4) NOT NULL DEFAULT 1,
                                  value_map           JSONB NOT NULL DEFAULT '{}',
                                  is_scored           BOOLEAN NOT NULL DEFAULT TRUE,
                                  created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                                  updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                                  UNIQUE (instrument_id, instrument_version, question_key)
);

CREATE TABLE instrument_subscales (
                                      id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
                                      instrument_id       UUID NOT NULL REFERENCES instruments(id) ON DELETE CASCADE,
                                      instrument_version  INTEGER NOT NULL,
                                      key                 VARCHAR(64) NOT NULL,
                                      name                VARCHAR(160) NOT NULL,
                                      description         TEXT NULL,
                                      created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                                      updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                                      UNIQUE (instrument_id, instrument_version, key)
);

-- subscale_key NULL = regla del puntaje total del instrumento.
CREATE TABLE instrument_scoring_rules (
                                          id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
                                          instrument_id       UUID NOT NULL REFERENCES instruments(id) ON DELETE CASCADE,
                                          instrument_version  INTEGER NOT NULL,
                                          subscale_key        VARCHAR(64) NULL,
                                          algorithm           VARCHAR(32) NOT NULL DEFAULT 'sum', -- sum | average | weighted_sum | lookup_table | custom
                                          min_items_required  INTEGER NULL,
                                          missing_strategy    VARCHAR(32) NOT NULL DEFAULT 'prorate', -- prorate | zero | null_if_missing
                                          config              JSONB NOT NULL DEFAULT '{}',
                                          created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                                          updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                                          UNIQUE (instrument_id, instrument_version, subscale_key)
);

CREATE TABLE instrument_score_bands (
                                        id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
                                        instrument_id       UUID NOT NULL REFERENCES instruments(id) ON DELETE CASCADE,
                                        instrument_version  INTEGER NOT NULL,
                                        subscale_key        VARCHAR(64) NULL,
                                        min_score           NUMERIC NULL,
                                        max_score           NUMERIC NULL,
                                        label               VARCHAR(120) NOT NULL,
                                        description         TEXT NULL,
                                        created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_instrument_items_instrument_version ON instrument_items (instrument_id, instrument_version);
CREATE INDEX idx_instrument_items_subscale ON instrument_items (instrument_id, instrument_version, subscale_key);
CREATE INDEX idx_instrument_subscales_instrument_version ON instrument_subscales (instrument_id, instrument_version);
CREATE INDEX idx_instrument_scoring_rules_instrument_version ON instrument_scoring_rules (instrument_id, instrument_version);
CREATE INDEX idx_instrument_score_bands_instrument_version ON instrument_score_bands (instrument_id, instrument_version, subscale_key);