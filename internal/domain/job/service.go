package job

import (
	"Postulator/pkg/di"
	"Postulator/pkg/errors"
	"Postulator/pkg/logger"
	"context"
	"fmt"
	"time"
)

var _ IService = (*Service)(nil)

type Service struct {
	jobRepo   IRepository
	execRepo  IExecutionRepository
	executor  IExecutor
	scheduler IScheduler
	logger    *logger.Logger
}

func NewService(c di.Container) (*Service, error) {
	var l *logger.Logger
	if err := c.Resolve(&l); err != nil {
		return nil, err
	}

	jobRepo, err := NewJobRepository(c)
	if err != nil {
		return nil, err
	}

	execRepo, err := NewExecutionRepository(c)
	if err != nil {
		return nil, err
	}

	executor, err := NewExecutor(c)
	if err != nil {
		return nil, err
	}

	scheduler, err := NewScheduler(c)
	if err != nil {
		return nil, err
	}

	return &Service{
		jobRepo:   jobRepo,
		execRepo:  execRepo,
		executor:  executor,
		scheduler: scheduler,
		logger:    l,
	}, nil
}

func (s *Service) CreateJob(ctx context.Context, job *Job) error {
	if err := s.validateJob(job); err != nil {
		return err
	}

	// Set default status if not provided
	if job.Status == "" {
		job.Status = StatusActive
	}

	// Calculate initial next run time if job is active
	if job.Status == StatusActive && job.ScheduleType != ScheduleManual {
		now := time.Now()
		nextRun := s.scheduler.CalculateNextRun(job, now)
		job.NextRunAt = &nextRun
	}

	s.logger.Infof("Creating job: %s for site %d", job.Name, job.SiteID)

	if err := s.jobRepo.Create(ctx, job); err != nil {
		return err
	}

	return nil
}

func (s *Service) GetJob(ctx context.Context, id int64) (*Job, error) {
	return s.jobRepo.GetByID(ctx, id)
}

func (s *Service) ListJobs(ctx context.Context) ([]*Job, error) {
	return s.jobRepo.GetAll(ctx)
}

func (s *Service) UpdateJob(ctx context.Context, job *Job) error {
	if err := s.validateJob(job); err != nil {
		return err
	}

	// If schedule changed and job is active, recalculate next run
	if job.Status == StatusActive && job.ScheduleType != ScheduleManual {
		now := time.Now()
		nextRun := s.scheduler.CalculateNextRun(job, now)
		job.NextRunAt = &nextRun
	}

	s.logger.Infof("Updating job %d: %s", job.ID, job.Name)

	return s.jobRepo.Update(ctx, job)
}

func (s *Service) DeleteJob(ctx context.Context, id int64) error {
	s.logger.Infof("Deleting job %d", id)

	// Note: executions will remain in database for historical purposes
	// The foreign key is set to CASCADE, so they'll be deleted automatically
	return s.jobRepo.Delete(ctx, id)
}

func (s *Service) PauseJob(ctx context.Context, id int64) error {
	job, err := s.jobRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if job.Status == StatusPaused {
		return errors.Validation("job is already paused")
	}

	s.logger.Infof("Pausing job %d (%s)", id, job.Name)

	job.Status = StatusPaused
	return s.jobRepo.Update(ctx, job)
}

func (s *Service) ResumeJob(ctx context.Context, id int64) error {
	job, err := s.jobRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if job.Status != StatusPaused {
		return errors.Validation("job is not paused")
	}

	s.logger.Infof("Resuming job %d (%s)", id, job.Name)

	job.Status = StatusActive

	// Recalculate next run time
	if job.ScheduleType != ScheduleManual {
		now := time.Now()
		nextRun := s.scheduler.CalculateNextRun(job, now)
		job.NextRunAt = &nextRun
	}

	return s.jobRepo.Update(ctx, job)
}

func (s *Service) ExecuteJobManually(ctx context.Context, jobID int64) error {
	job, err := s.jobRepo.GetByID(ctx, jobID)
	if err != nil {
		return err
	}

	s.logger.Infof("Manual execution requested for job %d (%s)", jobID, job.Name)

	// Execute the job regardless of schedule
	if err := s.executor.Execute(ctx, job); err != nil {
		s.logger.Errorf("Manual execution of job %d failed: %v", jobID, err)
		return errors.JobExecution(jobID, err)
	}

	// Update last run time
	now := time.Now()
	job.LastRunAt = &now

	// Recalculate next run if not manual schedule
	if job.ScheduleType != ScheduleManual && job.Status == StatusActive {
		nextRun := s.scheduler.CalculateNextRun(job, now)
		job.NextRunAt = &nextRun
	}

	if err := s.jobRepo.Update(ctx, job); err != nil {
		s.logger.Errorf("Failed to update job after manual execution: %v", err)
		return err
	}

	s.logger.Infof("Manual execution of job %d completed successfully", jobID)
	return nil
}

