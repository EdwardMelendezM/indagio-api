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
