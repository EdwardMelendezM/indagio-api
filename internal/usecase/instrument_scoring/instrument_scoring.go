package instrument_scoring

import (
	"context"
	"fmt"

	"indagio-api/internal/domain"

	"github.com/google/uuid"
)

type instrumentScoringUsecase struct {
	repo domain.InstrumentScoringRepository
}

func NewInstrumentScoringUsecase(repo domain.InstrumentScoringRepository) domain.InstrumentScoringUsecase {
	return &instrumentScoringUsecase{repo: repo}
}

func (uc *instrumentScoringUsecase) PublishVersion(ctx context.Context, actorID, instrumentID uuid.UUID, items []domain.InstrumentItem, subscales []domain.InstrumentSubscale, rules []domain.ScoringRule, bands []domain.ScoreBand) (*domain.InstrumentCatalog, error) {
	if actorID == uuid.Nil || instrumentID == uuid.Nil {
		return nil, fmt.Errorf("actor/instrument id: %w", domain.ErrValidation)
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("items: %w", domain.ErrValidation)
	}
	if len(rules) == 0 {
		return nil, fmt.Errorf("rules: at least a total-score rule is required: %w", domain.ErrValidation)
	}

	subscaleKeys := make(map[string]bool, len(subscales))
	for _, s := range subscales {
		subscaleKeys[s.Key] = true
	}
	for _, it := range items {
		if it.SubscaleKey != nil && !subscaleKeys[*it.SubscaleKey] {
			return nil, fmt.Errorf("item %q references unknown subscale %q: %w", it.QuestionKey, *it.SubscaleKey, domain.ErrValidation)
		}
	}
	for _, ru := range rules {
		if ru.SubscaleKey != nil && !subscaleKeys[*ru.SubscaleKey] {
			return nil, fmt.Errorf("rule references unknown subscale %q: %w", *ru.SubscaleKey, domain.ErrValidation)
		}
	}
	for _, b := range bands {
		if b.SubscaleKey != nil && !subscaleKeys[*b.SubscaleKey] {
			return nil, fmt.Errorf("band %q references unknown subscale %q: %w", b.Label, *b.SubscaleKey, domain.ErrValidation)
		}
	}

	return uc.repo.PublishVersion(ctx, instrumentID, items, subscales, rules, bands)
}

func (uc *instrumentScoringUsecase) GetCatalog(ctx context.Context, actorID, instrumentID uuid.UUID, version *int) (*domain.InstrumentCatalog, error) {
	if actorID == uuid.Nil || instrumentID == uuid.Nil {
		return nil, fmt.Errorf("actor/instrument id: %w", domain.ErrValidation)
	}
	if version == nil {
		return uc.repo.GetLatestCatalog(ctx, instrumentID)
	}
	return uc.repo.GetCatalogByVersion(ctx, instrumentID, *version)
}

func (uc *instrumentScoringUsecase) ListVersions(ctx context.Context, actorID, instrumentID uuid.UUID) ([]int, error) {
	if actorID == uuid.Nil || instrumentID == uuid.Nil {
		return nil, fmt.Errorf("actor/instrument id: %w", domain.ErrValidation)
	}
	return uc.repo.ListVersions(ctx, instrumentID)
}
