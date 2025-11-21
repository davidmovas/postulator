package schedule

import (
	"context"
	"errors"
	"math/rand"
	"strings"
	"time"

	"github.com/davidmovas/postulator/internal/domain/entities"
	"github.com/davidmovas/postulator/internal/domain/jobs"
	appErrors "github.com/davidmovas/postulator/pkg/errors"
	"github.com/davidmovas/postulator/pkg/logger"
)

const (
	TickerInterval = 1 * time.Minute
)

type Scheduler struct {
	jobRepo    jobs.Repository
	stateRepo  jobs.StateRepository
	executor   jobs.Executor
	calculator *Calculator
	logger     *logger.Logger
	stopChan   chan struct{}
	running    bool
}

func NewScheduler(
	jobRepo jobs.Repository,
	stateRepo jobs.StateRepository,
	executor jobs.Executor,
	logger *logger.Logger,
) jobs.Scheduler {
	return &Scheduler{
		jobRepo:    jobRepo,
		stateRepo:  stateRepo,
		executor:   executor,
		calculator: NewCalculator(),
		logger:     logger,
		stopChan:   make(chan struct{}),
	}
}

func (s *Scheduler) Start(ctx context.Context) error {
	s.logger.Info("Starting job scheduler")

	if err := s.RestoreState(ctx); err != nil {
		s.logger.Errorf("Failed to restore scheduler state: %v", err)
		return appErrors.Scheduler(err)
	}

	s.running = true
	go s.run(ctx)

	s.logger.Info("Job scheduler started successfully")
	return nil
}

func (s *Scheduler) Stop() error {
	s.logger.Info("Stopping job scheduler")
	s.running = false
	close(s.stopChan)
	return nil
}

func (s *Scheduler) run(ctx context.Context) {
	ticker := time.NewTicker(TickerInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			s.logger.Info("Scheduler stopped")
			return
		case <-ticker.C:
			s.checkAndExecuteDueJobs(ctx)
		}
	}
}

func (s *Scheduler) checkAndExecuteDueJobs(ctx context.Context) {
	now := time.Now()
	dueJobs, err := s.jobRepo.GetDue(ctx, now)
	if err != nil {
		s.logger.Errorf("Failed to get due jobs: %v", err)
		return
	}

	if len(dueJobs) == 0 {
		return
	}

	s.logger.Infof("Found %d due jobs to execute", len(dueJobs))

	for _, job := range dueJobs {
		go s.executeAndReschedule(ctx, job)
	}
}

func (s *Scheduler) executeAndReschedule(_ context.Context, job *entities.Job) {
	s.logger.Infof("Executing job %d (%s)", job.ID, job.Name)

	execCtx, cancel := context.WithTimeout(context.Background(), time.Minute*10)
	defer cancel()

	executionStart := time.Now()

	err := s.executor.Execute(execCtx, job)

	state := job.State
	if state == nil {
		state = &entities.State{JobID: job.ID}
	}

	state.LastRunAt = &executionStart

	if err != nil {
		s.logger.Errorf("Job %d execution failed: %v", job.ID, err)

		if isNoTopicsError(err) {
			s.logger.Warnf("No available topics for job %d, pausing job", job.ID)
			job.Status = entities.JobStatusPaused
			state.NextRunAt = nil
			state.NextRunBase = nil

			if updateErr := s.jobRepo.Update(execCtx, job); updateErr != nil {
				s.logger.Errorf("Failed to pause job %d: %v", job.ID, updateErr)
			}

			if updateErr := s.stateRepo.Update(execCtx, state); updateErr != nil {
				s.logger.Errorf("Failed to update state for job %d: %v", job.ID, updateErr)
			}

			return
		}

		if updateErr := s.stateRepo.IncrementExecutions(execCtx, job.ID, true); updateErr != nil {
			s.logger.Errorf("Failed to increment failed executions: %v", updateErr)
		}
	} else {
		if updateErr := s.stateRepo.IncrementExecutions(execCtx, job.ID, false); updateErr != nil {
			s.logger.Errorf("Failed to increment successful executions: %v", updateErr)
		}
	}

	if job.Status == entities.JobStatusActive && job.Schedule != nil && job.Schedule.Type != entities.ScheduleManual {
		if job.Schedule.Type == entities.ScheduleOnce {
			job.Status = entities.JobStatusCompleted
			state.NextRunAt = nil
			state.NextRunBase = nil

			if updateErr := s.jobRepo.Update(execCtx, job); updateErr != nil {
				s.logger.Errorf("Failed to complete job %d: %v", job.ID, updateErr)
			}

			if updateErr := s.stateRepo.Update(execCtx, state); updateErr != nil {
				s.logger.Errorf("Failed to update state for completed job %d: %v", job.ID, updateErr)
			}

			if deleteErr := s.jobRepo.Delete(execCtx, job.ID); deleteErr != nil {
				s.logger.Errorf("Failed to delete completed job %d: %v", job.ID, deleteErr)
			}
		} else {
			baseTime, withJitter, calcErr := s.calculator.CalculateNextRun(job, state.LastRunAt)
			if calcErr != nil {
				s.logger.Errorf("Failed to calculate next run for job %d: %v", job.ID, calcErr)
			} else {
				state.NextRunBase = &baseTime
				state.NextRunAt = &withJitter

				jitterDiff := withJitter.Sub(baseTime)
				s.logger.Infof("Job %d scheduled: base=%s, withJitter=%s, jitter=%v",
					job.ID, baseTime.Format("15:04:05"), withJitter.Format("15:04:05"), jitterDiff)

				if updateErr := s.stateRepo.Update(execCtx, state); updateErr != nil {
					s.logger.Errorf("Failed to update next run for job %d: %v", job.ID, updateErr)
				}
			}
		}
	}
}

