package exports

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"indagio-api/internal/domain"

	"github.com/google/uuid"
)

type stubExportRepo struct {
	exports map[uuid.UUID]*domain.ProjectExport
}

func (s *stubExportRepo) Create(ctx context.Context, projectID, createdBy uuid.UUID, format string, options json.RawMessage) (*domain.ProjectExport, error) {
	exportItem := &domain.ProjectExport{
		ID:        uuid.New(),
		ProjectID: projectID,
		CreatedBy: createdBy,
		Format:    domain.ExportFormat(format),
		Status:    domain.ExportStatusPending,
		Options:   options,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	s.exports[exportItem.ID] = exportItem
	return exportItem, nil
}

func (s *stubExportRepo) GetByID(ctx context.Context, exportID uuid.UUID) (*domain.ProjectExport, error) {
	exportItem, ok := s.exports[exportID]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return exportItem, nil
}

func (s *stubExportRepo) ListByProject(ctx context.Context, projectID uuid.UUID) ([]domain.ProjectExport, error) {
	out := make([]domain.ProjectExport, 0)
	for _, exportItem := range s.exports {
		if exportItem.ProjectID == projectID {
			out = append(out, *exportItem)
		}
	}
	return out, nil
}

func (s *stubExportRepo) UpdateStatus(ctx context.Context, exportID uuid.UUID, status domain.ExportStatus, fileKey *string, errorLog *string) (*domain.ProjectExport, error) {
	exportItem, ok := s.exports[exportID]
	if !ok {
		return nil, domain.ErrNotFound
	}
	exportItem.Status = status
	exportItem.FileKey = fileKey
	exportItem.ErrorLog = errorLog
	exportItem.UpdatedAt = time.Now().UTC()
	return exportItem, nil
}

func TestExportUsecase_RequestExport_ValidatesFormat(t *testing.T) {
	actorID := uuid.New()
	projectID := uuid.New()
	repo := &stubExportRepo{exports: map[uuid.UUID]*domain.ProjectExport{}}
	uc := NewExportUsecase(repo)
	_, err := uc.RequestExport(context.Background(), actorID, projectID, "xml", json.RawMessage(`{"include":"all"}`))
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
}

func TestExportUsecase_RequestExport_CreatesPendingExport(t *testing.T) {
	actorID := uuid.New()
	projectID := uuid.New()
	repo := &stubExportRepo{exports: map[uuid.UUID]*domain.ProjectExport{}}
	uc := NewExportUsecase(repo)
	exportItem, err := uc.RequestExport(context.Background(), actorID, projectID, "csv", json.RawMessage(`{"include":"answers"}`))
	if err != nil {
		t.Fatalf("request export: %v", err)
	}
	if exportItem.Status != domain.ExportStatusPending {
		t.Fatalf("expected pending status, got %s", exportItem.Status)
	}
}
