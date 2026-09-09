package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"indagio-api/internal/domain"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type postgresJobRepository struct {
	db *sql.DB
}

func NewPostgresJobRepository(db *sql.DB) domain.JobRepository {
	return &postgresJobRepository{db: db}
}

func (r *postgresJobRepository) Enqueue(ctx context.Context, jobType string, payload any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("jobs: marshal payload: %w", err)
	}
	_, err = r.db.ExecContext(ctx, queryEnqueue, jobType, raw)
	return err
}

func (r *postgresJobRepository) Dequeue(ctx context.Context, jobTypes []string) (*domain.Job, error) {
	row := r.db.QueryRowContext(ctx, queryDequeue, pq.Array(jobTypes))

	var job domain.Job
	err := row.Scan(
		&job.ID, &job.Type, &job.Payload,
		&job.Attempts, &job.MaxAttempts,
		&job.RunAt, &job.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &job, err
}

func (r *postgresJobRepository) MarkCompleted(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, queryMarkCompleted, id)
	return err
}

func (r *postgresJobRepository) MarkFailed(ctx context.Context, id uuid.UUID, errMsg string, scheduleRetry bool) error {
	if scheduleRetry {
		_, err := r.db.ExecContext(ctx, queryMarkFailed, id, errMsg)
		return err
	}
	_, err := r.db.ExecContext(ctx, `
		UPDATE background_jobs
		SET status='failed', error_log=$2, updated_at=NOW()
		WHERE id=$1`, id, errMsg,
	)
	return err
}

func (r *postgresJobRepository) PurgeOld(ctx context.Context, olderThanDays int) (int64, error) {
	row := r.db.QueryRowContext(ctx, queryPurgeOld, olderThanDays)
	var deleted int64
	err := row.Scan(&deleted)
	return deleted, err
}
