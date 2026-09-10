package sync

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"indagio-api/internal/domain"

	"github.com/google/uuid"
)

type stubSyncRepo struct {
	batches map[uuid.UUID]*domain.SyncBatch
}

func (s *stubSyncRepo) CreateBatch(ctx context.Context, projectID, participantID uuid.UUID, clientGeneratedBatchID string, payload json.RawMessage) (*domain.SyncBatch, error) {
	batch := &domain.SyncBatch{
		ID:                     uuid.New(),
		ProjectID:              projectID,
		ParticipantID:          participantID,
		ClientGeneratedBatchID: clientGeneratedBatchID,
		Payload:                payload,
		Status:                 domain.SyncBatchStatusPending,
		CreatedAt:              time.Now().UTC(),
		UpdatedAt:              time.Now().UTC(),
	}
	s.batches[batch.ID] = batch
	return batch, nil
}

func (s *stubSyncRepo) GetBatchByID(ctx context.Context, batchID uuid.UUID) (*domain.SyncBatch, error) {
	batch, ok := s.batches[batchID]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return batch, nil
}

func (s *stubSyncRepo) ListBatchesByParticipant(ctx context.Context, participantID uuid.UUID) ([]domain.SyncBatch, error) {
	out := make([]domain.SyncBatch, 0)
	for _, batch := range s.batches {
		if batch.ParticipantID == participantID {
			out = append(out, *batch)
		}
	}
	return out, nil
}

func (s *stubSyncRepo) MarkBatchProcessed(ctx context.Context, batchID uuid.UUID) (*domain.SyncBatch, error) {
	return nil, errors.New("not implemented")
}

func (s *stubSyncRepo) CreateEvent(ctx context.Context, batchID uuid.UUID, eventType, entityID string, payload json.RawMessage) (*domain.SyncEvent, error) {
	return nil, errors.New("not implemented")
}

func (s *stubSyncRepo) ListEventsByBatch(ctx context.Context, batchID uuid.UUID) ([]domain.SyncEvent, error) {
	return nil, nil
}

func TestSyncUsecase_SubmitBatch_ValidatesPayload(t *testing.T) {
	actorID := uuid.New()
	projectID := uuid.New()
	participantID := uuid.New()
	repo := &stubSyncRepo{batches: map[uuid.UUID]*domain.SyncBatch{}}
	uc := NewSyncUsecase(repo)
	_, err := uc.SubmitBatch(context.Background(), actorID, projectID, participantID, "batch-1", json.RawMessage(`not-json`))
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
}
