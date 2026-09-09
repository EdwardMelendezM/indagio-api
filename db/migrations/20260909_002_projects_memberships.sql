-- +migrate Up
CREATE TABLE projects (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    owner_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(160) NOT NULL,
    description TEXT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'draft',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    archived_at TIMESTAMPTZ NULL,
    deleted_at TIMESTAMPTZ NULL
);

CREATE TABLE project_members (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(32) NOT NULL DEFAULT 'member',
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    invited_by UUID NULL REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(project_id, user_id)
);

CREATE TABLE project_invitations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    email TEXT NOT NULL,
    role VARCHAR(32) NOT NULL DEFAULT 'member',
    invited_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_projects_owner_status ON projects (owner_id, status, created_at DESC);
CREATE INDEX idx_projects_deleted_at ON projects (deleted_at);
CREATE INDEX idx_project_members_project_user ON project_members (project_id, user_id);
CREATE INDEX idx_project_members_user_role ON project_members (user_id, role);
CREATE INDEX idx_project_invitations_project_status ON project_invitations (project_id, status, expires_at);
CREATE INDEX idx_project_invitations_email_status ON project_invitations (email, status, expires_at);

-- +migrate Down
DROP INDEX IF EXISTS idx_project_invitations_email_status;
DROP INDEX IF EXISTS idx_project_invitations_project_status;
DROP INDEX IF EXISTS idx_project_members_user_role;
DROP INDEX IF EXISTS idx_project_members_project_user;
DROP INDEX IF EXISTS idx_projects_deleted_at;
DROP INDEX IF EXISTS idx_projects_owner_status;

DROP TABLE IF EXISTS project_invitations;
DROP TABLE IF EXISTS project_members;
DROP TABLE IF EXISTS projects;
