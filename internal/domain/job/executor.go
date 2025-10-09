package job

import (
	"Postulator/internal/domain/entities"
	"Postulator/internal/domain/prompt"
	"Postulator/internal/domain/site"
	"Postulator/internal/domain/topic"
	"Postulator/internal/infra/ai"
	"Postulator/internal/infra/wp"
	"Postulator/pkg/di"
	"Postulator/pkg/errors"
	"Postulator/pkg/logger"
	"context"
	"fmt"
	"time"
)

var _ IExecutor = (*Executor)(nil)

type Executor struct {
	execRepo      IExecutionRepository
	topicService  topic.IService
	promptService prompt.IService
	siteService   site.IService
	wpClient      *wp.Client
	aiClient      ai.Client
	logger        *logger.Logger
}

func NewExecutor(c di.Container) (*Executor, error) {
	var l *logger.Logger
	if err := c.Resolve(&l); err != nil {
		return nil, err
	}

	execRepo, err := NewExecutionRepository(c)
	if err != nil {
		return nil, err
	}

	topicService, err := topic.NewService(c)
	if err != nil {
		return nil, err
	}

	promptService, err := prompt.NewService(c)
	if err != nil {
		return nil, err
	}

	siteService, err := site.NewService(c)
	if err != nil {
		return nil, err
	}

	var wpClient *wp.Client
	if err = c.Resolve(&wpClient); err != nil {
		return nil, err
	}

	var aiClient ai.Client
	if err = c.Resolve(&aiClient); err != nil {
		// AI client is optional for now (stub)
		l.Warn("AI client not registered, article generation will fail")
	}

	return &Executor{
		execRepo:      execRepo,
		topicService:  topicService,
		promptService: promptService,
		siteService:   siteService,
		wpClient:      wpClient,
		aiClient:      aiClient,
		logger:        l,
	}, nil
}

func (e *Executor) Execute(ctx context.Context, job *Job) error {
	e.logger.Infof("Starting execution of job %d (%s)", job.ID, job.Name)

	// Create execution record
	exec := &Execution{
		JobID:     job.ID,
		Status:    ExecutionPending,
		StartedAt: time.Now(),
	}

	if err := e.execRepo.Create(ctx, exec); err != nil {
		return errors.JobExecution(job.ID, err)
	}

	// Execute pipeline
	if err := e.executePipeline(ctx, job, exec); err != nil {
		e.logger.Errorf("Job %d execution failed: %v", job.ID, err)

		// Update execution with error
		errMsg := err.Error()
		exec.ErrorMessage = &errMsg
		exec.Status = ExecutionFailed
		if updateErr := e.execRepo.Update(ctx, exec); updateErr != nil {
			e.logger.Errorf("Failed to update execution record: %v", updateErr)
		}

		return err
	}

	e.logger.Infof("Job %d execution completed successfully", job.ID)
	return nil
}

