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

	if job.Status == "" {
		job.Status = StatusActive
	}

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

	if err = s.executor.Execute(ctx, job); err != nil {
		s.logger.Errorf("Manual execution of job %d failed: %v", jobID, err)
		return errors.JobExecution(jobID, err)
	}

	now := time.Now()
	job.LastRunAt = &now

	if job.ScheduleType != ScheduleManual && job.Status == StatusActive {
		nextRun := s.scheduler.CalculateNextRun(job, now)
		job.NextRunAt = &nextRun
	}

	if err = s.jobRepo.Update(ctx, job); err != nil {
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

		errMsg := "Rejected by user during validation"
		exec.ErrorMessage = &errMsg
		exec.Status = ExecutionFailed
		now := time.Now()
		exec.ValidatedAt = &now

		return s.execRepo.Update(ctx, exec)
	}

	job, err := s.jobRepo.GetByID(ctx, exec.JobID)
	if err != nil {
		return err
	}

	s.logger.Infof("Execution %d approved, publishing article", execID)

	exec.Status = ExecutionValidated
	now := time.Now()
	exec.ValidatedAt = &now

	if err = s.execRepo.Update(ctx, exec); err != nil {
		return err
	}

	if exec.GeneratedTitle == nil || exec.GeneratedContent == nil {
		return errors.Validation("execution missing generated content")
	}

	s.logger.Infof("Publishing validated content for execution %d", execID)

	// Use executor to properly publish the article to WordPress and create article record
	if err := s.executor.PublishValidatedArticle(ctx, job, exec); err != nil {
		s.logger.Errorf("Failed to publish validated article: %v", err)

		// Mark execution as failed
		errMsg := fmt.Sprintf("Failed to publish validated article: %v", err)
		exec.ErrorMessage = &errMsg
		exec.Status = ExecutionFailed
		if updateErr := s.execRepo.Update(ctx, exec); updateErr != nil {
			s.logger.Errorf("Failed to update execution after publish error: %v", updateErr)
		}

		return err
	}

	s.logger.Infof("Execution %d published successfully", execID)

	lastRun := time.Now()
	job.LastRunAt = &lastRun

	if job.ScheduleType != ScheduleManual && job.ScheduleType != ScheduleOnce && job.Status == StatusActive {
		nextRun := s.scheduler.CalculateNextRun(job, lastRun)
		job.NextRunAt = &nextRun
	} else if job.ScheduleType == ScheduleOnce {
		job.Status = StatusCompleted
	}

	if err = s.jobRepo.Update(ctx, job); err != nil {
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

	switch job.ScheduleType {
	case ScheduleDaily:
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
	default:
		return errors.Validation(fmt.Sprintf("unknown schedule type: %s", job.ScheduleType))
	}

	if job.JitterEnabled && job.JitterMinutes < 0 {
		return errors.Validation("jitter minutes cannot be negative")
	}

	return nil
}
