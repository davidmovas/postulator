package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/davidmovas/postulator/internal/domain/articles"
	"github.com/davidmovas/postulator/internal/domain/categories"
	"github.com/davidmovas/postulator/internal/domain/entities"
	"github.com/davidmovas/postulator/internal/domain/prompts"
	"github.com/davidmovas/postulator/internal/domain/providers"
	"github.com/davidmovas/postulator/internal/domain/sites"
	"github.com/davidmovas/postulator/internal/domain/topics"
	"github.com/davidmovas/postulator/pkg/errors"
	"github.com/davidmovas/postulator/pkg/logger"
)

var _ Service = (*service)(nil)

type service struct {
	scheduler       Scheduler
	executor        Executor
	siteService     sites.Service
	topicService    topics.Service
	promptService   prompts.Service
	providerService providers.Service
	categoryService categories.Service
	articleService  articles.Service
	repo            Repository
	stateRepo       StateRepository
	logger          *logger.Logger
}

func NewService(
	scheduler Scheduler,
	executor Executor,
	siteService sites.Service,
	topicService topics.Service,
	promptService prompts.Service,
	providerService providers.Service,
	categoryService categories.Service,
	articleService articles.Service,
	repo Repository,
	stateRepo StateRepository,
	logger *logger.Logger,
) Service {
	return &service{
		scheduler:       scheduler,
		executor:        executor,
		siteService:     siteService,
		topicService:    topicService,
		promptService:   promptService,
		providerService: providerService,
		categoryService: categoryService,
		articleService:  articleService,
		repo:            repo,
		stateRepo:       stateRepo,
		logger:          logger.WithScope("service").WithScope("jobs"),
	}
}

func (s *service) CreateJob(ctx context.Context, job *entities.Job) error {
	if err := s.validateJob(job); err != nil {
		return err
	}

	if err := s.validateDependencies(ctx, job); err != nil {
		return err
	}

	now := time.Now()
	job.CreatedAt = now
	job.UpdatedAt = now

	if err := s.repo.Create(ctx, job); err != nil {
		s.logger.ErrorWithErr(err, "Failed to create job")
		return err
	}

	if len(job.Categories) > 0 {
		if err := s.repo.SetCategories(ctx, job.ID, job.Categories); err != nil {
			s.logger.ErrorWithErr(err, "Failed to set job categories")
			return err
		}
	}

	if len(job.Topics) > 0 {
		if err := s.repo.SetTopics(ctx, job.ID, job.Topics); err != nil {
			s.logger.ErrorWithErr(err, "Failed to set job topics")
			return err
		}
	}

	if job.Schedule != nil && job.Schedule.Type != entities.ScheduleManual {
		nextRun, err := s.scheduler.CalculateNextRun(job, nil)
		if err != nil {
			s.logger.ErrorWithErr(err, "Failed to calculate next run")
			return err
		}

		if err = s.stateRepo.UpdateNextRun(ctx, job.ID, &nextRun); err != nil {
			s.logger.ErrorWithErr(err, "Failed to update next run")
			return err
		}
	}

	if err := s.scheduler.ScheduleJob(ctx, job); err != nil {
		s.logger.ErrorWithErr(err, "Failed to schedule job")
		return err
	}

	s.logger.Info("Job created successfully")
	return nil
}

func (s *service) GetJob(ctx context.Context, id int64) (*entities.Job, error) {
	job, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get job")
		return nil, err
	}

	state, err := s.stateRepo.Get(ctx, id)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get job state")
		return nil, err
	}
	job.State = state

	cats, err := s.repo.GetCategories(ctx, id)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get job categories")
		return nil, err
	}

	job.Categories = cats

	tops, err := s.repo.GetTopics(ctx, id)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get job topics")
		return nil, err
	}
	job.Topics = tops

	return job, nil
}

func (s *service) ListJobs(ctx context.Context) ([]*entities.Job, error) {
	jobs, err := s.repo.GetAll(ctx)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to list jobs")
		return nil, err
	}

	for _, job := range jobs {
		var state *entities.State
		state, err = s.stateRepo.Get(ctx, job.ID)
		if err != nil {
			s.logger.ErrorWithErr(err, "Failed to get job state")
			continue
		}
		job.State = state
	}

	return jobs, nil
}

func (s *service) UpdateJob(ctx context.Context, job *entities.Job) error {
	if err := s.validateJob(job); err != nil {
		return err
	}

	if err := s.validateDependencies(ctx, job); err != nil {
		return err
	}

	job.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, job); err != nil {
		s.logger.ErrorWithErr(err, "Failed to update job")
		return err
	}

	if job.Categories != nil {
		if err := s.repo.SetCategories(ctx, job.ID, job.Categories); err != nil {
			s.logger.ErrorWithErr(err, "Failed to update job categories")
			return err
		}
	}

	if job.Topics != nil {
		if err := s.repo.SetTopics(ctx, job.ID, job.Topics); err != nil {
			s.logger.ErrorWithErr(err, "Failed to update job topics")
			return err
		}
	}

	if err := s.scheduler.ScheduleJob(ctx, job); err != nil {
		s.logger.ErrorWithErr(err, "Failed to reschedule job")
		return err
	}

	return nil
}

func (s *service) DeleteJob(ctx context.Context, id int64) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		s.logger.ErrorWithErr(err, "Failed to delete job")
		return err
	}

	s.logger.Info("Job deleted successfully")
	return nil
}

