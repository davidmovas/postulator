package execution

import (
	"context"
	"fmt"
	"time"

	"github.com/davidmovas/postulator/internal/domain/articles"
	"github.com/davidmovas/postulator/internal/domain/entities"
	"github.com/davidmovas/postulator/internal/domain/jobs"
	"github.com/davidmovas/postulator/internal/domain/providers"
	"github.com/davidmovas/postulator/internal/domain/sites"
	"github.com/davidmovas/postulator/pkg/errors"
	"github.com/davidmovas/postulator/pkg/logger"
)

var _ Service = (*service)(nil)

type service struct {
	repo            Repository
	jobService      jobs.Service
	articleService  articles.Service
	siteService     sites.Service
	providerService providers.Service
	logger          *logger.Logger
}

func NewService(
	repo Repository,
	jobService jobs.Service,
	articleService articles.Service,
	siteService sites.Service,
	providerService providers.Service,
	logger *logger.Logger,
) Service {
	return &service{
		repo:            repo,
		jobService:      jobService,
		articleService:  articleService,
		siteService:     siteService,
		providerService: providerService,
		logger:          logger.WithScope("service").WithScope("execution"),
	}
}

func (s *service) CreateExecution(ctx context.Context, exec *entities.Execution) error {
	if err := s.validateExecution(exec); err != nil {
		return err
	}

	if err := s.validateDependencies(ctx, exec); err != nil {
		return err
	}

	exec.StartedAt = time.Now()

	if err := s.repo.Create(ctx, exec); err != nil {
		s.logger.ErrorWithErr(err, "Failed to create execution")
		return err
	}

	return nil
}

func (s *service) GetExecution(ctx context.Context, id int64) (*entities.Execution, error) {
	exec, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get execution")
		return nil, err
	}

	return exec, nil
}

func (s *service) ListExecutions(ctx context.Context, jobID int64, limit, offset int) ([]*entities.Execution, int, error) {
	executions, amount, err := s.repo.GetByJobID(ctx, jobID, limit, offset)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to list executions")
		return nil, 0, err
	}

	return executions, amount, nil
}

func (s *service) GetPendingValidations(ctx context.Context) ([]*entities.Execution, error) {
	executions, err := s.repo.GetPendingValidation(ctx)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get pending validations")
		return nil, err
	}

	s.logger.Debug("Pending validations retrieved")
	return executions, nil
}

func (s *service) UpdateStatus(ctx context.Context, id int64, status entities.ExecutionStatus) error {
	exec, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get execution for status update")
		return err
	}

	if err = s.validateStatusTransition(exec.Status, status); err != nil {
		return err
	}

	exec.Status = status
	now := time.Now()

	switch status {
	case entities.ExecutionStatusGenerating:
		break
	case entities.ExecutionStatusPendingValidation:
		exec.GeneratedAt = &now
	case entities.ExecutionStatusValidated:
		exec.ValidatedAt = &now
	case entities.ExecutionStatusPublished:
		exec.PublishedAt = &now
		exec.CompletedAt = &now
	case entities.ExecutionStatusRejected, entities.ExecutionStatusFailed:
		exec.CompletedAt = &now
	}

	if err = s.repo.Update(ctx, exec); err != nil {
		s.logger.ErrorWithErr(err, "Failed to update execution status")
		return err
	}

	return nil
}

func (s *service) ApproveExecution(ctx context.Context, id int64) error {
	exec, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get execution for approval")
		return err
	}

	if exec.Status != entities.ExecutionStatusPendingValidation {
		return errors.Validation("Execution is not pending validation")
	}

	if err = s.UpdateStatus(ctx, id, entities.ExecutionStatusValidated); err != nil {
		return err
	}

	if exec.ArticleID != nil {
		if err = s.articleService.PublishDraft(ctx, *exec.ArticleID); err != nil {
			s.logger.ErrorWithErr(err, "Failed to publish article after approval")
			return err
		}
	}

	return nil
}

func (s *service) RejectExecution(ctx context.Context, id int64) error {
	exec, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get execution for rejection")
		return err
	}

	if exec.Status != entities.ExecutionStatusPendingValidation {
		return errors.Validation("Execution is not pending validation")
	}

	if err = s.UpdateStatus(ctx, id, entities.ExecutionStatusRejected); err != nil {
		return err
	}

	return nil
}

