package domain

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ExportFormat defines the output format for a data export.
type ExportFormat string

const (
	ExportFormatCSV  ExportFormat = "csv"
	ExportFormatJSON ExportFormat = "json"
	ExportFormatZIP  ExportFormat = "zip"
)

// ExportStatus tracks the state of an async export job.
type ExportStatus string

const (
	ExportStatusPending    ExportStatus = "pending"
	ExportStatusProcessing ExportStatus = "processing"
	ExportStatusCompleted  ExportStatus = "completed"
	ExportStatusFailed     ExportStatus = "failed"
)

// ProjectExport is a background export job for research data.
type ProjectExport struct {
	ID          uuid.UUID
	ProjectID   uuid.UUID
	CreatedBy   uuid.UUID
	Format      ExportFormat
	Status      ExportStatus
	Options     json.RawMessage
	FileKey     *string
	ErrorLog    *string
	StartedAt   *time.Time
	CompletedAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// ExportRepository persists export jobs and their results.
type ExportRepository interface {
	Create(ctx context.Context, projectID, createdBy uuid.UUID, format string, options json.RawMessage) (*ProjectExport, error)
	GetByID(ctx context.Context, exportID uuid.UUID) (*ProjectExport, error)
	ListByProject(ctx context.Context, projectID uuid.UUID) ([]ProjectExport, error)
	UpdateStatus(ctx context.Context, exportID uuid.UUID, status ExportStatus, fileKey *string, errorLog *string) (*ProjectExport, error)
}

// ExportUsecase handles request, listing and status workflows for project exports.
type ExportUsecase interface {
	RequestExport(ctx context.Context, actorID, projectID uuid.UUID, format string, options json.RawMessage) (*ProjectExport, error)
	ListExports(ctx context.Context, actorID, projectID uuid.UUID) ([]ProjectExport, error)
	GetExport(ctx context.Context, actorID, exportID uuid.UUID) (*ProjectExport, error)
}
