package execution

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/davidmovas/postulator/internal/domain/entities"
	"github.com/davidmovas/postulator/internal/infra/ai"
	"github.com/davidmovas/postulator/pkg/errors"
)

type PipelineStep string

const (
	StepInitialize      PipelineStep = "initialize"
	StepValidate        PipelineStep = "validate"
	StepSelectTopic     PipelineStep = "select_topic"
	StepSelectCategory  PipelineStep = "select_category"
	StepCreateExecution PipelineStep = "create_execution"
	StepRenderPrompt    PipelineStep = "render_prompt"
	StepGenerateAI      PipelineStep = "generate_ai"
	StepValidateOutput  PipelineStep = "validate_output"
	StepPublish         PipelineStep = "publish"
	StepMarkUsed        PipelineStep = "mark_used"
	StepComplete        PipelineStep = "complete"
)

type pipelineContext struct {
	Job       *entities.Job
	Execution *entities.Execution
	Site      *entities.Site
	Topic     *entities.Topic
	Category  *entities.Category
	Prompt    *entities.Prompt
	Provider  *entities.Provider

	SystemPrompt string
	UserPrompt   string

	GeneratedTitle   string
	GeneratedContent string

	Article *entities.Article

	StartTime time.Time
}

func (e *Executor) executePipeline(ctx context.Context, job *entities.Job) error {
	pctx := &pipelineContext{
		Job:       job,
		StartTime: time.Now(),
	}

	steps := []struct {
		name PipelineStep
		fn   func(context.Context, *pipelineContext) error
	}{
		{StepInitialize, e.stepInitialize},
		{StepValidate, e.stepValidate},
		{StepSelectTopic, e.stepSelectTopic},
		{StepSelectCategory, e.stepSelectCategory},
		{StepCreateExecution, e.stepCreateExecution},
		{StepRenderPrompt, e.stepRenderPrompt},
		{StepGenerateAI, e.stepGenerateAI},
		{StepValidateOutput, e.stepValidateOutput},
		{StepPublish, e.stepPublish},
		{StepMarkUsed, e.stepMarkUsed},
		{StepComplete, e.stepComplete},
	}

	for _, step := range steps {
		e.logger.Infof("Job %d: Executing step '%s'", job.ID, step.name)

		if err := step.fn(ctx, pctx); err != nil {
			e.logger.Errorf("Job %d: Step '%s' failed: %v", job.ID, step.name, err)
			return errors.JobExecution(job.ID, err).WithContext("step", string(step.name))
		}

		if pctx.Execution != nil && pctx.Execution.Status == entities.ExecutionStatusPendingValidation {
			e.logger.Infof("Job %d: Paused at step '%s' for validation", job.ID, step.name)
			return nil
		}
	}

	return nil
}

func (e *Executor) stepInitialize(_ context.Context, pctx *pipelineContext) error {
	if pctx.Job.State != nil {
		now := time.Now()
		pctx.Job.State.LastRunAt = &now
	}

	e.logger.Debugf("Job %d: Initialization complete", pctx.Job.ID)
	return nil
}

func (e *Executor) stepValidate(ctx context.Context, pctx *pipelineContext) error {
	site, err := e.siteService.GetSiteWithPassword(ctx, pctx.Job.SiteID)
	if err != nil {
		return fmt.Errorf("failed to get site: %w", err)
	}

	if site.Status != entities.StatusActive {
		return errors.Validation("site is not active").
			WithContext("site_status", string(site.Status))
	}

	pctx.Site = site

	if pctx.Job.Schedule == nil || pctx.Job.Schedule.Type != entities.ScheduleManual {
		if len(pctx.Job.Topics) == 0 {
			return errors.Validation("no topics assigned to job").
				WithContext("reason", "no_topics_available")
		}

		if len(pctx.Job.Categories) == 0 {
			return errors.Validation("no categories assigned to job")
		}
	}

	return nil
}

func (e *Executor) stepSelectTopic(ctx context.Context, pctx *pipelineContext) error {
	var topic *entities.Topic
	var err error

	switch pctx.Job.TopicStrategy {
	case entities.StrategyUnique:
		topic, err = e.topicService.GetNextTopicForJob(ctx, pctx.Job)
		if err != nil {
			return errors.JobExecution(pctx.Job.ID, err).
				WithContext("reason", "no_topics_available")
		}

	case entities.StrategyVariation:
		var originalTopic *entities.Topic
		originalTopic, err = e.topicService.GetNextTopicForJob(ctx, pctx.Job)
		if err != nil {
			return errors.JobExecution(pctx.Job.ID, err).
				WithContext("reason", "no_topics_available")
		}

		topic, err = e.topicService.GetOrGenerateVariation(ctx, pctx.Job.AIProviderID, pctx.Job.SiteID, originalTopic.ID)
		if err != nil {
			return fmt.Errorf("failed to generate topic variation: %w", err)
		}

	default:
		return errors.Validation("invalid topic strategy")
	}

	pctx.Topic = topic
	e.logger.Infof("Job %d: Selected topic %d (%s)", pctx.Job.ID, topic.ID, topic.Title)
	return nil
}

