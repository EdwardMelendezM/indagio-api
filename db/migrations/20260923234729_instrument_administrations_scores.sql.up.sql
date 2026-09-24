-- +migrate Up

-- Agrupa las respuestas (answer_records) de un participante a un instrumento
-- en un intento/aplicación específico. Sigue el mismo patrón offline que
-- answer_records/sync_batches: el cliente puede generar el intento antes de
-- sincronizar, de ahí client_generated_id + UNIQUE compuesto para dedupe.
CREATE TABLE instrument_administrations (
                                            id                   UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
                                            project_id           UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
                                            participant_id       UUID NOT NULL REFERENCES participants(id) ON DELETE CASCADE,
                                            instrument_id        UUID NOT NULL REFERENCES instruments(id) ON DELETE CASCADE,
                                            instrument_version   INTEGER NOT NULL,
                                            status               VARCHAR(32) NOT NULL DEFAULT 'in_progress', -- in_progress | completed | abandoned
                                            sync_status          VARCHAR(32) NOT NULL DEFAULT 'pending',
                                            client_generated_id  VARCHAR(128) NULL,
                                            started_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                                            completed_at         TIMESTAMPTZ NULL,
                                            created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                                            updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                                            UNIQUE (project_id, participant_id, client_generated_id)
);

-- Liga cada respuesta a su intento específico. Nullable porque
-- answer_records.instrument_id ya es nullable (compatibilidad con datos
-- existentes sin administración asignada).
ALTER TABLE answer_records
    ADD COLUMN administration_id UUID NULL REFERENCES instrument_administrations(id) ON DELETE SET NULL;

-- Resultado calculado y persistido de una aplicación (total y por subescala).
-- Se guarda en vez de calcularse al vuelo para poder auditar con qué
-- versión del algoritmo se obtuvo, y para no recalcular en cada lectura.
CREATE TABLE instrument_scores (
                                   id                   UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
                                   administration_id    UUID NOT NULL REFERENCES instrument_administrations(id) ON DELETE CASCADE,
                                   subscale_key         VARCHAR(64) NULL, -- NULL = puntaje total
                                   raw_score            NUMERIC NULL,
                                   scaled_score         NUMERIC NULL,
                                   items_answered       INTEGER NOT NULL DEFAULT 0,
                                   items_expected       INTEGER NOT NULL DEFAULT 0,
                                   band_id              UUID NULL REFERENCES instrument_score_bands(id) ON DELETE SET NULL,
                                   computation_version  VARCHAR(32) NULL,
                                   computed_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                                   metadata             JSONB NOT NULL DEFAULT '{}',
                                   UNIQUE (administration_id, subscale_key)
);

CREATE INDEX idx_instrument_administrations_project_participant ON instrument_administrations (project_id, participant_id, created_at DESC);
CREATE INDEX idx_instrument_administrations_instrument_status ON instrument_administrations (instrument_id, status, created_at DESC);
CREATE INDEX idx_instrument_administrations_client_generated_id ON instrument_administrations (client_generated_id);
CREATE INDEX idx_answer_records_administration ON answer_records (administration_id);
CREATE INDEX idx_instrument_scores_administration ON instrument_scores (administration_id);