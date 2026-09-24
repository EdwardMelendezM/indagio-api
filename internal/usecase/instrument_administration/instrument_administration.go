package instrument_administrations

import (
	"context"
	"fmt"
	"time"

	"indagio-api/internal/domain"

	"github.com/google/uuid"
)

type instrumentAdministrationUsecase struct {
	repo        domain.InstrumentAdministrationRepository
	scoringRepo domain.InstrumentScoringRepository
	answerRepo  domain.AnswerRepository
	engine      domain.ScoringEngine
}

func NewInstrumentAdministrationUsecase(repo domain.InstrumentAdministrationRepository, scoringRepo domain.InstrumentScoringRepository, answerRepo domain.AnswerRepository, engine domain.ScoringEngine) domain.InstrumentAdministrationUsecase {
	return &instrumentAdministrationUsecase{repo: repo, scoringRepo: scoringRepo, answerRepo: answerRepo, engine: engine}
}

func (uc *instrumentAdministrationUsecase) StartAdministration(ctx context.Context, actorID, projectID, participantID, instrumentID uuid.UUID, clientGeneratedID *string) (*domain.InstrumentAdministration, error) {
	if actorID == uuid.Nil || projectID == uuid.Nil || participantID == uuid.Nil || instrumentID == uuid.Nil {
		return nil, fmt.Errorf("actor/project/participant/instrument id: %w", domain.ErrValidation)
	}
	catalog, err := uc.scoringRepo.GetLatestCatalog(ctx, instrumentID)
	if err != nil {
		return nil, fmt.Errorf("resolve instrument catalog: %w", err)
	}
	return uc.repo.CreateAdministration(ctx, projectID, participantID, instrumentID, catalog.Version, clientGeneratedID)
}

func (uc *instrumentAdministrationUsecase) CompleteAdministration(ctx context.Context, actorID, administrationID uuid.UUID) (*domain.InstrumentAdministration, []domain.InstrumentScore, error) {
	if actorID == uuid.Nil || administrationID == uuid.Nil {
		return nil, nil, fmt.Errorf("actor/administration id: %w", domain.ErrValidation)
	}
	admin, err := uc.repo.GetAdministrationByID(ctx, administrationID)
	if err != nil {
		return nil, nil, err
	}
	if admin.Status == domain.AdministrationStatusAbandoned {
		return nil, nil, fmt.Errorf("administration %s is abandoned: %w", administrationID, domain.ErrConflict)
	}
	now := time.Now().UTC()
	admin, err = uc.repo.UpdateStatus(ctx, administrationID, domain.AdministrationStatusCompleted, &now)
	if err != nil {
		return nil, nil, err
	}
	scores, err := uc.computeAndSave(ctx, admin)
	if err != nil {
		return nil, nil, err
	}
	return admin, scores, nil
}

func (uc *instrumentAdministrationUsecase) AbandonAdministration(ctx context.Context, actorID, administrationID uuid.UUID) (*domain.InstrumentAdministration, error) {
	if actorID == uuid.Nil || administrationID == uuid.Nil {
		return nil, fmt.Errorf("actor/administration id: %w", domain.ErrValidation)
	}
	return uc.repo.UpdateStatus(ctx, administrationID, domain.AdministrationStatusAbandoned, nil)
}

func (uc *instrumentAdministrationUsecase) ListAdministrations(ctx context.Context, actorID, participantID, instrumentID uuid.UUID) ([]domain.InstrumentAdministration, error) {
	if actorID == uuid.Nil || participantID == uuid.Nil || instrumentID == uuid.Nil {
		return nil, fmt.Errorf("actor/participant/instrument id: %w", domain.ErrValidation)
	}
	return uc.repo.ListAdministrationsByParticipant(ctx, participantID, instrumentID)
}

func (uc *instrumentAdministrationUsecase) GetScores(ctx context.Context, actorID, administrationID uuid.UUID) ([]domain.InstrumentScore, error) {
	if actorID == uuid.Nil || administrationID == uuid.Nil {
		return nil, fmt.Errorf("actor/administration id: %w", domain.ErrValidation)
	}
	return uc.repo.GetScores(ctx, administrationID)
}

func (uc *instrumentAdministrationUsecase) RecomputeScores(ctx context.Context, actorID, administrationID uuid.UUID) ([]domain.InstrumentScore, error) {
	if actorID == uuid.Nil || administrationID == uuid.Nil {
		return nil, fmt.Errorf("actor/administration id: %w", domain.ErrValidation)
	}
	admin, err := uc.repo.GetAdministrationByID(ctx, administrationID)
	if err != nil {
		return nil, err
	}
	return uc.computeAndSave(ctx, admin)
}

func (uc *instrumentAdministrationUsecase) computeAndSave(ctx context.Context, admin *domain.InstrumentAdministration) ([]domain.InstrumentScore, error) {
	catalog, err := uc.scoringRepo.GetCatalogByVersion(ctx, admin.InstrumentID, admin.InstrumentVersion)
	if err != nil {
		return nil, fmt.Errorf("resolve catalog v%d: %w", admin.InstrumentVersion, err)
	}
	// Requires AnswerRepository.ListAnswersByAdministration — see CHANGES.md.
	answers, err := uc.answerRepo.ListAnswersByAdministration(ctx, admin.ID)
	if err != nil {
		return nil, fmt.Errorf("list answers: %w", err)
	}
	scores, err := uc.engine.Compute(catalog.Items, catalog.Rules, catalog.Bands, answers)
	if err != nil {
		return nil, fmt.Errorf("compute scores: %w", err)
	}
	for i := range scores {
		scores[i].AdministrationID = admin.ID
		scores[i].ComputedAt = time.Now().UTC()
	}
	if err := uc.repo.SaveScores(ctx, admin.ID, scores); err != nil {
		return nil, fmt.Errorf("save scores: %w", err)
	}
	return scores, nil
}
