-- +migrate Up
CREATE TABLE project_exports (
                                 id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
                                 project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
                                 created_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
                                 format VARCHAR(32) NOT NULL,
                                 status VARCHAR(32) NOT NULL DEFAULT 'pending',
                                 options JSONB NOT NULL DEFAULT '{}',
                                 file_key TEXT NULL,
                                 error_log TEXT NULL,
                                 started_at TIMESTAMPTZ NULL,
                                 completed_at TIMESTAMPTZ NULL,
                                 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                                 updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_project_exports_project_status ON project_exports (project_id, status, created_at DESC);
CREATE INDEX idx_project_exports_created_by ON project_exports (created_by, created_at DESC);
