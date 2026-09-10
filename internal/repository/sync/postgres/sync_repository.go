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

type syncRepository struct {
	db *sql.DB
}

func NewSyncRepository(db *sql.DB) domain.SyncRepository {
	return &syncRepository{db: db}
}

func (r *syncRepository) CreateBatch(ctx context.Context, projectID, participantID uuid.UUID, clientGeneratedBatchID string, payload json.RawMessage) (*domain.SyncBatch, error) {
	batch := &domain.SyncBatch{
		ID:                     uuid.New(),
		ProjectID:              projectID,
		ParticipantID:          participantID,
		ClientGeneratedBatchID: clientGeneratedBatchID,
		Payload:                payload,
		Status:                 domain.SyncBatchStatusPending,
	}
	query := `
		INSERT INTO sync_batches (id, project_id, participant_id, client_generated_batch_id, payload, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5::jsonb, $6, NOW(), NOW())
		RETURNING id, project_id, participant_id, client_generated_batch_id, payload, status, created_at, updated_at
	`
	row := r.db.QueryRowContext(ctx, query, batch.ID, projectID, participantID, clientGeneratedBatchID, string(payload), batch.Status)
	if err := row.Scan(&batch.ID, &batch.ProjectID, &batch.ParticipantID, &batch.ClientGeneratedBatchID, &batch.Payload, &batch.Status, &batch.CreatedAt, &batch.UpdatedAt); err != nil {
		return nil, fmt.Errorf("create sync batch: %w", err)
	}
	return batch, nil
}

func (r *syncRepository) GetBatchByID(ctx context.Context, batchID uuid.UUID) (*domain.SyncBatch, error) {
	query := `SELECT id, project_id, participant_id, client_generated_batch_id, payload, status, created_at, updated_at FROM sync_batches WHERE id = $1`
	var batch domain.SyncBatch
	if err := r.db.QueryRowContext(ctx, query, batchID).Scan(
		&batch.ID,
		&batch.ProjectID,
		&batch.ParticipantID,
		&batch.ClientGeneratedBatchID,
		&batch.Payload,
		&batch.Status,
		&batch.CreatedAt,
		&batch.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("read sync batch: %w", err)
	}
	return &batch, nil
}

func (r *syncRepository) ListBatchesByParticipant(ctx context.Context, participantID uuid.UUID) ([]domain.SyncBatch, error) {
	query := `SELECT id, project_id, participant_id, client_generated_batch_id, payload, status, created_at, updated_at FROM sync_batches WHERE participant_id = $1 ORDER BY created_at DESC`
	rows, err := r.db.QueryContext(ctx, query, participantID)
	if err != nil {
		return nil, fmt.Errorf("list sync batches: %w", err)
	}
	defer rows.Close()
	batches := make([]domain.SyncBatch, 0)
	for rows.Next() {
		var batch domain.SyncBatch
		if err := rows.Scan(&batch.ID, &batch.ProjectID, &batch.ParticipantID, &batch.ClientGeneratedBatchID, &batch.Payload, &batch.Status, &batch.CreatedAt, &batch.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan sync batch: %w", err)
		}
		batches = append(batches, batch)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sync batches: %w", err)
	}
	return batches, nil
}

func (r *syncRepository) MarkBatchProcessed(ctx context.Context, batchID uuid.UUID) (*domain.SyncBatch, error) {
	query := `UPDATE sync_batches SET status = $1, updated_at = NOW() WHERE id = $2 RETURNING id, project_id, participant_id, client_generated_batch_id, payload, status, created_at, updated_at`
	var batch domain.SyncBatch
	if err := r.db.QueryRowContext(ctx, query, domain.SyncBatchStatusProcessed, batchID).Scan(
		&batch.ID,
		&batch.ProjectID,
		&batch.ParticipantID,
		&batch.ClientGeneratedBatchID,
		&batch.Payload,
		&batch.Status,
		&batch.CreatedAt,
		&batch.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("mark sync batch processed: %w", err)
	}
	return &batch, nil
}

func (r *syncRepository) CreateEvent(ctx context.Context, batchID uuid.UUID, eventType, entityID string, payload json.RawMessage) (*domain.SyncEvent, error) {
	event := &domain.SyncEvent{ID: uuid.New(), BatchID: batchID, EventType: eventType, EntityID: entityID, Payload: payload}
	query := `INSERT INTO sync_events (id, batch_id, event_type, entity_id, payload, created_at) VALUES ($1, $2, $3, $4, $5::jsonb, NOW()) RETURNING id, batch_id, event_type, entity_id, payload, created_at`
	row := r.db.QueryRowContext(ctx, query, event.ID, event.BatchID, event.EventType, event.EntityID, string(payload))
	if err := row.Scan(&event.ID, &event.BatchID, &event.EventType, &event.EntityID, &event.Payload, &event.CreatedAt); err != nil {
		return nil, fmt.Errorf("create sync event: %w", err)
	}
	return event, nil
}

func (r *syncRepository) ListEventsByBatch(ctx context.Context, batchID uuid.UUID) ([]domain.SyncEvent, error) {
	query := `SELECT id, batch_id, event_type, entity_id, payload, created_at FROM sync_events WHERE batch_id = $1 ORDER BY created_at DESC`
	rows, err := r.db.QueryContext(ctx, query, batchID)
	if err != nil {
		return nil, fmt.Errorf("list sync events: %w", err)
	}
	defer rows.Close()
	events := make([]domain.SyncEvent, 0)
	for rows.Next() {
		var event domain.SyncEvent
		if err := rows.Scan(&event.ID, &event.BatchID, &event.EventType, &event.EntityID, &event.Payload, &event.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan sync event: %w", err)
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sync events: %w", err)
	}
	return events, nil
}