func (s *service) PauseJob(ctx context.Context, id int64) error {
	job, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get job for pausing")
		return err
	}

	if job.Status == entities.JobStatusPaused {
		return nil
	}

	job.Status = entities.JobStatusPaused
	job.UpdatedAt = time.Now()

	if err = s.repo.Update(ctx, job); err != nil {
		s.logger.ErrorWithErr(err, "Failed to pause job")
		return err
	}

	s.logger.Info("Job paused successfully")
	return nil
}

func (s *service) ResumeJob(ctx context.Context, id int64) error {
	job, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get job for resuming")
		return err
	}

	if job.Status == entities.JobStatusActive {
		return nil
	}

	job.Status = entities.JobStatusActive
	job.UpdatedAt = time.Now()

	if err = s.repo.Update(ctx, job); err != nil {
		s.logger.ErrorWithErr(err, "Failed to resume job")
		return err
	}

	if err = s.scheduler.ScheduleJob(ctx, job); err != nil {
		s.logger.ErrorWithErr(err, "Failed to reschedule job")
		return err
	}

	return nil
}

func (s *service) ExecuteManually(ctx context.Context, jobID int64) error {
	job, err := s.repo.GetByID(ctx, jobID)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get job for manual execution")
		return err
	}

	if job.Status != entities.JobStatusActive {
		return errors.Validation("Job is not active")
	}

	if err = s.scheduler.TriggerJob(ctx, jobID); err != nil {
		s.logger.ErrorWithErr(err, "Failed to trigger job execution")
		return err
	}

	s.logger.Info("Job executed manually successfully")
	return nil
}

func (s *service) validateJob(job *entities.Job) error {
	if strings.TrimSpace(job.Name) == "" {
		return errors.Validation("Job name is required")
	}

	if job.SiteID <= 0 {
		return errors.Validation("Site ID is required")
	}

	if job.PromptID <= 0 {
		return errors.Validation("Prompt ID is required")
	}

	if job.AIProviderID <= 0 {
		return errors.Validation("AI Provider ID is required")
	}

	if validTopicStrategies := map[entities.TopicStrategy]bool{
		entities.StrategyUnique:    true,
		entities.StrategyVariation: true,
	}; !validTopicStrategies[job.TopicStrategy] {
		return errors.Validation("Invalid topic strategy")
	}

	if validCategoryStrategies := map[entities.CategoryStrategy]bool{
		entities.CategoryFixed:  true,
		entities.CategoryRandom: true,
		entities.CategoryRotate: true,
	}; !validCategoryStrategies[job.CategoryStrategy] {
		return errors.Validation("Invalid category strategy")
	}

	if job.Schedule != nil {
		if validScheduleTypes := map[entities.ScheduleType]bool{
			entities.ScheduleManual:   true,
			entities.ScheduleOnce:     true,
			entities.ScheduleInterval: true,
			entities.ScheduleDaily:    true,
		}; !validScheduleTypes[job.Schedule.Type] {
			return errors.Validation("Invalid schedule type")
		}

		if err := s.validateScheduleConfig(job.Schedule); err != nil {
			return err
		}
	} else {
		return errors.Validation("Schedule is required")
	}

	return nil
}

func (s *service) validateDependencies(ctx context.Context, job *entities.Job) error {
	if _, err := s.siteService.GetSite(ctx, job.SiteID); err != nil {
		return errors.Validation("Site does not exist")
	}

	if _, err := s.promptService.GetPrompt(ctx, job.PromptID); err != nil {
		return errors.Validation("Prompt does not exist")
	}

	if _, err := s.providerService.GetProvider(ctx, job.AIProviderID); err != nil {
		return errors.Validation("AI Provider does not exist")
	}

	for _, categoryID := range job.Categories {
		if _, err := s.categoryService.GetCategory(ctx, categoryID); err != nil {
			return errors.Validation("Category does not exist")
		}
	}

	for _, topicID := range job.Topics {
		if _, err := s.topicService.GetTopic(ctx, topicID); err != nil {
			return errors.Validation("Topic does not exist")
		}
	}

	return nil
}

func (s *service) validateScheduleConfig(schedule *entities.Schedule) error {
	if schedule == nil {
		return errors.Validation("Schedule is required")
	}

	switch schedule.Type {
	case entities.ScheduleOnce:
		var config entities.OnceSchedule
		if err := json.Unmarshal(schedule.Config, &config); err != nil {
			return errors.Validation("Invalid once schedule configuration")
		}

		fmt.Printf("[Schedule Once Config] Execute At: [%s]", config.ExecuteAt.String())

		if config.ExecuteAt.Before(time.Now()) {
			return errors.Validation("Execute at must be in the future")
		}

	case entities.ScheduleInterval:
		var config entities.IntervalSchedule
		if err := json.Unmarshal(schedule.Config, &config); err != nil {
			return errors.Validation("Invalid interval schedule configuration")
		}
		if config.Value <= 0 {
			return errors.Validation("Interval value must be positive")
		}
		validUnits := map[string]bool{"minutes": true, "hours": true, "days": true}
		if !validUnits[config.Unit] {
			return errors.Validation("Invalid interval unit")
		}

	case entities.ScheduleDaily:
		var config entities.DailySchedule
		if err := json.Unmarshal(schedule.Config, &config); err != nil {
			return errors.Validation("Invalid daily schedule configuration")
		}
		if config.Hour < 0 || config.Hour > 23 {
			return errors.Validation("Hour must be between 0 and 23")
		}
		if config.Minute < 0 || config.Minute > 59 {
			return errors.Validation("Minute must be between 0 and 59")
		}
		for _, day := range config.Weekdays {
			if day < 0 || day > 6 {
				return errors.Validation("Weekday must be between 0 and 6")
			}
		}
	}

	return nil
}
