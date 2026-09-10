-- +migrate Up
CREATE TABLE instruments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name VARCHAR(160) NOT NULL,
    kind VARCHAR(64) NOT NULL,
    config JSONB NOT NULL DEFAULT '{}',
    version INTEGER NOT NULL DEFAULT 1,
    status VARCHAR(32) NOT NULL DEFAULT 'draft',
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE questionnaires (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name VARCHAR(160) NOT NULL,
    version INTEGER NOT NULL DEFAULT 1,
    status VARCHAR(32) NOT NULL DEFAULT 'draft',
    schema JSONB NOT NULL DEFAULT '{}',
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_instruments_project_status ON instruments (project_id, status, created_at DESC);
CREATE INDEX idx_instruments_kind ON instruments (kind);
CREATE INDEX idx_questionnaires_project_status ON questionnaires (project_id, status, created_at DESC);

-- +migrate Down
DROP INDEX IF EXISTS idx_questionnaires_project_status;
DROP INDEX IF EXISTS idx_instruments_kind;
DROP INDEX IF EXISTS idx_instruments_project_status;

DROP TABLE IF EXISTS questionnaires;
DROP TABLE IF EXISTS instruments;
