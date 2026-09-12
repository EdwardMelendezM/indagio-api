package instruments

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"indagio-api/internal/domain"

	"github.com/google/uuid"
)

type instrumentUsecase struct {
	repo domain.InstrumentRepository
}

func NewInstrumentUsecase(repo domain.InstrumentRepository) domain.InstrumentUsecase {
	return &instrumentUsecase{repo: repo}
}

func (uc *instrumentUsecase) CreateInstrument(ctx context.Context, actorID, projectID uuid.UUID, name string, kind string, config json.RawMessage) (*domain.Instrument, error) {
	if actorID == uuid.Nil || projectID == uuid.Nil {
		return nil, fmt.Errorf("actor/project id: %w", domain.ErrValidation)
	}
	name = strings.TrimSpace(name)
	if len(name) < 2 || len(name) > 160 {
		return nil, fmt.Errorf("name: %w", domain.ErrValidation)
	}
	if !json.Valid(config) {
		return nil, fmt.Errorf("config: %w", domain.ErrValidation)
	}
	instrumentKind := domain.InstrumentKind(kind)
	if instrumentKind != domain.InstrumentKindText && instrumentKind != domain.InstrumentKindNumeric && instrumentKind != domain.InstrumentKindScale && instrumentKind != domain.InstrumentKindSingleChoice && instrumentKind != domain.InstrumentKindMultiChoice && instrumentKind != domain.InstrumentKindAudio && instrumentKind != domain.InstrumentKindVideo {
		return nil, fmt.Errorf("kind: %w", domain.ErrValidation)
	}

	return uc.repo.CreateInstrument(ctx, projectID, actorID, name, instrumentKind, config)
}

func (uc *instrumentUsecase) ListInstruments(ctx context.Context, actorID, projectID uuid.UUID) ([]domain.Instrument, error) {
	if actorID == uuid.Nil || projectID == uuid.Nil {
		return nil, fmt.Errorf("actor/project id: %w", domain.ErrValidation)
	}
	return uc.repo.ListInstrumentsByProject(ctx, projectID)
}

func (uc *instrumentUsecase) UpdateInstrument(ctx context.Context, actorID, instrumentID uuid.UUID, name *string, config *json.RawMessage, status *domain.InstrumentStatus) (*domain.Instrument, error) {
	if actorID == uuid.Nil || instrumentID == uuid.Nil {
		return nil, fmt.Errorf("actor/instrument id: %w", domain.ErrValidation)
	}
	if name != nil {
		*name = strings.TrimSpace(*name)
		if len(*name) < 2 || len(*name) > 160 {
			return nil, fmt.Errorf("name: %w", domain.ErrValidation)
		}
	}
	if config != nil && !json.Valid(*config) {
		return nil, fmt.Errorf("config: %w", domain.ErrValidation)
	}
	if status != nil {
		switch *status {
		case domain.InstrumentStatusDraft, domain.InstrumentStatusActive, domain.InstrumentStatusArchived:
		default:
			return nil, fmt.Errorf("status: %w", domain.ErrValidation)
		}
	}
	return uc.repo.UpdateInstrument(ctx, instrumentID, name, config, status)
}
