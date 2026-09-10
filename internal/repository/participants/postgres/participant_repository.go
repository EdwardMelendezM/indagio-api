package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"indagio-api/internal/domain"

	"github.com/google/uuid"
)

type postgresParticipantRepository struct {
	db *sql.DB
}

func NewParticipantRepository(db *sql.DB) domain.ParticipantRepository {
	return &postgresParticipantRepository{db: db}
}

func (r *postgresParticipantRepository) Create(ctx context.Context, projectID uuid.UUID, code, displayName, externalIdentifier string) (*domain.Participant, error) {
	participant := &domain.Participant{
		ID:                 uuid.New(),
		ProjectID:          projectID,
		Code:               code,
		DisplayName:        displayName,
		ExternalIdentifier: externalIdentifier,
		Status:             domain.ParticipantStatusPending,
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO participants (id, project_id, code, display_name, external_identifier, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, participant.ID, participant.ProjectID, participant.Code, participant.DisplayName, participant.ExternalIdentifier, participant.Status, participant.CreatedAt, participant.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert participant: %w", err)
	}
	return participant, nil
}

func (r *postgresParticipantRepository) GetByID(ctx context.Context, participantID uuid.UUID) (*domain.Participant, error) {
	var p domain.Participant
	var externalIdentifier sql.NullString
	var lastAccessAt sql.NullTime
	err := r.db.QueryRowContext(ctx, `
		SELECT id, project_id, code, display_name, external_identifier, status, created_at, updated_at, last_access_at
		FROM participants
		WHERE id = $1
	`, participantID).Scan(
		&p.ID,
		&p.ProjectID,
		&p.Code,
		&p.DisplayName,
		&externalIdentifier,
		&p.Status,
		&p.CreatedAt,
		&p.UpdatedAt,
		&lastAccessAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("participant %s: %w", participantID, domain.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("query participant: %w", err)
	}
	if externalIdentifier.Valid {
		p.ExternalIdentifier = externalIdentifier.String
	}
	if lastAccessAt.Valid {
		p.LastAccessAt = &lastAccessAt.Time
	}
	return &p, nil
}

func (r *postgresParticipantRepository) GetByCode(ctx context.Context, projectID uuid.UUID, code string) (*domain.Participant, error) {
	var p domain.Participant
	var externalIdentifier sql.NullString
	var lastAccessAt sql.NullTime
	err := r.db.QueryRowContext(ctx, `
		SELECT id, project_id, code, display_name, external_identifier, status, created_at, updated_at, last_access_at
		FROM participants
		WHERE project_id = $1 AND code = $2
	`, projectID, code).Scan(
		&p.ID,
		&p.ProjectID,
		&p.Code,
		&p.DisplayName,
		&externalIdentifier,
		&p.Status,
		&p.CreatedAt,
		&p.UpdatedAt,
		&lastAccessAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("participant code %s: %w", code, domain.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("query participant by code: %w", err)
	}
	if externalIdentifier.Valid {
		p.ExternalIdentifier = externalIdentifier.String
	}
	if lastAccessAt.Valid {
		p.LastAccessAt = &lastAccessAt.Time
	}
	return &p, nil
}

func (r *postgresParticipantRepository) ListByProject(ctx context.Context, projectID uuid.UUID) ([]domain.Participant, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, project_id, code, display_name, external_identifier, status, created_at, updated_at, last_access_at
		FROM participants
		WHERE project_id = $1
		ORDER BY created_at DESC
	`, projectID)
	if err != nil {
		return nil, fmt.Errorf("list participants: %w", err)
	}
	defer rows.Close()

	participants := make([]domain.Participant, 0)
	for rows.Next() {
		var p domain.Participant
		var externalIdentifier sql.NullString
		var lastAccessAt sql.NullTime
		if err := rows.Scan(
			&p.ID,
			&p.ProjectID,
			&p.Code,
			&p.DisplayName,
			&externalIdentifier,
			&p.Status,
			&p.CreatedAt,
			&p.UpdatedAt,
			&lastAccessAt,
		); err != nil {
			return nil, fmt.Errorf("scan participant: %w", err)
		}
		if externalIdentifier.Valid {
			p.ExternalIdentifier = externalIdentifier.String
		}
		if lastAccessAt.Valid {
			p.LastAccessAt = &lastAccessAt.Time
		}
		participants = append(participants, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate participants: %w", err)
	}
	return participants, nil
}

func (r *postgresParticipantRepository) UpdateStatus(ctx context.Context, participantID uuid.UUID, status domain.ParticipantStatus) (*domain.Participant, error) {
	result, err := r.db.ExecContext(ctx, `
		UPDATE participants
		SET status = $2, updated_at = NOW()
		WHERE id = $1
	`, participantID, status)
	if err != nil {
		return nil, fmt.Errorf("update participant status: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("rows affected: %w", err)
	}
	if rows == 0 {
		return nil, fmt.Errorf("participant %s: %w", participantID, domain.ErrNotFound)
	}
	return r.GetByID(ctx, participantID)
}
