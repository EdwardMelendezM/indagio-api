package worker

import (
	"context"
	"log/slog"
	"time"
)

type JobRunner struct {
	job      PeriodicJob
	interval time.Duration
	timeout  time.Duration
	logger   *slog.Logger
}

func NewJobRunner(job PeriodicJob, interval, timeout time.Duration, logger *slog.Logger) *JobRunner {
	return &JobRunner{
		job:      job,
		interval: interval,
		timeout:  timeout,
		logger:   logger,
	}
}

func (r *JobRunner) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(r.interval)
		defer ticker.Stop()

		r.logger.Info("Job started", "name", r.job.Name(), "interval", r.interval)

		for {
			select {
			case <-ticker.C:
				jobCtx, cancel := context.WithTimeout(ctx, r.timeout)
				if err := r.job.Execute(jobCtx); err != nil {
					r.logger.Error("Job failed", "name", r.job.Name(), "error", err)
				}
				cancel()
			case <-ctx.Done():
				r.logger.Info("Stopping job...", "name", r.job.Name())
				return
			}
		}
	}()
}