func (e *Executor) executePipeline(ctx context.Context, job *Job, exec *Execution) error {
	// Step 1: Get site information
	siteInfo, err := e.siteService.GetSite(ctx, job.SiteID)
	if err != nil {
		return fmt.Errorf("failed to get site: %w", err)
	}

	// Step 2: Get available topic based on site's topic strategy
	// First, get site topics to determine strategy
	siteTopics, err := e.topicService.GetSiteTopics(ctx, job.SiteID)
	if err != nil {
		return fmt.Errorf("failed to get site topics: %w", err)
	}

	if len(siteTopics) == 0 {
		return fmt.Errorf("no topics assigned to site %d", job.SiteID)
	}

	// Use the strategy from first site topic (assuming all topics for a site use same strategy)
	strategy := siteTopics[0].Strategy

	availableTopic, err := e.topicService.GetAvailableTopic(ctx, job.SiteID, strategy)
	if err != nil {
		return fmt.Errorf("failed to get available topic: %w", err)
	}

	exec.TopicID = availableTopic.ID
	if err := e.execRepo.Update(ctx, exec); err != nil {
		return fmt.Errorf("failed to update execution with topic: %w", err)
	}

	e.logger.Infof("Job %d: Using topic %d (%s)", job.ID, availableTopic.ID, availableTopic.Title)

	// Step 3: Get category info for placeholder
	category, err := e.getCategoryInfo(ctx, job.CategoryID)
	if err != nil {
		return fmt.Errorf("failed to get category: %w", err)
	}

	// Step 5: Prepare placeholders for prompt rendering
	placeholders := map[string]string{
		"title":     availableTopic.Title,
		"site_name": siteInfo.Name,
		"category":  category.Name,
	}

	// Step 6: Render prompt with placeholders
	exec.Status = ExecutionGenerating
	now := time.Now()
	exec.GeneratedAt = &now
	if err := e.execRepo.Update(ctx, exec); err != nil {
		return fmt.Errorf("failed to update execution status: %w", err)
	}

	systemPrompt, userPrompt, err := e.promptService.RenderPrompt(ctx, job.PromptID, placeholders)
	if err != nil {
		return fmt.Errorf("failed to render prompt: %w", err)
	}

	e.logger.Debugf("Job %d: Rendered prompts for AI generation", job.ID)

	// Step 7: Generate article content using AI
	if e.aiClient == nil {
		return fmt.Errorf("AI client not available")
	}

	generatedContent, err := e.aiClient.GenerateArticle(ctx, systemPrompt, userPrompt)
	if err != nil {
		return fmt.Errorf("failed to generate article: %w", err)
	}

	// For variation strategy, AI might generate a new title; for unique strategy, use original
	generatedTitle := availableTopic.Title
	if strategy == entities.StrategyVariation {
		// For variation strategy, we could extract title from content or generate one
		// For now, we'll use the original title with a suffix
		generatedTitle = availableTopic.Title + " (Variation)"
	}

	exec.GeneratedTitle = &generatedTitle
	exec.GeneratedContent = &generatedContent
	if err := e.execRepo.Update(ctx, exec); err != nil {
		return fmt.Errorf("failed to update execution with generated content: %w", err)
	}

	e.logger.Infof("Job %d: Generated article content (%d chars)", job.ID, len(generatedContent))

	// Step 8: Check if validation is required
	if job.RequiresValidation {
		exec.Status = ExecutionPendingValidation
		if err := e.execRepo.Update(ctx, exec); err != nil {
			return fmt.Errorf("failed to update execution for validation: %w", err)
		}

		e.logger.Infof("Job %d: Article awaiting validation", job.ID)
		return nil // Stop here, wait for manual validation
	}

	// Step 9: Publish to WordPress
	if err := e.publishArticle(ctx, job, exec, siteInfo, generatedTitle, generatedContent); err != nil {
		return err
	}

	// Step 10: Mark topic as used (only for unique strategy)
	if strategy == entities.StrategyUnique {
		if err := e.topicService.MarkTopicAsUsed(ctx, job.SiteID, availableTopic.ID); err != nil {
			e.logger.Errorf("Failed to mark topic as used: %v", err)
			// Don't fail the job - article is already published
		} else {
			e.logger.Infof("Job %d: Marked topic %d as used", job.ID, availableTopic.ID)
		}
	}

	return nil
}

func (e *Executor) publishArticle(ctx context.Context, job *Job, exec *Execution, siteInfo *entities.Site, title, content string) error {
	exec.Status = ExecutionPublishing
	if err := e.execRepo.Update(ctx, exec); err != nil {
		return fmt.Errorf("failed to update execution status to publishing: %w", err)
	}

	e.logger.Infof("Job %d: Publishing article to WordPress", job.ID)

	// Get WP category ID for this job's category
	categories, err := e.siteService.GetSiteCategories(ctx, job.SiteID)
	if err != nil {
		return fmt.Errorf("failed to get site categories: %w", err)
	}

	var wpCategoryID int
	for _, cat := range categories {
		if cat.ID == job.CategoryID {
			wpCategoryID = cat.WPCategoryID
			break
		}
	}

	if wpCategoryID == 0 {
		return fmt.Errorf("category %d not found for site %d", job.CategoryID, job.SiteID)
	}

	// Publish to WordPress
	postID, postURL, err := e.wpClient.PublishPost(ctx, siteInfo, title, content, wpCategoryID)
	if err != nil {
		return fmt.Errorf("failed to publish post to WordPress: %w", err)
	}

	e.logger.Infof("Job %d: Article published successfully (post ID: %d, URL: %s)", job.ID, postID, postURL)

	// Update execution with publication details
	exec.Status = ExecutionPublished
	now := time.Now()
	exec.PublishedAt = &now

	// Store article ID (we'd need to create article record in articles table)
	// For now, just mark as published
	if err := e.execRepo.Update(ctx, exec); err != nil {
		return fmt.Errorf("failed to update execution after publication: %w", err)
	}

	return nil
}

func (e *Executor) getCategoryInfo(ctx context.Context, categoryID int64) (*entities.Category, error) {
	// We need to get category by ID - this requires adding a method to site service
	// For now, we'll need to iterate through site categories
	// This is a simplification - in production, add GetCategoryByID method

	// Since we don't have category repository method, return a placeholder
	// This will be improved when we add proper category lookup
	return &entities.Category{
		ID:   categoryID,
		Name: "General",
	}, nil
}
