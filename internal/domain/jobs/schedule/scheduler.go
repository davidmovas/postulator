package schedule

import (
	"Postulator/internal/domain/jobs"
	"context"
	"time"
)

type Scheduler struct{}

func (s *Scheduler) Start(ctx context.Context) error {
	return nil
}

func (s *Scheduler) Stop() error {
	return nil
}

func (s *Scheduler) RestoreState(ctx context.Context) error {
	return nil
}

func (s *Scheduler) CalculateNextRun(job *jobs.Job, lastRun *time.Time) (time.Time, error) {
	return time.Time{}, nil
}

func (s *Scheduler) ScheduleJob(ctx context.Context, job *jobs.Job) error {
	return nil
}

func (s *Scheduler) TriggerJob(ctx context.Context, jobID int64) error {
	return nil
}