func (e *Executor) stepSelectCategory(ctx context.Context, pctx *pipelineContext) error {
	var categoryID int64

	switch pctx.Job.CategoryStrategy {
	case entities.CategoryFixed:
		if len(pctx.Job.Categories) == 0 {
			return errors.Validation("no categories assigned to job")
		}
		categoryID = pctx.Job.Categories[0]

	case entities.CategoryRandom:
		if len(pctx.Job.Categories) == 0 {
			return errors.Validation("no categories assigned to job")
		}
		categoryID = pctx.Job.Categories[e.randomIndex(len(pctx.Job.Categories))]

	case entities.CategoryRotate:
		if len(pctx.Job.Categories) == 0 {
			return errors.Validation("no categories assigned to job")
		}

		state := pctx.Job.State
		if state == nil {
			state = &entities.State{LastCategoryIndex: 0}
		}

		categoryID = pctx.Job.Categories[state.LastCategoryIndex]

		state.LastCategoryIndex = (state.LastCategoryIndex + 1) % len(pctx.Job.Categories)
		if err := e.stateRepo.UpdateCategoryIndex(ctx, pctx.Job.ID, state.LastCategoryIndex); err != nil {
			e.logger.Warnf("Failed to update category index: %v", err)
		}

	default:
		return errors.Validation("invalid category strategy")
	}

	category, err := e.categoryService.GetCategory(ctx, categoryID)
	if err != nil {
		return fmt.Errorf("failed to get category: %w", err)
	}

	pctx.Category = category
	e.logger.Infof("Job %d: Selected category %d (%s)", pctx.Job.ID, category.ID, category.Name)
	return nil
}

func (e *Executor) stepRenderPrompt(ctx context.Context, pctx *pipelineContext) error {
	if pctx.Site == nil {
		return errors.Validation("site not loaded")
	}
	if pctx.Topic == nil {
		return errors.Validation("topic not selected")
	}
	if pctx.Category == nil {
		return errors.Validation("category not selected")
	}

	prompt, err := e.promptService.GetPrompt(ctx, pctx.Job.PromptID)
	if err != nil {
		return fmt.Errorf("failed to get prompt: %w", err)
	}

	pctx.Prompt = prompt

	systemPrompt, userPrompt, err := e.promptService.RenderPrompt(ctx, prompt.ID, e.buildPlaceholders(pctx))
	if err != nil {
		return fmt.Errorf("failed to render prompt: %w", err)
	}

	pctx.SystemPrompt = systemPrompt
	pctx.UserPrompt = userPrompt

	e.logger.Debugf("Job %d: Rendered prompts (system: %d chars, user: %d chars)",
		pctx.Job.ID, len(systemPrompt), len(userPrompt))

	return nil
}

func (e *Executor) buildPlaceholders(pctx *pipelineContext) map[string]string {
	placeholders := make(map[string]string)

	for _, placeholder := range pctx.Prompt.Placeholders {
		placeholders[placeholder] = ""
	}

	placeholders["title"] = pctx.Topic.Title
	placeholders["siteName"] = pctx.Site.Name
	placeholders["siteUrl"] = pctx.Site.URL
	placeholders["category"] = pctx.Category.Name

	if pctx.Job.PlaceholdersValues != nil {
		for placeholder, value := range pctx.Job.PlaceholdersValues {
			placeholders[placeholder] = value
		}
	}

	return placeholders
}

func (e *Executor) stepGenerateAI(ctx context.Context, pctx *pipelineContext) error {
	pctx.Execution.Status = entities.ExecutionStatusGenerating
	if err := e.execRepo.Update(ctx, pctx.Execution); err != nil {
		return fmt.Errorf("failed to update execution status: %w", err)
	}

	provider, err := e.providerService.GetProvider(ctx, pctx.Job.AIProviderID)
	if err != nil {
		return fmt.Errorf("failed to get AI provider: %w", err)
	}

	pctx.Provider = provider
	pctx.Execution.AIModel = provider.Model

	aiClient, err := ai.CreateClient(provider)
	if err != nil {
		return fmt.Errorf("failed to create AI client: %w", err)
	}

	startTime := time.Now()

	result, err := aiClient.GenerateArticle(ctx, pctx.SystemPrompt, pctx.UserPrompt)
	if err != nil {
		return errors.AI(string(provider.Type), err)
	}

	generationTime := int(time.Since(startTime).Milliseconds())

	pctx.GeneratedTitle = result.Title
	pctx.GeneratedContent = result.Content

	now := time.Now()
	pctx.Execution.GeneratedAt = &now
	pctx.Execution.GenerationTimeMs = &generationTime

	if result.TokensUsed > 0 {
		pctx.Execution.TokensUsed = &result.TokensUsed
	}
	if result.Cost > 0 {
		pctx.Execution.CostUSD = &result.Cost
	}

	if err = e.execRepo.Update(ctx, pctx.Execution); err != nil {
		return fmt.Errorf("failed to update execution with generated content: %w", err)
	}

	e.logger.Infof("Job %d: Generated article (title: %s, content: %d chars, time: %dms)",
		pctx.Job.ID, pctx.GeneratedTitle, len(pctx.GeneratedContent), generationTime)

	return nil
}