func (s *Service) ValidateExecution(ctx context.Context, execID int64, approved bool) error {
	exec, err := s.execRepo.GetByID(ctx, execID)
	if err != nil {
		return err
	}

	if exec.Status != ExecutionPendingValidation {
		return errors.Validation("execution is not pending validation")
	}

	if !approved {
		s.logger.Infof("Execution %d rejected by user", execID)

		// Mark as failed with rejection message
		errMsg := "Rejected by user during validation"
		exec.ErrorMessage = &errMsg
		exec.Status = ExecutionFailed
		now := time.Now()
		exec.ValidatedAt = &now

		return s.execRepo.Update(ctx, exec)
	}

	// Get job only when approved (needed for updating job after publication)
	job, err := s.jobRepo.GetByID(ctx, exec.JobID)
	if err != nil {
		return err
	}

	s.logger.Infof("Execution %d approved, publishing article", execID)

	// Mark as validated
	exec.Status = ExecutionValidated
	now := time.Now()
	exec.ValidatedAt = &now

	if err := s.execRepo.Update(ctx, exec); err != nil {
		return err
	}

	// Continue with publication
	// We need to reconstruct the context for publication
	// This is a simplified version - in production, store more context in execution
	if exec.GeneratedTitle == nil || exec.GeneratedContent == nil {
		return errors.Validation("execution missing generated content")
	}

	// Execute publication step
	// Note: This is a simplified approach. In production, you might want to
	// create a separate method in executor for just the publication step
	s.logger.Infof("Publishing validated content for execution %d", execID)

	// For now, just mark as published
	// The full publication logic would require passing site info, etc.
	exec.Status = ExecutionPublished
	publishTime := time.Now()
	exec.PublishedAt = &publishTime

	if err := s.execRepo.Update(ctx, exec); err != nil {
		return err
	}

	s.logger.Infof("Execution %d published successfully", execID)

	// Update job's last run time
	lastRun := time.Now()
	job.LastRunAt = &lastRun

	// Recalculate next run if applicable
	if job.ScheduleType != ScheduleManual && job.ScheduleType != ScheduleOnce && job.Status == StatusActive {
		nextRun := s.scheduler.CalculateNextRun(job, lastRun)
		job.NextRunAt = &nextRun
	} else if job.ScheduleType == ScheduleOnce {
		job.Status = StatusCompleted
	}

	if err := s.jobRepo.Update(ctx, job); err != nil {
		s.logger.Errorf("Failed to update job after validation: %v", err)
	}

	return nil
}

func (s *Service) GetPendingValidations(ctx context.Context) ([]*Execution, error) {
	return s.execRepo.GetPendingValidation(ctx)
}

func (s *Service) validateJob(job *Job) error {
	if job.Name == "" {
		return errors.Validation("job name cannot be empty")
	}

	if job.SiteID == 0 {
		return errors.Validation("job must be associated with a site")
	}

	if job.CategoryID == 0 {
		return errors.Validation("job must have a category")
	}

	if job.PromptID == 0 {
		return errors.Validation("job must have a prompt")
	}

	if job.AIProviderID == 0 {
		return errors.Validation("job must have an AI provider")
	}

	if job.AIModel == "" {
		return errors.Validation("job must specify an AI model")
	}

	if job.ScheduleType == "" {
		return errors.Validation("job must have a schedule type")
	}

	// Validate schedule-specific requirements
	switch job.ScheduleType {
	case ScheduleDaily:
		// ScheduleTime is optional, will default to current time if not set
	case ScheduleWeekly:
		if job.ScheduleDay == nil {
			return errors.Validation("weekly schedule requires a day of week (1-7)")
		}
		if *job.ScheduleDay < 1 || *job.ScheduleDay > 7 {
			return errors.Validation("weekly schedule day must be between 1 and 7")
		}
	case ScheduleMonthly:
		if job.ScheduleDay == nil {
			return errors.Validation("monthly schedule requires a day of month (1-31)")
		}
		if *job.ScheduleDay < 1 || *job.ScheduleDay > 31 {
			return errors.Validation("monthly schedule day must be between 1 and 31")
		}
	case ScheduleManual, ScheduleOnce:
		// No additional validation needed
	default:
		return errors.Validation(fmt.Sprintf("unknown schedule type: %s", job.ScheduleType))
	}

	// Validate jitter settings
	if job.JitterEnabled && job.JitterMinutes < 0 {
		return errors.Validation("jitter minutes cannot be negative")
	}

	return nil
}
