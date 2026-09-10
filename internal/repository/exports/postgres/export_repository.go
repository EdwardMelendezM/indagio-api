package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"indagio-api/internal/domain"

	"github.com/google/uuid"
)

type exportRepository struct {
	db *sql.DB
}

func NewExportRepository(db *sql.DB) domain.ExportRepository {
	return &exportRepository{db: db}
}

func (r *exportRepository) Create(ctx context.Context, projectID, createdBy uuid.UUID, format string, options json.RawMessage) (*domain.ProjectExport, error) {
	exportItem := &domain.ProjectExport{
		ID:        uuid.New(),
		ProjectID: projectID,
		CreatedBy: createdBy,
		Format:    domain.ExportFormat(format),
		Status:    domain.ExportStatusPending,
		Options:   options,
	}
	query := `
		INSERT INTO project_exports (id, project_id, created_by, format, status, options, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6::jsonb, NOW(), NOW())
		RETURNING id, project_id, created_by, format, status, options, file_key, error_log, started_at, completed_at, created_at, updated_at
	`
	row := r.db.QueryRowContext(ctx, query, exportItem.ID, projectID, createdBy, format, exportItem.Status, string(options))
	if err := row.Scan(&exportItem.ID, &exportItem.ProjectID, &exportItem.CreatedBy, &exportItem.Format, &exportItem.Status, &exportItem.Options, &exportItem.FileKey, &exportItem.ErrorLog, &exportItem.StartedAt, &exportItem.CompletedAt, &exportItem.CreatedAt, &exportItem.UpdatedAt); err != nil {
		return nil, fmt.Errorf("insert project export: %w", err)
	}
	return exportItem, nil
}

func (r *exportRepository) GetByID(ctx context.Context, exportID uuid.UUID) (*domain.ProjectExport, error) {
	query := `SELECT id, project_id, created_by, format, status, options, file_key, error_log, started_at, completed_at, created_at, updated_at FROM project_exports WHERE id = $1`
	var exportItem domain.ProjectExport
	if err := r.db.QueryRowContext(ctx, query, exportID).Scan(
		&exportItem.ID,
		&exportItem.ProjectID,
		&exportItem.CreatedBy,
		&exportItem.Format,
		&exportItem.Status,
		&exportItem.Options,
		&exportItem.FileKey,
		&exportItem.ErrorLog,
		&exportItem.StartedAt,
		&exportItem.CompletedAt,
		&exportItem.CreatedAt,
		&exportItem.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("read project export: %w", err)
	}
	return &exportItem, nil
}

func (r *exportRepository) ListByProject(ctx context.Context, projectID uuid.UUID) ([]domain.ProjectExport, error) {
	query := `SELECT id, project_id, created_by, format, status, options, file_key, error_log, started_at, completed_at, created_at, updated_at FROM project_exports WHERE project_id = $1 ORDER BY created_at DESC`
	rows, err := r.db.QueryContext(ctx, query, projectID)
	if err != nil {
		return nil, fmt.Errorf("list project exports: %w", err)
	}
	defer rows.Close()
	out := make([]domain.ProjectExport, 0)
	for rows.Next() {
		var exportItem domain.ProjectExport
		if err := rows.Scan(
			&exportItem.ID,
			&exportItem.ProjectID,
			&exportItem.CreatedBy,
			&exportItem.Format,
			&exportItem.Status,
			&exportItem.Options,
			&exportItem.FileKey,
			&exportItem.ErrorLog,
			&exportItem.StartedAt,
			&exportItem.CompletedAt,
			&exportItem.CreatedAt,
			&exportItem.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan project export: %w", err)
		}
		out = append(out, exportItem)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate project exports: %w", err)
	}
	return out, nil
}

func (r *exportRepository) UpdateStatus(ctx context.Context, exportID uuid.UUID, status domain.ExportStatus, fileKey *string, errorLog *string) (*domain.ProjectExport, error) {
	query := `UPDATE project_exports SET status = $2, file_key = $3, error_log = $4, updated_at = NOW(), completed_at = CASE WHEN $2 IN ('completed','failed') THEN NOW() ELSE completed_at END WHERE id = $1 RETURNING id, project_id, created_by, format, status, options, file_key, error_log, started_at, completed_at, created_at, updated_at`
	var exportItem domain.ProjectExport
	if err := r.db.QueryRowContext(ctx, query, exportID, status, fileKey, errorLog).Scan(
		&exportItem.ID,
		&exportItem.ProjectID,
		&exportItem.CreatedBy,
		&exportItem.Format,
		&exportItem.Status,
		&exportItem.Options,
		&exportItem.FileKey,
		&exportItem.ErrorLog,
		&exportItem.StartedAt,
		&exportItem.CompletedAt,
		&exportItem.CreatedAt,
		&exportItem.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("update project export status: %w", err)
	}
	return &exportItem, nil
}
