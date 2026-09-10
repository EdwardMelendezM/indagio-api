package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"indagio-api/internal/domain"

	"github.com/google/uuid"
)

type postgresInstrumentRepository struct {
	db *sql.DB
}

func NewInstrumentRepository(db *sql.DB) domain.InstrumentRepository {
	return &postgresInstrumentRepository{db: db}
}

func (r *postgresInstrumentRepository) CreateInstrument(ctx context.Context, projectID, createdBy uuid.UUID, name string, kind domain.InstrumentKind, config json.RawMessage) (*domain.Instrument, error) {
	instrument := &domain.Instrument{
		ID:        uuid.New(),
		ProjectID: projectID,
		Name:      name,
		Kind:      kind,
		Config:    config,
		Version:   1,
		Status:    domain.InstrumentStatusDraft,
		CreatedBy: createdBy,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO instruments (id, project_id, name, kind, config, version, status, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, instrument.ID, instrument.ProjectID, instrument.Name, instrument.Kind, instrument.Config, instrument.Version, instrument.Status, instrument.CreatedBy, instrument.CreatedAt, instrument.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert instrument: %w", err)
	}
	return instrument, nil
}

func (r *postgresInstrumentRepository) GetInstrumentByID(ctx context.Context, instrumentID uuid.UUID) (*domain.Instrument, error) {
	var instrument domain.Instrument
	var config []byte
	err := r.db.QueryRowContext(ctx, `
		SELECT id, project_id, name, kind, config, version, status, created_by, created_at, updated_at
		FROM instruments WHERE id = $1
	`, instrumentID).Scan(
		&instrument.ID,
		&instrument.ProjectID,
		&instrument.Name,
		&instrument.Kind,
		&config,
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
	instrument.Config = json.RawMessage(config)
	return &instrument, nil
}

func (r *postgresInstrumentRepository) ListInstrumentsByProject(ctx context.Context, projectID uuid.UUID) ([]domain.Instrument, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, project_id, name, kind, config, version, status, created_by, created_at, updated_at
		FROM instruments WHERE project_id = $1 ORDER BY created_at DESC
	`, projectID)
	if err != nil {
		return nil, fmt.Errorf("list instruments: %w", err)
	}
	defer rows.Close()

	items := make([]domain.Instrument, 0)
	for rows.Next() {
		var instrument domain.Instrument
		var config []byte
		if err := rows.Scan(&instrument.ID, &instrument.ProjectID, &instrument.Name, &instrument.Kind, &config, &instrument.Version, &instrument.Status, &instrument.CreatedBy, &instrument.CreatedAt, &instrument.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan instrument: %w", err)
		}
		instrument.Config = json.RawMessage(config)
		items = append(items, instrument)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate instruments: %w", err)
	}
	return items, nil
}

func (r *postgresInstrumentRepository) UpdateInstrument(ctx context.Context, instrumentID uuid.UUID, name *string, config *json.RawMessage, status *domain.InstrumentStatus) (*domain.Instrument, error) {
	setParts := []string{"updated_at = NOW()"}
	args := []any{instrumentID}
	idx := 2
	if name != nil {
		setParts = append(setParts, fmt.Sprintf("name = $%d", idx))
		args = append(args, *name)
		idx++
	}
	if config != nil {
		setParts = append(setParts, fmt.Sprintf("config = $%d", idx))
		args = append(args, *config)
		idx++
	}
	if status != nil {
		setParts = append(setParts, fmt.Sprintf("status = $%d", idx))
		args = append(args, string(*status))
		idx++
	}
	if len(setParts) == 1 {
		return r.GetInstrumentByID(ctx, instrumentID)
	}
	query := fmt.Sprintf(`UPDATE instruments SET %s WHERE id = $1`, joinSQL(setParts))
	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("update instrument: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("rows affected: %w", err)
	}
	if rows == 0 {
		return nil, fmt.Errorf("instrument %s: %w", instrumentID, domain.ErrNotFound)
	}
	return r.GetInstrumentByID(ctx, instrumentID)
}

func (r *postgresInstrumentRepository) CreateQuestionnaire(ctx context.Context, projectID, createdBy uuid.UUID, name string, schema json.RawMessage) (*domain.Questionnaire, error) {
	questionnaire := &domain.Questionnaire{
		ID:        uuid.New(),
		ProjectID: projectID,
		Name:      name,
		Version:   1,
		Status:    "draft",
		Schema:    schema,
		CreatedBy: createdBy,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO questionnaires (id, project_id, name, version, status, schema, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, questionnaire.ID, questionnaire.ProjectID, questionnaire.Name, questionnaire.Version, questionnaire.Status, questionnaire.Schema, questionnaire.CreatedBy, questionnaire.CreatedAt, questionnaire.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert questionnaire: %w", err)
	}
	return questionnaire, nil
}

func (r *postgresInstrumentRepository) ListQuestionnairesByProject(ctx context.Context, projectID uuid.UUID) ([]domain.Questionnaire, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, project_id, name, version, status, schema, created_by, created_at, updated_at
		FROM questionnaires WHERE project_id = $1 ORDER BY created_at DESC
	`, projectID)
	if err != nil {
		return nil, fmt.Errorf("list questionnaires: %w", err)
	}
	defer rows.Close()
	items := make([]domain.Questionnaire, 0)
	for rows.Next() {
		var q domain.Questionnaire
		var schema []byte
		if err := rows.Scan(&q.ID, &q.ProjectID, &q.Name, &q.Version, &q.Status, &schema, &q.CreatedBy, &q.CreatedAt, &q.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan questionnaire: %w", err)
		}
		q.Schema = json.RawMessage(schema)
		items = append(items, q)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate questionnaires: %w", err)
	}
	return items, nil
}

func (r *postgresInstrumentRepository) GetQuestionnaireByID(ctx context.Context, questionnaireID uuid.UUID) (*domain.Questionnaire, error) {
	var q domain.Questionnaire
	var schema []byte
	err := r.db.QueryRowContext(ctx, `
		SELECT id, project_id, name, version, status, schema, created_by, created_at, updated_at
		FROM questionnaires WHERE id = $1
	`, questionnaireID).Scan(&q.ID, &q.ProjectID, &q.Name, &q.Version, &q.Status, &schema, &q.CreatedBy, &q.CreatedAt, &q.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("questionnaire %s: %w", questionnaireID, domain.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("query questionnaire: %w", err)
	}
	q.Schema = json.RawMessage(schema)
	return &q, nil
}

func (r *postgresInstrumentRepository) PublishQuestionnaire(ctx context.Context, questionnaireID uuid.UUID) (*domain.Questionnaire, error) {
	result, err := r.db.ExecContext(ctx, `
		UPDATE questionnaires SET status = 'published', updated_at = NOW() WHERE id = $1
	`, questionnaireID)
	if err != nil {
		return nil, fmt.Errorf("publish questionnaire: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("rows affected: %w", err)
	}
	if rows == 0 {
		return nil, fmt.Errorf("questionnaire %s: %w", questionnaireID, domain.ErrNotFound)
	}
	return r.GetQuestionnaireByID(ctx, questionnaireID)
}

func joinSQL(parts []string) string {
	return strings.Join(parts, ", ")
}
