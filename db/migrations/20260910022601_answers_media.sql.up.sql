-- +migrate Up
CREATE TABLE answer_records (
                                id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
                                project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
                                participant_id UUID NOT NULL REFERENCES participants(id) ON DELETE CASCADE,
                                instrument_id UUID NULL REFERENCES instruments(id) ON DELETE SET NULL,
                                questionnaire_id UUID NULL REFERENCES questionnaires(id) ON DELETE SET NULL,
                                question_key VARCHAR(120) NOT NULL,
                                answer_type VARCHAR(32) NOT NULL DEFAULT 'text',
                                value JSONB NOT NULL DEFAULT '{}',
                                status VARCHAR(32) NOT NULL DEFAULT 'draft',
                                sync_status VARCHAR(32) NOT NULL DEFAULT 'pending',
                                client_generated_id VARCHAR(128) NULL,
                                created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                                updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE media_files (
                             id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
                             project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
                             participant_id UUID NOT NULL REFERENCES participants(id) ON DELETE CASCADE,
                             answer_id UUID NULL REFERENCES answer_records(id) ON DELETE SET NULL,
                             file_key VARCHAR(255) NOT NULL,
                             mime_type VARCHAR(80) NOT NULL,
                             size_bytes BIGINT NOT NULL DEFAULT 0,
                             duration_seconds INTEGER NULL,
                             checksum VARCHAR(128) NULL,
                             status VARCHAR(32) NOT NULL DEFAULT 'pending',
                             created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                             updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_answer_records_project_participant ON answer_records (project_id, participant_id, created_at DESC);
CREATE INDEX idx_answer_records_client_generated_id ON answer_records (client_generated_id);
CREATE INDEX idx_media_files_project_status ON media_files (project_id, status, created_at DESC);
