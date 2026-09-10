-- +migrate Down
DROP INDEX IF EXISTS idx_background_jobs_status_run_at;
DROP INDEX IF EXISTS idx_otps_email_active;
DROP INDEX IF EXISTS idx_users_deleted_at;
DROP INDEX IF EXISTS idx_users_email;

DROP TABLE IF EXISTS background_jobs;
DROP TABLE IF EXISTS user_adm;
DROP TABLE IF EXISTS otps;
DROP TABLE IF EXISTS users;

DROP TYPE IF EXISTS user_role;
DROP EXTENSION IF EXISTS "uuid-ossp";