func (s *Scheduler) RestoreState(ctx context.Context) error {
	s.logger.Info("Restoring scheduler state")

	activeJobs, err := s.jobRepo.GetActive(ctx)
	if err != nil {
		return appErrors.Scheduler(err)
	}

	now := time.Now()
	restoredCount := 0
	missedCount := 0

	for _, job := range activeJobs {
		var state *entities.State
		state, err = s.stateRepo.Get(ctx, job.ID)
		if err != nil {
			s.logger.Warnf("Failed to get state for job %d: %v", job.ID, err)
			state = &entities.State{JobID: job.ID}
		}

		job.State = state

		if job.Schedule == nil || job.Schedule.Type == entities.ScheduleManual {
			continue
		}

		needsReschedule := false

		if state.NextRunAt == nil {
			needsReschedule = true
		} else if state.NextRunAt.Before(now) {
			missedCount++
			delay := time.Duration(rand.Intn(300)) * time.Second
			nextRun := now.Add(delay)
			state.NextRunAt = &nextRun

			s.logger.Warnf("Job %d (%s) missed execution, rescheduling with %v delay",
				job.ID, job.Name, delay)

			needsReschedule = true
		}

		if needsReschedule {
			if state.NextRunBase == nil || state.NextRunAt.Before(now) {
				baseTime, withJitter, calcErr := s.calculator.CalculateNextRun(job, state.LastRunAt)
				if calcErr != nil {
					s.logger.Errorf("Failed to calculate next run for job %d: %v", job.ID, calcErr)
					continue
				}

				state.NextRunBase = &baseTime
				state.NextRunAt = &withJitter
			}

			if err = s.stateRepo.Update(ctx, state); err != nil {
				s.logger.Errorf("Failed to update state for job %d during restore: %v", job.ID, err)
				continue
			}

			restoredCount++
		}
	}

	s.logger.Infof("State restored: %d active jobs, %d jobs restored, %d missed executions rescheduled",
		len(activeJobs), restoredCount, missedCount)

	return nil
}

func (s *Scheduler) CalculateNextRun(job *entities.Job, lastRun *time.Time) (baseTime time.Time, withJitter time.Time, err error) {
	return s.calculator.CalculateNextRun(job, lastRun)
}

func (s *Scheduler) ScheduleJob(ctx context.Context, job *entities.Job) error {
	if job.Schedule != nil {
		if err := ValidateSchedule(job.Schedule); err != nil {
			return err
		}
	}

	if job.Schedule != nil && job.Schedule.Type != entities.ScheduleManual {
		baseTime, withJitter, err := s.calculator.CalculateNextRun(job, nil)
		if err != nil {
			return err
		}

		state := &entities.State{
			JobID:       job.ID,
			NextRunBase: &baseTime,
			NextRunAt:   &withJitter,
		}

		if err = s.stateRepo.Update(ctx, state); err != nil {
			return appErrors.Scheduler(err)
		}

		s.logger.Infof("Job %d scheduled: first run at %s (base: %s, jitter: %v)",
			job.ID, withJitter.Format("2006-01-02 15:04:05"),
			baseTime.Format("2006-01-02 15:04:05"),
			withJitter.Sub(baseTime))
	}

	job.Status = entities.JobStatusActive

	if err := s.jobRepo.Update(ctx, job); err != nil {
		return appErrors.Scheduler(err)
	}

	return nil
}

func (s *Scheduler) TriggerJob(ctx context.Context, jobID int64) error {
	job, err := s.jobRepo.GetByID(ctx, jobID)
	if err != nil {
		return err
	}

	if job.Status != entities.JobStatusActive {
		return appErrors.Validation("job is not active")
	}

	state, err := s.stateRepo.Get(ctx, jobID)
	if err != nil {
		return err
	}

	job.State = state

	s.executeAndReschedule(ctx, job)

	return nil
}

func isNoTopicsError(err error) bool {
	if err == nil {
		return false
	}

	var appErr *appErrors.AppError
	if errors.As(err, &appErr) {
		if strings.Contains(appErr.Message, "topics") {
			return true
		}
	}

	return false
}
