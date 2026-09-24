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

type postgresInstrumentAdministrationRepository struct {
	db *sql.DB
}

func NewInstrumentAdministrationRepository(db *sql.DB) domain.InstrumentAdministrationRepository {
	return &postgresInstrumentAdministrationRepository{db: db}
}

func (r *postgresInstrumentAdministrationRepository) CreateAdministration(ctx context.Context, projectID, participantID, instrumentID uuid.UUID, instrumentVersion int, clientGeneratedID *string) (*domain.InstrumentAdministration, error) {
	admin := &domain.InstrumentAdministration{
		ID:                uuid.New(),
		ProjectID:         projectID,
		ParticipantID:     participantID,
		InstrumentID:      instrumentID,
		InstrumentVersion: instrumentVersion,
		Status:            domain.AdministrationStatusInProgress,
		SyncStatus:        domain.SyncStatusPending,
		ClientGeneratedID: clientGeneratedID,
		StartedAt:         time.Now().UTC(),
		CreatedAt:         time.Now().UTC(),
		UpdatedAt:         time.Now().UTC(),
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO instrument_administrations (id, project_id, participant_id, instrument_id, instrument_version, status, sync_status, client_generated_id, started_at, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
	`, admin.ID, admin.ProjectID, admin.ParticipantID, admin.InstrumentID, admin.InstrumentVersion, admin.Status, admin.SyncStatus, admin.ClientGeneratedID, admin.StartedAt, admin.CreatedAt, admin.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert administration: %w", err)
	}
	return admin, nil
}

func (r *postgresInstrumentAdministrationRepository) GetAdministrationByID(ctx context.Context, administrationID uuid.UUID) (*domain.InstrumentAdministration, error) {
	var a domain.InstrumentAdministration
	var clientGeneratedID sql.NullString
	var completedAt sql.NullTime
	err := r.db.QueryRowContext(ctx, `
		SELECT id, project_id, participant_id, instrument_id, instrument_version, status, sync_status, client_generated_id, started_at, completed_at, created_at, updated_at
		FROM instrument_administrations WHERE id = $1
	`, administrationID).Scan(&a.ID, &a.ProjectID, &a.ParticipantID, &a.InstrumentID, &a.InstrumentVersion, &a.Status, &a.SyncStatus, &clientGeneratedID, &a.StartedAt, &completedAt, &a.CreatedAt, &a.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("administration %s: %w", administrationID, domain.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("query administration: %w", err)
	}
	if clientGeneratedID.Valid {
		a.ClientGeneratedID = &clientGeneratedID.String
	}
	if completedAt.Valid {
		a.CompletedAt = &completedAt.Time
	}
	return &a, nil
}

func (r *postgresInstrumentAdministrationRepository) ListAdministrationsByParticipant(ctx context.Context, participantID, instrumentID uuid.UUID) ([]domain.InstrumentAdministration, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, project_id, participant_id, instrument_id, instrument_version, status, sync_status, client_generated_id, started_at, completed_at, created_at, updated_at
		FROM instrument_administrations WHERE participant_id = $1 AND instrument_id = $2 ORDER BY created_at DESC
	`, participantID, instrumentID)
	if err != nil {
		return nil, fmt.Errorf("list administrations: %w", err)
	}
	defer rows.Close()
	items := make([]domain.InstrumentAdministration, 0)
	for rows.Next() {
		var a domain.InstrumentAdministration
		var clientGeneratedID sql.NullString
		var completedAt sql.NullTime
		if err := rows.Scan(&a.ID, &a.ProjectID, &a.ParticipantID, &a.InstrumentID, &a.InstrumentVersion, &a.Status, &a.SyncStatus, &clientGeneratedID, &a.StartedAt, &completedAt, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan administration: %w", err)
		}
		if clientGeneratedID.Valid {
			a.ClientGeneratedID = &clientGeneratedID.String
		}
		if completedAt.Valid {
			a.CompletedAt = &completedAt.Time
		}
		items = append(items, a)
	}
	return items, rows.Err()
}

func (r *postgresInstrumentAdministrationRepository) UpdateStatus(ctx context.Context, administrationID uuid.UUID, status domain.AdministrationStatus, completedAt *time.Time) (*domain.InstrumentAdministration, error) {
	result, err := r.db.ExecContext(ctx, `
		UPDATE instrument_administrations SET status = $1, completed_at = $2, updated_at = NOW() WHERE id = $3
	`, status, completedAt, administrationID)
	if err != nil {
		return nil, fmt.Errorf("update administration status: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return nil, fmt.Errorf("administration %s: %w", administrationID, domain.ErrNotFound)
	}
	return r.GetAdministrationByID(ctx, administrationID)
}

func (r *postgresInstrumentAdministrationRepository) SaveScores(ctx context.Context, administrationID uuid.UUID, scores []domain.InstrumentScore) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM instrument_scores WHERE administration_id = $1`, administrationID); err != nil {
		return fmt.Errorf("clear previous scores: %w", err)
	}
	for _, s := range scores {
		id := s.ID
		if id == uuid.Nil {
			id = uuid.New()
		}
		metadata := s.Metadata
		if len(metadata) == 0 {
			metadata = json.RawMessage("{}")
		}
		_, err := tx.ExecContext(ctx, `
			INSERT INTO instrument_scores (id, administration_id, subscale_key, raw_score, scaled_score, items_answered, items_expected, band_id, computation_version, computed_at, metadata)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		`, id, administrationID, s.SubscaleKey, s.RawScore, s.ScaledScore, s.ItemsAnswered, s.ItemsExpected, s.BandID, s.ComputationVersion, s.ComputedAt, []byte(metadata))
		if err != nil {
			return fmt.Errorf("insert score: %w", err)
		}
	}
	return tx.Commit()
}

func (r *postgresInstrumentAdministrationRepository) GetScores(ctx context.Context, administrationID uuid.UUID) ([]domain.InstrumentScore, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT s.id, s.administration_id, s.subscale_key, s.raw_score, s.scaled_score, s.items_answered, s.items_expected, s.band_id, b.label, s.computation_version, s.computed_at, s.metadata
		FROM instrument_scores s
		LEFT JOIN instrument_score_bands b ON b.id = s.band_id
		WHERE s.administration_id = $1
	`, administrationID)
	if err != nil {
		return nil, fmt.Errorf("list scores: %w", err)
	}
	defer rows.Close()
	scores := make([]domain.InstrumentScore, 0)
	for rows.Next() {
		var s domain.InstrumentScore
		var subscaleKey sql.NullString
		var rawScore, scaledScore sql.NullFloat64
		var bandID sql.NullString
		var bandLabel sql.NullString
		var computationVersion sql.NullString
		var metadata []byte
		if err := rows.Scan(&s.ID, &s.AdministrationID, &subscaleKey, &rawScore, &scaledScore, &s.ItemsAnswered, &s.ItemsExpected, &bandID, &bandLabel, &computationVersion, &s.ComputedAt, &metadata); err != nil {
			return nil, fmt.Errorf("scan score: %w", err)
		}
		if subscaleKey.Valid {
			s.SubscaleKey = &subscaleKey.String
		}
		if rawScore.Valid {
			s.RawScore = &rawScore.Float64
		}
		if scaledScore.Valid {
			s.ScaledScore = &scaledScore.Float64
		}
		if bandID.Valid {
			if parsed, err := uuid.Parse(bandID.String); err == nil {
				s.BandID = &parsed
			}
		}
		if bandLabel.Valid {
			s.BandLabel = &bandLabel.String
		}
		if computationVersion.Valid {
			s.ComputationVersion = &computationVersion.String
		}
		s.Metadata = json.RawMessage(metadata)
		scores = append(scores, s)
	}
	return scores, rows.Err()
}
