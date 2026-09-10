-- +migrate Up
CREATE TABLE sync_batches (
                              id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
                              project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
                              participant_id UUID NOT NULL REFERENCES participants(id) ON DELETE CASCADE,
                              client_generated_batch_id VARCHAR(128) NOT NULL,
                              payload JSONB NOT NULL DEFAULT '{}',
                              status VARCHAR(32) NOT NULL DEFAULT 'pending',
                              created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                              updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                              UNIQUE(project_id, participant_id, client_generated_batch_id)
);

CREATE TABLE sync_events (
                             id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
                             batch_id UUID NOT NULL REFERENCES sync_batches(id) ON DELETE CASCADE,
                             event_type VARCHAR(64) NOT NULL,
                             entity_id VARCHAR(128) NOT NULL,
                             payload JSONB NOT NULL DEFAULT '{}',
                             created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_sync_batches_project_participant_status ON sync_batches (project_id, participant_id, status, created_at DESC);
CREATE INDEX idx_sync_events_batch ON sync_events (batch_id, created_at DESC);
