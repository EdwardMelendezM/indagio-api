package domain

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// SyncBatchStatus tracks the processing state of a batch sent by a mobile client.
type SyncBatchStatus string

const (
	SyncBatchStatusPending   SyncBatchStatus = "pending"
	SyncBatchStatusProcessed SyncBatchStatus = "processed"
	SyncBatchStatusFailed    SyncBatchStatus = "failed"
)

// SyncBatch represents a bulk offline submission from a participant.
type SyncBatch struct {
	ID                     uuid.UUID
	ProjectID              uuid.UUID
	ParticipantID          uuid.UUID
	ClientGeneratedBatchID string
	Payload                json.RawMessage
	Status                 SyncBatchStatus
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

// SyncEvent captures a processed item within a batch.
type SyncEvent struct {
	ID        uuid.UUID
	BatchID   uuid.UUID
	EventType string
	EntityID  string
	Payload   json.RawMessage
	CreatedAt time.Time
}

// SyncRepository persists offline sync batches and processed events.
type SyncRepository interface {
	CreateBatch(ctx context.Context, projectID, participantID uuid.UUID, clientGeneratedBatchID string, payload json.RawMessage) (*SyncBatch, error)
	GetBatchByID(ctx context.Context, batchID uuid.UUID) (*SyncBatch, error)
	ListBatchesByParticipant(ctx context.Context, participantID uuid.UUID) ([]SyncBatch, error)
	MarkBatchProcessed(ctx context.Context, batchID uuid.UUID) (*SyncBatch, error)
	CreateEvent(ctx context.Context, batchID uuid.UUID, eventType, entityID string, payload json.RawMessage) (*SyncEvent, error)
	ListEventsByBatch(ctx context.Context, batchID uuid.UUID) ([]SyncEvent, error)
}

// SyncUsecase manages batch synchronization and idempotent retry semantics.
type SyncUsecase interface {
	SubmitBatch(ctx context.Context, actorID, projectID, participantID uuid.UUID, clientGeneratedBatchID string, payload json.RawMessage) (*SyncBatch, error)
	ListBatches(ctx context.Context, actorID, participantID uuid.UUID) ([]SyncBatch, error)
	GetBatch(ctx context.Context, actorID, batchID uuid.UUID) (*SyncBatch, error)
}
