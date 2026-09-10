-- +migrate Up
CREATE TABLE participants (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    code VARCHAR(32) NOT NULL,
    display_name VARCHAR(160) NOT NULL,
    external_identifier VARCHAR(160) NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_access_at TIMESTAMPTZ NULL,
    UNIQUE(project_id, code),
    UNIQUE(project_id, external_identifier)
);

CREATE INDEX idx_participants_project_status ON participants (project_id, status, created_at DESC);
CREATE INDEX idx_participants_code ON participants (code);
CREATE INDEX idx_participants_external_identifier ON participants (project_id, external_identifier);

-- +migrate Down
DROP INDEX IF EXISTS idx_participants_external_identifier;
DROP INDEX IF EXISTS idx_participants_code;
DROP INDEX IF EXISTS idx_participants_project_status;
DROP TABLE IF EXISTS participants;