func (e *Executor) stepValidateOutput(ctx context.Context, pctx *pipelineContext) error {
	if pctx.GeneratedTitle == "" {
		return errors.Validation("generated title is empty")
	}

	if pctx.GeneratedContent == "" {
		return errors.Validation("generated content is empty")
	}

	wordCount := len(strings.Fields(pctx.GeneratedContent))
	if wordCount < 100 {
		return errors.Validation("generated content is too short").
			WithContext("word_count", wordCount)
	}

	return nil
}

func (e *Executor) stepPublish(ctx context.Context, pctx *pipelineContext) error {
	// Always attempt to create a WP post. The publishArticle will send draft when RequiresValidation.
	pctx.Execution.Status = entities.ExecutionStatusPublishing
	if err := e.execRepo.Update(ctx, pctx.Execution); err != nil {
		return fmt.Errorf("failed to update execution status: %w", err)
	}

	article, err := e.publishArticle(ctx, pctx)
	if err != nil {
		return err
	}

	pctx.Article = article
	pctx.Execution.ArticleID = &article.ID

	if pctx.Job.RequiresValidation {
		// Pause pipeline for manual validation; article exists as WP draft.
		pctx.Execution.Status = entities.ExecutionStatusPendingValidation
		if err = e.execRepo.Update(ctx, pctx.Execution); err != nil {
			return fmt.Errorf("failed to update execution to pending validation: %w", err)
		}
		e.logger.Infof("Job %d: Article created as draft and awaiting validation (Article ID: %d, WP ID: %d)",
			pctx.Job.ID, article.ID, article.WPPostID)
		return nil
	}

	now := time.Now()
	pctx.Execution.PublishedAt = &now
	pctx.Execution.Status = entities.ExecutionStatusPublished

	if err = e.execRepo.Update(ctx, pctx.Execution); err != nil {
		return fmt.Errorf("failed to update execution after publication: %w", err)
	}

	e.logger.Infof("Job %d: Article published successfully (ID: %d, WP ID: %d)",
		pctx.Job.ID, article.ID, article.WPPostID)

	return nil
}

func (e *Executor) stepMarkUsed(ctx context.Context, pctx *pipelineContext) error {
	if pctx.Job.TopicStrategy != entities.StrategyUnique {
		return nil
	}

	if err := e.topicService.MarkTopicUsed(ctx, pctx.Job.SiteID, pctx.Topic.ID); err != nil {
		e.logger.Errorf("Failed to mark topic as used: %v", err)
		return nil
	}

	e.logger.Infof("Job %d: Marked topic %d as used", pctx.Job.ID, pctx.Topic.ID)
	return nil
}

func (e *Executor) stepComplete(ctx context.Context, pctx *pipelineContext) error {
	now := time.Now()
	pctx.Execution.CompletedAt = &now

	if err := e.execRepo.Update(ctx, pctx.Execution); err != nil {
		return fmt.Errorf("failed to update execution completion time: %w", err)
	}

	totalTime := time.Since(pctx.StartTime)
	e.logger.Infof("Job %d: Execution completed successfully in %v", pctx.Job.ID, totalTime)

	return nil
}

func (e *Executor) randomIndex(max int) int {
	return int(time.Now().UnixNano() % int64(max))
}

func (e *Executor) stepCreateExecution(ctx context.Context, pctx *pipelineContext) error {
	if pctx.Topic == nil || pctx.Category == nil {
		return errors.Validation("topic or category not selected")
	}

	provider, err := e.providerService.GetProvider(ctx, pctx.Job.AIProviderID)
	if err != nil {
		return fmt.Errorf("failed to get AI provider: %w", err)
	}
	pctx.Provider = provider

	exec := &entities.Execution{
		JobID:        pctx.Job.ID,
		SiteID:       pctx.Job.SiteID,
		TopicID:      pctx.Topic.ID,
		PromptID:     pctx.Job.PromptID,
		AIProviderID: pctx.Job.AIProviderID,
		AIModel:      provider.Model,
		CategoryID:   pctx.Category.ID,
		Status:       entities.ExecutionStatusPending,
		StartedAt:    time.Now(),
	}

	if err = e.execRepo.Create(ctx, exec); err != nil {
		return fmt.Errorf("failed to create execution record: %w", err)
	}

	pctx.Execution = exec
	e.logger.Infof("Job %d: Created execution %d", pctx.Job.ID, exec.ID)
	return nil
}
