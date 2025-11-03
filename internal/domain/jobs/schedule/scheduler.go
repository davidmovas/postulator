package schedule

import (
	"context"
	"errors"
	"math/rand"
	"strings"
	"time"

	"github.com/davidmovas/postulator/internal/domain/entities"
	"github.com/davidmovas/postulator/internal/domain/jobs"
	"github.com/davidmovas/postulator/internal/domain/jobs/execution"
	appErrors "github.com/davidmovas/postulator/pkg/errors"
	"github.com/davidmovas/postulator/pkg/logger"
)

const (
	TickerInterval = 1 * time.Minute
)

type Scheduler struct {
	jobRepo    jobs.Repository
	stateRepo  jobs.StateRepository
	executor   *execution.Executor
	calculator *Calculator
	logger     *logger.Logger
	stopChan   chan struct{}
	running    bool
}

func NewScheduler(
	jobRepo jobs.Repository,
	stateRepo jobs.StateRepository,
	executor *execution.Executor,
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

func (s *Scheduler) executeAndReschedule(ctx context.Context, job *entities.Job) {
	s.logger.Infof("Executing job %d (%s)", job.ID, job.Name)

	execCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	err := s.executor.Execute(execCtx, job)

	now := time.Now()
	state := job.State
	if state == nil {
		state = &entities.State{JobID: job.ID}
	}

	state.LastRunAt = &now

	if err != nil {
		s.logger.Errorf("Job %d execution failed: %v", job.ID, err)

		if isNoTopicsError(err) {
			s.logger.Warnf("No available topics for job %d, pausing job", job.ID)
			job.Status = entities.JobStatusPaused
			state.NextRunAt = nil

			if updateErr := s.jobRepo.Update(ctx, job); updateErr != nil {
				s.logger.Errorf("Failed to pause job %d: %v", job.ID, updateErr)
			}

			if updateErr := s.stateRepo.Update(ctx, state); updateErr != nil {
				s.logger.Errorf("Failed to update state for job %d: %v", job.ID, updateErr)
			}

			return
		}

		if updateErr := s.stateRepo.IncrementExecutions(ctx, job.ID, true); updateErr != nil {
			s.logger.Errorf("Failed to increment failed executions: %v", updateErr)
		}
	} else {
		if updateErr := s.stateRepo.IncrementExecutions(ctx, job.ID, false); updateErr != nil {
			s.logger.Errorf("Failed to increment successful executions: %v", updateErr)
		}
	}

	if job.Status == entities.JobStatusActive && job.Schedule != nil && job.Schedule.Type != entities.ScheduleManual {
		if job.Schedule.Type == entities.ScheduleOnce {
			job.Status = entities.JobStatusCompleted
			state.NextRunAt = nil

			if updateErr := s.jobRepo.Update(ctx, job); updateErr != nil {
				s.logger.Errorf("Failed to complete job %d: %v", job.ID, updateErr)
			}

			if updateErr := s.stateRepo.Update(ctx, state); updateErr != nil {
				s.logger.Errorf("Failed to update state for completed job %d: %v", job.ID, updateErr)
			}

			if deleteErr := s.jobRepo.Delete(ctx, job.ID); deleteErr != nil {
				s.logger.Errorf("Failed to delete completed job %d: %v", job.ID, deleteErr)
			}
		} else {
			nextRun, calcErr := s.calculator.CalculateNextRun(job, state.LastRunAt)
			if calcErr != nil {
				s.logger.Errorf("Failed to calculate next run for job %d: %v", job.ID, calcErr)
			} else {
				state.NextRunAt = &nextRun
				if updateErr := s.stateRepo.UpdateNextRun(ctx, job.ID, &nextRun); updateErr != nil {
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

		if state.NextRunAt == nil || state.NextRunAt.Before(now) {
			if state.NextRunAt != nil && state.NextRunAt.Before(now) {
				missedCount++
				delay := time.Duration(rand.Intn(300)) * time.Second
				nextRun := now.Add(delay)
				state.NextRunAt = &nextRun

				s.logger.Warnf("Job %d (%s) missed execution, rescheduling with %v delay",
					job.ID, job.Name, delay)
			} else {
				var nextRun time.Time
				nextRun, err = s.calculator.CalculateNextRun(job, state.LastRunAt)
				if err != nil {
					s.logger.Errorf("Failed to calculate next run for job %d: %v", job.ID, err)
					continue
				}
				state.NextRunAt = &nextRun
			}

			if err = s.stateRepo.UpdateNextRun(ctx, job.ID, state.NextRunAt); err != nil {
				s.logger.Errorf("Failed to update next run for job %d during state restore: %v", job.ID, err)
				continue
			}

			restoredCount++
		}
	}

	s.logger.Infof("State restored: %d active jobs, %d jobs restored, %d missed executions rescheduled",
		len(activeJobs), restoredCount, missedCount)

	return nil
}

func (s *Scheduler) CalculateNextRun(job *entities.Job, lastRun *time.Time) (time.Time, error) {
	return s.calculator.CalculateNextRun(job, lastRun)
}

func (s *Scheduler) ScheduleJob(ctx context.Context, job *entities.Job) error {
	if job.Schedule != nil {
		if err := ValidateSchedule(job.Schedule); err != nil {
			return err
		}
	}

	if job.Schedule != nil && job.Schedule.Type != entities.ScheduleManual {
		nextRun, err := s.calculator.CalculateNextRun(job, nil)
		if err != nil {
			return err
		}

		state := &entities.State{
			JobID:     job.ID,
			NextRunAt: &nextRun,
		}

		if err = s.stateRepo.Update(ctx, state); err != nil {
			return appErrors.Scheduler(err)
		}
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

	go s.executeAndReschedule(ctx, job)

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
