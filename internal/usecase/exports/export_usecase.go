package exports

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"indagio-api/internal/domain"

	"github.com/google/uuid"
)

type exportUsecase struct {
	repo domain.ExportRepository
}

func NewExportUsecase(repo domain.ExportRepository) domain.ExportUsecase {
	return &exportUsecase{repo: repo}
}

func (uc *exportUsecase) RequestExport(ctx context.Context, actorID, projectID uuid.UUID, format string, options json.RawMessage) (*domain.ProjectExport, error) {
	if actorID == uuid.Nil || projectID == uuid.Nil {
		return nil, fmt.Errorf("actor/project ids: %w", domain.ErrValidation)
	}
	format = strings.TrimSpace(strings.ToLower(format))
	if format == "" {
		return nil, fmt.Errorf("export format: %w", domain.ErrValidation)
	}
	allowed := map[string]struct{}{
		string(domain.ExportFormatCSV):  {},
		string(domain.ExportFormatJSON): {},
		string(domain.ExportFormatZIP):  {},
	}
	if _, ok := allowed[format]; !ok {
		return nil, fmt.Errorf("export format: %w", domain.ErrValidation)
	}
	if len(options) > 0 && !json.Valid(options) {
		return nil, fmt.Errorf("export options: %w", domain.ErrValidation)
	}
	return uc.repo.Create(ctx, projectID, actorID, format, options)
}

func (uc *exportUsecase) ListExports(ctx context.Context, actorID, projectID uuid.UUID) ([]domain.ProjectExport, error) {
	if actorID == uuid.Nil || projectID == uuid.Nil {
		return nil, fmt.Errorf("actor/project ids: %w", domain.ErrValidation)
	}
	return uc.repo.ListByProject(ctx, projectID)
}

func (uc *exportUsecase) GetExport(ctx context.Context, actorID, exportID uuid.UUID) (*domain.ProjectExport, error) {
	if actorID == uuid.Nil || exportID == uuid.Nil {
		return nil, fmt.Errorf("actor/export ids: %w", domain.ErrValidation)
	}
	return uc.repo.GetByID(ctx, exportID)
}
