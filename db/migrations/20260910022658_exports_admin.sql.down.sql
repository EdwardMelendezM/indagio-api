-- +migrate Down
DROP INDEX IF EXISTS idx_project_exports_created_by;
DROP INDEX IF EXISTS idx_project_exports_project_status;
DROP TABLE IF EXISTS project_exports;
