package execution

import (
	"context"
	"fmt"
	"time"

	"github.com/davidmovas/postulator/internal/domain/entities"
	"github.com/davidmovas/postulator/internal/domain/topics"
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
	Job            *entities.Job
	Execution      *entities.Execution
	Site           *entities.Site
	OriginalTopic  *entities.Topic
	VariationTopic *entities.Topic
	Category       *entities.Category
	Prompt         *entities.Prompt
	Provider       *entities.Provider

	Strategy topics.TopicStrategyHandler

	SystemPrompt string
	UserPrompt   string

	GeneratedTitle   string
	GeneratedExcerpt string
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
			if errors.IsNoResources(err) {
				e.logger.Infof("Job %d: Pipeline stopped gracefully due to no resources (step='%s')", job.ID, step.name)
				return nil
			}
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

func (e *Executor) stepInitialize(ctx context.Context, pctx *pipelineContext) error {
	now := time.Now()

	if pctx.Job.State == nil {
		st := &entities.State{JobID: pctx.Job.ID, LastRunAt: &now}
		if err := e.stateRepo.Update(ctx, st); err != nil {
			e.logger.Warnf("Job %d: Failed to persist initial LastRunAt: %v", pctx.Job.ID, err)
		} else {
			pctx.Job.State = st
		}
	} else {
		pctx.Job.State.LastRunAt = &now
		if err := e.stateRepo.Update(ctx, pctx.Job.State); err != nil {
			e.logger.Warnf("Job %d: Failed to update LastRunAt: %v", pctx.Job.ID, err)
		}
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

	strat, err := e.topicService.GetStrategy(pctx.Job.TopicStrategy)
	if err != nil {
		return err
	}
	pctx.Strategy = strat

	if pctx.Job.Schedule == nil {
		if err = strat.CanExecute(ctx, pctx.Job); err != nil {
			if errors.IsNoResources(err) {
				if pauseErr := e.pauseJob(ctx, pctx.Job.ID); pauseErr != nil {
					e.logger.Warnf("Job %d: Failed to pause on no-resources: %v", pctx.Job.ID, pauseErr)
				} else {
					e.logger.Infof("Job %d: Paused due to no resources for strategy '%s'", pctx.Job.ID, pctx.Job.TopicStrategy)
				}
				return errors.NoResources("topics")
			}
			return err
		}
	}

	if len(pctx.Job.Categories) == 0 {
		return errors.Validation("no categories assigned to job")
	}

	return nil
}

func (e *Executor) stepSelectTopic(ctx context.Context, pctx *pipelineContext) error {
	if pctx.Strategy == nil {
		strat, err := e.topicService.GetStrategy(pctx.Job.TopicStrategy)
		if err != nil {
			return err
		}
		pctx.Strategy = strat
	}
	original, variation, err := pctx.Strategy.PickTopic(ctx, pctx.Job)
	if err != nil {
		return err
	}
	pctx.OriginalTopic = original
	pctx.VariationTopic = variation

	e.logger.Infof("Job %d: Selected topic %d (%s)", pctx.Job.ID, variation.ID, variation.Title)
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

func (e *Executor) stepCreateExecution(ctx context.Context, pctx *pipelineContext) error {
	if pctx.VariationTopic == nil || pctx.Category == nil {
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
		TopicID:      pctx.VariationTopic.ID,
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

func (e *Executor) stepRenderPrompt(ctx context.Context, pctx *pipelineContext) error {
	if pctx.Site == nil {
		return errors.Validation("site not loaded")
	}
	if pctx.VariationTopic == nil {
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

	placeholders["title"] = pctx.VariationTopic.Title
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
		if recordErr := e.statsRecorder.RecordArticleFailed(ctx, pctx.Site.ID); recordErr != nil {
			e.logger.Warnf("Job %d: Failed to record article failed stats: %v", pctx.Job.ID, recordErr)
		}
		return fmt.Errorf("failed to update execution status: %w", err)
	}

	provider, err := e.providerService.GetProvider(ctx, pctx.Job.AIProviderID)
	if err != nil {
		if recordErr := e.statsRecorder.RecordArticleFailed(ctx, pctx.Site.ID); recordErr != nil {
			e.logger.Warnf("Job %d: Failed to record article failed stats: %v", pctx.Job.ID, recordErr)
		}
		return fmt.Errorf("failed to get AI provider: %w", err)
	}

	pctx.Provider = provider
	pctx.Execution.AIModel = provider.Model

	aiClient, err := ai.CreateClient(provider)
	if err != nil {
		if recordErr := e.statsRecorder.RecordArticleFailed(ctx, pctx.Site.ID); recordErr != nil {
			e.logger.Warnf("Job %d: Failed to record article failed stats: %v", pctx.Job.ID, recordErr)
		}
		return fmt.Errorf("failed to create AI client: %w", err)
	}

	startTime := time.Now()

	result, err := aiClient.GenerateArticle(ctx, pctx.SystemPrompt, pctx.UserPrompt)
	if err != nil {
		if recordErr := e.statsRecorder.RecordArticleFailed(ctx, pctx.Site.ID); recordErr != nil {
			e.logger.Warnf("Job %d: Failed to record article failed stats: %v", pctx.Job.ID, recordErr)
		}
		return errors.AI(string(provider.Type), err)
	}

	generationTime := int(time.Since(startTime).Milliseconds())

	pctx.GeneratedTitle = result.Title
	pctx.GeneratedExcerpt = result.Excerpt
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
		if recordErr := e.statsRecorder.RecordArticleFailed(ctx, pctx.Site.ID); recordErr != nil {
			e.logger.Warnf("Job %d: Failed to record article failed stats: %v", pctx.Job.ID, recordErr)
		}
		return fmt.Errorf("failed to update execution with generated content: %w", err)
	}

	e.logger.Infof("Job %d: Generated article (title: %s, content: %d chars, time: %dms)",
		pctx.Job.ID, pctx.GeneratedTitle, len(pctx.GeneratedContent), generationTime)

	return nil
}

func (e *Executor) stepValidateOutput(_ context.Context, pctx *pipelineContext) error {
	if pctx.GeneratedTitle == "" {
		return errors.Validation("generated title is empty")
	}

	if pctx.GeneratedContent == "" {
		return errors.Validation("generated content is empty")
	}

	return nil
}

func (e *Executor) stepPublish(ctx context.Context, pctx *pipelineContext) error {
	pctx.Execution.Status = entities.ExecutionStatusPublishing
	if err := e.execRepo.Update(ctx, pctx.Execution); err != nil {
		return fmt.Errorf("failed to update execution status: %w", err)
	}

	article, err := e.publishArticle(ctx, pctx)
	if err != nil {
		if recordErr := e.statsRecorder.RecordArticleFailed(ctx, pctx.Site.ID); recordErr != nil {
			e.logger.Warnf("Job %d: Failed to record article failed stats: %v", pctx.Job.ID, recordErr)
		}
		return err
	}

	pctx.Article = article
	pctx.Execution.ArticleID = &article.ID

	if pctx.Job.RequiresValidation {
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
		if recordErr := e.statsRecorder.RecordArticleFailed(ctx, pctx.Site.ID); recordErr != nil {
			e.logger.Warnf("Job %d: Failed to record article failed stats: %v", pctx.Job.ID, recordErr)
		}
		return fmt.Errorf("failed to update execution after publication: %w", err)
	}

	e.logger.Infof("Job %d: Article published successfully (ID: %d, WP ID: %d)",
		pctx.Job.ID, article.ID, article.WPPostID)

	return nil
}

func (e *Executor) stepMarkUsed(ctx context.Context, pctx *pipelineContext) error {
	if pctx.Strategy == nil || pctx.VariationTopic == nil {
		return nil
	}
	if err := pctx.Strategy.OnExecutionSuccess(ctx, pctx.Job, pctx.VariationTopic); err != nil {
		e.logger.Warnf("Job %d: Post-success hook failed: %v", pctx.Job.ID, err)
		return nil
	}

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

	if pctx.Job.Schedule == nil || pctx.Job.Schedule.Type != entities.ScheduleManual {
		if pctx.Strategy == nil {
			strat, err := e.topicService.GetStrategy(pctx.Job.TopicStrategy)
			if err != nil {
				e.logger.Warnf("Job %d: Failed to get strategy for post-run check: %v", pctx.Job.ID, err)
				return nil
			}
			pctx.Strategy = strat
		}

		if err := pctx.Strategy.CanExecute(ctx, pctx.Job); err != nil {
			if errors.IsNoResources(err) {
				if pauseErr := e.pauseJob(ctx, pctx.Job.ID); pauseErr != nil {
					e.logger.Warnf("Job %d: Failed to pause after resources exhausted: %v", pctx.Job.ID, pauseErr)
				} else {
					e.logger.Infof("Job %d: Paused after run due to no resources left for next execution (strategy='%s')", pctx.Job.ID, pctx.Job.TopicStrategy)
				}
				return nil
			}
			e.logger.Warnf("Job %d: Post-run resource check failed: %v", pctx.Job.ID, err)
		}
	}

	if err := e.statsRecorder.RecordArticlePublished(ctx, pctx.Site.ID, len(pctx.GeneratedContent)); err != nil {
		e.logger.Warnf("Job %d: Failed to record article published stats: %v", pctx.Job.ID, err)
	}

	return nil
}

func (e *Executor) randomIndex(max int) int {
	return int(time.Now().UnixNano() % int64(max))
}
