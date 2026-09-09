package mocks

import (
	"context"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"foro-unsaac-backend/internal/domain"
)

// JobRepository is a mock implementation of domain.JobRepository
type JobRepository struct {
	mock.Mock
}

func (m *JobRepository) Enqueue(ctx context.Context, jobType string, payload any) error {
	args := m.Called(ctx, jobType, payload)
	return args.Error(0)
}

func (m *JobRepository) Dequeue(ctx context.Context, jobTypes []string) (*domain.Job, error) {
	args := m.Called(ctx, jobTypes)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Job), args.Error(1)
}

func (m *JobRepository) MarkCompleted(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *JobRepository) MarkFailed(ctx context.Context, id uuid.UUID, errMsg string, scheduleRetry bool) error {
	args := m.Called(ctx, id, errMsg, scheduleRetry)
	return args.Error(0)
}

func (m *JobRepository) PurgeOld(ctx context.Context, olderThanDays int) (int64, error) {
	args := m.Called(ctx, olderThanDays)
	return args.Get(0).(int64), args.Error(1)
}