func (s *service) GetJobMetrics(ctx context.Context, jobID int64) (*entities.Metrics, error) {
	totalExecutions, err := s.repo.CountByJob(ctx, jobID)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to count executions for metrics")
		return nil, err
	}

	allExecutions, _, err := s.repo.GetByJobID(ctx, jobID, 10_000, 0)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get executions for metrics")
		return nil, err
	}

	metrics := &entities.Metrics{
		TotalExecutions: totalExecutions,
	}

	for _, exec := range allExecutions {
		switch exec.Status {
		case entities.ExecutionStatusPublished:
			metrics.SuccessfulExecutions++
		case entities.ExecutionStatusFailed:
			metrics.FailedExecutions++
		case entities.ExecutionStatusRejected:
			metrics.RejectedExecutions++
		}
	}

	avgTime, err := s.repo.GetAverageGenerationTime(ctx, jobID)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get average generation time")
		return nil, err
	}
	metrics.AverageTimeMs = avgTime

	totalTokens, err := s.repo.GetTotalTokens(ctx, time.Now().AddDate(0, -1, 0), time.Now())
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get total tokens")
		return nil, err
	}
	metrics.TotalTokens = totalTokens

	totalCost, err := s.repo.GetTotalCost(ctx, time.Now().AddDate(0, -1, 0), time.Now())
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get total cost")
		return nil, err
	}
	metrics.TotalCost = totalCost

	return metrics, nil
}

func (s *service) validateExecution(exec *entities.Execution) error {
	if exec.JobID <= 0 {
		return errors.Validation("Job ID is required")
	}

	if exec.SiteID <= 0 {
		return errors.Validation("Site ID is required")
	}

	if exec.TopicID <= 0 {
		return errors.Validation("Topic ID is required")
	}

	if exec.PromptID <= 0 {
		return errors.Validation("Prompt ID is required")
	}

	if exec.AIProviderID <= 0 {
		return errors.Validation("AI Provider ID is required")
	}

	if exec.CategoryID <= 0 {
		return errors.Validation("Category ID is required")
	}

	return nil
}

func (s *service) validateDependencies(ctx context.Context, exec *entities.Execution) error {
	if _, err := s.jobService.GetJob(ctx, exec.JobID); err != nil {
		return errors.Validation("Job does not exist")
	}

	if _, err := s.siteService.GetSite(ctx, exec.SiteID); err != nil {
		return errors.Validation("Site does not exist")
	}

	if _, err := s.providerService.GetProvider(ctx, exec.AIProviderID); err != nil {
		return errors.Validation("AI Provider does not exist")
	}

	return nil
}

func (s *service) validateStatusTransition(from, to entities.ExecutionStatus) error {
	validTransitions := map[entities.ExecutionStatus]map[entities.ExecutionStatus]bool{
		entities.ExecutionStatusPending: {
			entities.ExecutionStatusGenerating: true,
			entities.ExecutionStatusFailed:     true,
		},
		entities.ExecutionStatusGenerating: {
			entities.ExecutionStatusPendingValidation: true,
			entities.ExecutionStatusFailed:            true,
		},
		entities.ExecutionStatusPendingValidation: {
			entities.ExecutionStatusValidated: true,
			entities.ExecutionStatusRejected:  true,
			entities.ExecutionStatusFailed:    true,
		},
		entities.ExecutionStatusValidated: {
			entities.ExecutionStatusPublishing: true,
			entities.ExecutionStatusFailed:     true,
		},
		entities.ExecutionStatusPublishing: {
			entities.ExecutionStatusPublished: true,
			entities.ExecutionStatusFailed:    true,
		},
		entities.ExecutionStatusPublished: {},
		entities.ExecutionStatusRejected:  {},
		entities.ExecutionStatusFailed:    {},
	}

	if transitions, exists := validTransitions[from]; exists {
		if !transitions[to] {
			return errors.Validation(fmt.Sprintf("Invalid status transition from %s to %s", from, to))
		}
	}

	return nil
}
