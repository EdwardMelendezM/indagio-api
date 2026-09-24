package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"indagio-api/internal/domain"

	"github.com/google/uuid"
)

type postgresAnswerRepository struct {
	db *sql.DB
}

func NewAnswerRepository(db *sql.DB) domain.AnswerRepository {
	return &postgresAnswerRepository{db: db}
}

func (r *postgresAnswerRepository) GetInstrumentByID(ctx context.Context, instrumentID uuid.UUID) (*domain.Instrument, error) {
	var instrument domain.Instrument
	err := r.db.QueryRowContext(ctx, `
		SELECT id, project_id, name, kind, config, version, status, created_by, created_at, updated_at
		FROM instruments
		WHERE id = $1
	`, instrumentID).Scan(
		&instrument.ID,
		&instrument.ProjectID,
		&instrument.Name,
		&instrument.Kind,
		&instrument.Config,
		&instrument.Version,
		&instrument.Status,
		&instrument.CreatedBy,
		&instrument.CreatedAt,
		&instrument.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("instrument %s: %w", instrumentID, domain.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("query instrument: %w", err)
	}
	return &instrument, nil
}

func scanAnswerRow(answer *domain.AnswerRecord, instrumentID *sql.NullString, administrationID *sql.NullString, clientGeneratedID *sql.NullString, value *[]byte) error {
	if instrumentID.Valid {
		parsed, err := uuid.Parse(instrumentID.String)
		if err == nil {
			answer.InstrumentID = &parsed
		}
	}
	if administrationID.Valid {
		parsed, err := uuid.Parse(administrationID.String)
		if err == nil {
			answer.AdministrationID = &parsed
		}
	}
	if clientGeneratedID.Valid {
		answer.ClientGeneratedID = &clientGeneratedID.String
	}
	answer.Value = json.RawMessage(*value)
	return nil
}

func (r *postgresAnswerRepository) CreateAnswer(ctx context.Context, projectID, participantID uuid.UUID, instrumentID *uuid.UUID, administrationID uuid.UUID, questionKey string, answerType domain.AnswerType, value json.RawMessage, clientGeneratedID *string) (*domain.AnswerRecord, error) {
	answer := &domain.AnswerRecord{
		ID:                uuid.New(),
		ProjectID:         projectID,
		ParticipantID:     participantID,
		InstrumentID:      instrumentID,
		AdministrationID:  &administrationID,
		QuestionKey:       questionKey,
		AnswerType:        answerType,
		Value:             value,
		Status:            domain.AnswerStatusDraft,
		SyncStatus:        domain.SyncStatusPending,
		ClientGeneratedID: clientGeneratedID,
		CreatedAt:         time.Now().UTC(),
		UpdatedAt:         time.Now().UTC(),
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO answer_records (id, project_id, participant_id, instrument_id, administration_id, question_key, answer_type, value, status, sync_status, client_generated_id, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
	`, answer.ID, answer.ProjectID, answer.ParticipantID, answer.InstrumentID, answer.AdministrationID, answer.QuestionKey, answer.AnswerType, answer.Value, answer.Status, answer.SyncStatus, answer.ClientGeneratedID, answer.CreatedAt, answer.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert answer: %w", err)
	}
	return answer, nil
}

func (r *postgresAnswerRepository) GetAnswerByID(ctx context.Context, answerID uuid.UUID) (*domain.AnswerRecord, error) {
	var answer domain.AnswerRecord
	var instrumentID sql.NullString
	var administrationID sql.NullString
	var clientGeneratedID sql.NullString
	var value []byte
	err := r.db.QueryRowContext(ctx, `
		SELECT id, project_id, participant_id, instrument_id, administration_id, question_key, answer_type, value, status, sync_status, client_generated_id, created_at, updated_at 
		FROM answer_records WHERE id = $1
	`, answerID).Scan(&answer.ID, &answer.ProjectID, &answer.ParticipantID, &instrumentID, &administrationID, &answer.QuestionKey, &answer.AnswerType, &value, &answer.Status, &answer.SyncStatus, &clientGeneratedID, &answer.CreatedAt, &answer.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("answer %s: %w", answerID, domain.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("query answer: %w", err)
	}
	_ = scanAnswerRow(&answer, &instrumentID, &administrationID, &clientGeneratedID, &value)
	return &answer, nil
}

func (r *postgresAnswerRepository) ListAnswersByParticipant(ctx context.Context, participantID uuid.UUID, instrumentID *uuid.UUID) ([]domain.AnswerRecord, error) {
	query := `
		SELECT id, project_id, participant_id, instrument_id, administration_id, question_key, answer_type, value, status, sync_status, client_generated_id, created_at, updated_at
		FROM answer_records WHERE participant_id = $1`
	args := []any{participantID}
	if instrumentID != nil {
		query += " AND instrument_id = $2"
		args = append(args, *instrumentID)
	}
	query += " ORDER BY created_at DESC"
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list answers: %w", err)
	}
	defer rows.Close()
	answers := make([]domain.AnswerRecord, 0)
	for rows.Next() {
		var answer domain.AnswerRecord
		var instrumentID sql.NullString
		var administrationID sql.NullString
		var clientGeneratedID sql.NullString
		var value []byte
		if err := rows.Scan(&answer.ID, &answer.ProjectID, &answer.ParticipantID, &instrumentID, &administrationID, &answer.QuestionKey, &answer.AnswerType, &value, &answer.Status, &answer.SyncStatus, &clientGeneratedID, &answer.CreatedAt, &answer.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan answer: %w", err)
		}
		_ = scanAnswerRow(&answer, &instrumentID, &administrationID, &clientGeneratedID, &value)
		answers = append(answers, answer)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate answers: %w", err)
	}
	return answers, nil
}

// ListAnswersByAdministration is used by the scoring engine to pull every
// answer collected for a single instrument attempt.
func (r *postgresAnswerRepository) ListAnswersByAdministration(ctx context.Context, administrationID uuid.UUID) ([]domain.AnswerRecord, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, project_id, participant_id, instrument_id, administration_id, question_key, answer_type, value, status, sync_status, client_generated_id, created_at, updated_at
		FROM answer_records WHERE administration_id = $1 ORDER BY created_at
	`, administrationID)
	if err != nil {
		return nil, fmt.Errorf("list answers by administration: %w", err)
	}
	defer rows.Close()
	answers := make([]domain.AnswerRecord, 0)
	for rows.Next() {
		var answer domain.AnswerRecord
		var instrumentID sql.NullString
		var administrationIDCol sql.NullString
		var clientGeneratedID sql.NullString
		var value []byte
		if err := rows.Scan(&answer.ID, &answer.ProjectID, &answer.ParticipantID, &instrumentID, &administrationIDCol, &answer.QuestionKey, &answer.AnswerType, &value, &answer.Status, &answer.SyncStatus, &clientGeneratedID, &answer.CreatedAt, &answer.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan answer: %w", err)
		}
		_ = scanAnswerRow(&answer, &instrumentID, &administrationIDCol, &clientGeneratedID, &value)
		answers = append(answers, answer)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate answers: %w", err)
	}
	return answers, nil
}

func (r *postgresAnswerRepository) UpsertAnswer(ctx context.Context, projectID, participantID uuid.UUID, answerType domain.AnswerType, questionKey string, value json.RawMessage, clientGeneratedID string) (*domain.AnswerRecord, error) {
	return nil, fmt.Errorf("not implemented: %w", domain.ErrValidation)
}

func (r *postgresAnswerRepository) CreateMedia(ctx context.Context, projectID, participantID uuid.UUID, answerID *uuid.UUID, fileKey, mimeType string, sizeBytes int64, durationSeconds *int, checksum *string) (*domain.MediaFile, error) {
	media := &domain.MediaFile{
		ID:              uuid.New(),
		ProjectID:       projectID,
		ParticipantID:   participantID,
		AnswerID:        answerID,
		FileKey:         fileKey,
		MimeType:        mimeType,
		SizeBytes:       sizeBytes,
		DurationSeconds: durationSeconds,
		Checksum:        checksum,
		Status:          "pending",
		CreatedAt:       time.Now().UTC(),
		UpdatedAt:       time.Now().UTC(),
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO media_files (id, project_id, participant_id, answer_id, file_key, mime_type, size_bytes, duration_seconds, checksum, status, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
	`, media.ID, media.ProjectID, media.ParticipantID, media.AnswerID, media.FileKey, media.MimeType, media.SizeBytes, media.DurationSeconds, media.Checksum, media.Status, media.CreatedAt, media.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert media: %w", err)
	}
	return media, nil
}

func (r *postgresAnswerRepository) ListMediaByAnswer(ctx context.Context, answerID uuid.UUID) ([]domain.MediaFile, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, project_id, participant_id, answer_id, file_key, mime_type, size_bytes, duration_seconds, checksum, status, created_at, updated_at
		FROM media_files WHERE answer_id = $1 ORDER BY created_at DESC
	`, answerID)
	if err != nil {
		return nil, fmt.Errorf("list media: %w", err)
	}
	defer rows.Close()
	mediaFiles := make([]domain.MediaFile, 0)
	for rows.Next() {
		var m domain.MediaFile
		var answerID sql.NullString
		var durationSeconds sql.NullInt64
		var checksum sql.NullString
		if err := rows.Scan(&m.ID, &m.ProjectID, &m.ParticipantID, &answerID, &m.FileKey, &m.MimeType, &m.SizeBytes, &durationSeconds, &checksum, &m.Status, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan media: %w", err)
		}
		if answerID.Valid {
			parsed, err := uuid.Parse(answerID.String)
			if err == nil {
				m.AnswerID = &parsed
			}
		}
		if durationSeconds.Valid {
			d := int(durationSeconds.Int64)
			m.DurationSeconds = &d
		}
		if checksum.Valid {
			m.Checksum = &checksum.String
		}
		mediaFiles = append(mediaFiles, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate media: %w", err)
	}
	return mediaFiles, nil
}
