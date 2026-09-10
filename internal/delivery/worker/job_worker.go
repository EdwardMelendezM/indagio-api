package worker

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"indagio-api/internal/domain"
)

type JobWorker struct {
	jobRepo      domain.JobRepository
	emailSvc     domain.EmailService
	logger       *slog.Logger
	jobTypes     []string
	pollInterval time.Duration
}

func NewJobWorker(
	jobRepo domain.JobRepository,
	emailSvc domain.EmailService,
	logger *slog.Logger,
) *JobWorker {
	return &JobWorker{
		jobRepo:  jobRepo,
		emailSvc: emailSvc,
		logger:   logger,
		jobTypes: []string{
			domain.JobTypeSendOTP,
			domain.JobTypeSendPushReaction,
			domain.JobTypeSendPushComment,
			domain.JobTypeSendPushNewThread,
			domain.JobTypeSendPushChatMessage,
			domain.JobTypeSendPushScholarshipDeadline,
			domain.JobTypeSendPasswordReset,
		},
		pollInterval: 3 * time.Second,
	}
}

func (w *JobWorker) Run(ctx context.Context) {
	w.logger.Info("job worker started")
	for {
		select {
		case <-ctx.Done():
			w.logger.Info("job worker stopped")
			return
		default:
			job, err := w.jobRepo.Dequeue(ctx, w.jobTypes)
			if err != nil {
				w.logger.Error("dequeue error", "error", err)
				time.Sleep(w.pollInterval)
				continue
			}
			if job == nil {
				time.Sleep(w.pollInterval)
				continue
			}
			w.process(ctx, job)
		}
	}
}

func (w *JobWorker) process(ctx context.Context, job *domain.Job) {
	var execErr error

	switch job.Type {

	case domain.JobTypeSendOTP:
		var p domain.OTPJobPayload
		if err := json.Unmarshal(job.Payload, &p); err != nil {
			err := w.jobRepo.MarkFailed(ctx, job.ID, "bad payload: "+err.Error(), false)
			if err != nil {
				return
			}
			return
		}
		execErr = w.emailSvc.SendOTP(ctx, p.Email, p.Code)

	case domain.JobTypeSendPasswordReset:
		var p domain.PasswordResetJobPayload
		if err := json.Unmarshal(job.Payload, &p); err != nil {
			err := w.jobRepo.MarkFailed(ctx, job.ID, "bad payload: "+err.Error(), false)
			if err != nil {
				return
			}
			return
		}
		execErr = w.emailSvc.SendPasswordReset(ctx, p.Email, p.Code)

	default:
		w.logger.Warn("unknown job type", "type", job.Type, "id", job.ID)
		err := w.jobRepo.MarkFailed(ctx, job.ID, "unknown type", false)
		if err != nil {
			return
		}
		return
	}

	if execErr != nil {
		w.logger.Error("job failed", "type", job.Type, "id", job.ID,
			"attempt", job.Attempts, "error", execErr)
		err := w.jobRepo.MarkFailed(ctx, job.ID, execErr.Error(), true)
		if err != nil {
			return
		}
		return
	}

	err := w.jobRepo.MarkCompleted(ctx, job.ID)
	if err != nil {
		return
	}
	w.logger.Info("job completed", "type", job.Type, "id", job.ID)
}
