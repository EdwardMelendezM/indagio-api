package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"indagio-api/internal/domain"

	"github.com/google/uuid"
)

type syncUsecase struct {
	repo domain.SyncRepository
}

func NewSyncUsecase(repo domain.SyncRepository) domain.SyncUsecase {
	return &syncUsecase{repo: repo}
}

func (uc *syncUsecase) SubmitBatch(ctx context.Context, actorID, projectID, participantID uuid.UUID, clientGeneratedBatchID string, payload json.RawMessage) (*domain.SyncBatch, error) {
	if actorID == uuid.Nil || projectID == uuid.Nil || participantID == uuid.Nil {
		return nil, fmt.Errorf("actor/project/participant ids: %w", domain.ErrValidation)
	}
	clientGeneratedBatchID = strings.TrimSpace(clientGeneratedBatchID)
	if clientGeneratedBatchID == "" {
		return nil, fmt.Errorf("batch id: %w", domain.ErrValidation)
	}
	if !json.Valid(payload) {
		return nil, fmt.Errorf("payload: %w", domain.ErrValidation)
	}
	return uc.repo.CreateBatch(ctx, projectID, participantID, clientGeneratedBatchID, payload)
}

func (uc *syncUsecase) ListBatches(ctx context.Context, actorID, participantID uuid.UUID) ([]domain.SyncBatch, error) {
	if actorID == uuid.Nil || participantID == uuid.Nil {
		return nil, fmt.Errorf("actor/participant ids: %w", domain.ErrValidation)
	}
	return uc.repo.ListBatchesByParticipant(ctx, participantID)
}

func (uc *syncUsecase) GetBatch(ctx context.Context, actorID, batchID uuid.UUID) (*domain.SyncBatch, error) {
	if actorID == uuid.Nil || batchID == uuid.Nil {
		return nil, fmt.Errorf("actor/batch ids: %w", domain.ErrValidation)
	}
	return uc.repo.GetBatchByID(ctx, batchID)
}
