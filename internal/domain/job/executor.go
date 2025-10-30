package job

import (
	"Postulator/internal/domain/article"
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
	"strings"
	"time"
)

type wpPublisher interface {
	PublishPost(ctx context.Context, site *entities.Site, title, content string, categoryID int) (postID int, postURL string, err error)
}

var _ IExecutor = (*Executor)(nil)

type Executor struct {
	execRepo      IExecutionRepository
	articleRepo   article.IRepository
	topicService  topic.IService
	promptService prompt.IService
	siteService   site.IService
	wpClient      wpPublisher
	aiClient      ai.IClient
	logger        *logger.Logger
}

func NewExecutor(c di.Container) (*Executor, error) {
	var l *logger.Logger
	if err := c.Resolve(&l); err != nil {
		return nil, err
	}

	var execRepo IExecutionRepository
	if err := c.Resolve(&execRepo); err != nil {
		return nil, err
	}

	var articleRepo article.IRepository
	if err := c.Resolve(&articleRepo); err != nil {
		return nil, err
	}

	var topicService topic.IService
	if err := c.Resolve(&topicService); err != nil {
		return nil, err
	}

	var promptService prompt.IService
	if err := c.Resolve(&promptService); err != nil {
		return nil, err
	}

	var siteService site.IService
	if err := c.Resolve(&siteService); err != nil {
		return nil, err
	}

	var wpClient *wp.Client
	if err := c.Resolve(&wpClient); err != nil {
		return nil, err
	}

	var aiClient ai.IClient
	if err := c.Resolve(&aiClient); err != nil {
		return nil, fmt.Errorf("AI client is required for job execution: %w", err)
	}

	return &Executor{
		execRepo:      execRepo,
		articleRepo:   articleRepo,
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

	// Create an execution record
	exec := &Execution{
		JobID:     job.ID,
		Status:    ExecutionPending,
		StartedAt: time.Now(),
	}

	if err := e.execRepo.Create(ctx, exec); err != nil {
		return errors.JobExecution(job.ID, err)
	}

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
	siteInfo, err := e.siteService.GetSiteWithPassword(ctx, job.SiteID)
	if err != nil {
		return errors.JobExecution(job.ID, fmt.Errorf("failed to get site: %w", err))
	}

	// Step 2: Get available topic based on site's topic strategy
	// First, get site topics to determine strategy
	siteTopics, err := e.topicService.GetSiteTopics(ctx, job.SiteID)
	if err != nil {
		return errors.JobExecution(job.ID, fmt.Errorf("failed to get site topics: %w", err))
	}

	if len(siteTopics) == 0 {
		return errors.JobExecution(job.ID, fmt.Errorf("no topics assigned to site %d", job.SiteID))
	}

	// Use the strategy from first site topic (assuming all topics for a site use same strategy)
	strategy := siteTopics[0].Strategy

	availableTopic, err := e.topicService.GetAvailableTopic(ctx, job.SiteID, strategy)
	if err != nil {
		return errors.JobExecutionWithNote(job.ID, "there is no available topic", fmt.Errorf("failed to get available topic: %w", err))
	}

	exec.TopicID = availableTopic.ID
	if err = e.execRepo.Update(ctx, exec); err != nil {
		return fmt.Errorf("failed to update execution with topic: %w", err)
	}

	e.logger.Infof("Job %d: Using topic %d (%s)", job.ID, availableTopic.ID, availableTopic.Title)

	// Step 3: Get category info for placeholder
	category, err := e.getCategoryInfo(ctx, job.CategoryID, job.SiteID)
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
	if err = e.execRepo.Update(ctx, exec); err != nil {
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

	// For variation strategy, use the topic title as-is
	// The AI should be prompted to generate variations in the content itself
	// Title variations can be added by using specific prompts that instruct the AI
	// to create unique angles or perspectives on the topic
	generatedTitle := availableTopic.Title

	exec.GeneratedTitle = &generatedTitle
	exec.GeneratedContent = &generatedContent
	if err = e.execRepo.Update(ctx, exec); err != nil {
		return fmt.Errorf("failed to update execution with generated content: %w", err)
	}

	e.logger.Infof("Job %d: Generated article content (%d chars)", job.ID, len(generatedContent))

	// Step 8: Check if validation is required
	if job.RequiresValidation {
		exec.Status = ExecutionPendingValidation
		if err = e.execRepo.Update(ctx, exec); err != nil {
			return fmt.Errorf("failed to update execution for validation: %w", err)
		}

		e.logger.Infof("Job %d: Article awaiting validation", job.ID)
		return nil // Stop here, wait for manual validation
	}

	// Step 9: Publish to WordPress
	if err = e.publishArticle(ctx, job, exec, siteInfo, generatedTitle, generatedContent); err != nil {
		return err
	}

	// Step 10: Mark topic as used (only for unique strategy)
	if strategy == entities.StrategyUnique {
		if err = e.topicService.MarkTopicAsUsed(ctx, job.SiteID, availableTopic.ID); err != nil {
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

	// TODO: TEMP CONTENT FOR TEST
	content = `
	<!-- wp:paragraph -->
<p>The pursuit of health is a fundamental human endeavor, and at the heart of this journey lies the powerful, symbiotic relationship between physical activity and overall well-being. Sport and exercise are not merely tools for aesthetic improvement; they are foundational pillars for a vibrant, functional, and fulfilling life. In our increasingly sedentary world, understanding and embracing the multifaceted benefits of an active lifestyle is more critical than ever.</p>
<!-- /wp:paragraph -->

<!-- wp:paragraph -->
<p>The impact extends far beyond the physical, weaving into the very fabric of our mental and emotional resilience.</p>
<!-- /wp:paragraph -->

<!-- wp:heading -->
<h2>Physical Health Benefits</h2>
<!-- /wp:heading -->

<!-- wp:paragraph -->
<p>Physically, the advantages of regular sport participation are profound and well-documented. The most obvious benefit is the improvement in cardiovascular health. Engaging in activities like running, swimming, or cycling strengthens the heart muscle, making it more efficient at pumping blood throughout the body. This enhanced efficiency lowers resting heart rate and blood pressure, significantly reducing the risk of heart disease, stroke, and other cardiovascular complications.</p>
<!-- /wp:paragraph -->

<!-- wp:paragraph -->
<p>Furthermore, physical activity is a key regulator of metabolic function. It helps the body manage blood sugar levels more effectively, increasing insulin sensitivity and playing a crucial role in preventing and managing type 2 diabetes. It also aids in maintaining a healthy weight by burning calories and boosting metabolism, which in turn reduces strain on joints and organs.</p>
<!-- /wp:paragraph -->

<!-- wp:heading -->
<h2>Strength and Mobility</h2>
<!-- /wp:heading -->

<!-- wp:paragraph -->
<p>Alongside internal health, sport is instrumental in building and maintaining a robust musculoskeletal system. Weight-bearing and resistance exercises, such as weightlifting or bodyweight training, stimulate muscle growth and increase bone density. This is vital for long-term mobility, balance, and independence, particularly as we age.</p>
<!-- /wp:paragraph -->

<!-- wp:paragraph -->
<p>Strong muscles and bones protect against injuries from falls and help prevent conditions like osteoporosis and sarcopenia. The functional strength gained from sports translates directly into everyday life, making tasks easier and reducing physical fatigue. The body becomes not just a vessel, but a capable and resilient instrument.</p>
<!-- /wp:paragraph -->

<!-- wp:heading -->
<h2>Mental Well-being</h2>
<!-- /wp:heading -->

<!-- wp:paragraph -->
<p>However, to focus solely on the physical aspects would be to overlook one of the most powerful benefits of sport: its impact on mental health. Engaging in physical activity is a potent antidote to stress, anxiety, and depression. During exercise, the brain releases a cascade of chemicals, including endorphins, often referred to as "feel-good" hormones.</p>
<!-- /wp:paragraph -->

<!-- wp:paragraph -->
<p>These endorphins act as natural mood elevators and painkillers, creating a phenomenon commonly known as the "runner's high." This biochemical shift can alleviate feelings of sadness and tension, promoting a state of relaxation and well-being long after the activity has ended. Regular participation in sport has been shown to be as effective as medication for some individuals dealing with mild to moderate depression.</p>
<!-- /wp:paragraph -->

<!-- wp:heading -->
<h2>Cognitive Benefits and Personal Growth</h2>
<!-- /wp:heading -->

<!-- wp:paragraph -->
<p>Moreover, sport cultivates mental fortitude and cognitive function. The challenges inherent in athletic pursuit—pushing through fatigue, mastering a new skill, coping with loss, and striving for a goal—build character. Participants learn discipline, patience, perseverance, and resilience.</p>
<!-- /wp:paragraph -->

<!-- wp:paragraph -->
<p>These qualities, forged on the track, court, or gym floor, are directly transferable to personal and professional life. Simultaneously, physical activity increases blood flow to the brain, which can enhance memory, sharpen concentration, and stimulate creativity. It is a natural cognitive enhancer, protecting against age-related cognitive decline and improving overall brain health.</p>
<!-- /wp:paragraph -->

<!-- wp:heading -->
<h2>The Social Dimension</h2>
<!-- /wp:heading -->

<!-- wp:paragraph -->
<p>Beyond the individual, sport possesses a unique social dimension that is essential for human well-being. Team sports, in particular, foster a profound sense of community, belonging, and camaraderie. They teach invaluable lessons in teamwork, communication, and trust.</p>
<!-- /wp:paragraph -->

<!-- wp:paragraph -->
<p>Being part of a team provides a built-in support network, a group of individuals who share a common goal and can offer encouragement during setbacks and celebrate successes together. This social connection is a powerful buffer against loneliness and isolation, contributing significantly to emotional health. Even individual sports practiced in a club or class setting can provide this crucial social interaction and a sense of shared purpose.</p>
<!-- /wp:paragraph -->

<!-- wp:heading -->
<h2>Conclusion</h2>
<!-- /wp:heading -->

<!-- wp:paragraph -->
<p>In conclusion, the integration of sport and physical activity into daily life is a non-negotiable component of holistic health. It is a comprehensive strategy that fortifies the body against disease, sharpens the mind against decline, and nourishes the spirit against the strains of modern life.</p>
<!-- /wp:paragraph -->

<!-- wp:paragraph -->
<p>From the powerful heart and strong bones to the clear, resilient mind and the sense of community, the rewards are immense and interconnected. Embracing an active lifestyle is ultimately a profound commitment to oneself—a commitment to living not just longer, but with greater vitality, purpose, and joy.</p>
<!-- /wp:paragraph -->
	`
	// Publish to WordPress
	postID, postURL, err := e.wpClient.PublishPost(ctx, siteInfo, title, content, wpCategoryID)
	if err != nil {
		return fmt.Errorf("failed to publish post to WordPress: %w", err)
	}

	e.logger.Infof("Job %d: Article published successfully (post ID: %d, URL: %s)", job.ID, postID, postURL)

	// Get original topic title for article record
	t, err := e.topicService.GetTopic(ctx, exec.TopicID)
	if err != nil {
		return fmt.Errorf("failed to get topic for article record: %w", err)
	}

	// Calculate word count
	wordCount := len(strings.Fields(content))

	// Create article record in database
	now := time.Now()
	articleRecord := &entities.Article{
		SiteID:        job.SiteID,
		JobID:         &job.ID,
		TopicID:       exec.TopicID,
		Title:         title,
		OriginalTitle: t.Title,
		Content:       content,
		WPPostID:      postID,
		WPPostURL:     postURL,
		WPCategoryID:  wpCategoryID,
		Status:        entities.ArticleStatusPublished,
		WordCount:     &wordCount,
		PublishedAt:   &now,
	}

	if err = e.articleRepo.Create(ctx, articleRecord); err != nil {
		e.logger.Errorf("Failed to create article record: %v", err)
		// Don't fail the job - article is already published to WordPress
	} else {
		e.logger.Infof("Job %d: Article record created with ID %d", job.ID, articleRecord.ID)

		// Update execution with article ID
		exec.ArticleID = &articleRecord.ID
	}

	// Update execution with publication details
	exec.Status = ExecutionPublished
	exec.PublishedAt = &now

	if err = e.execRepo.Update(ctx, exec); err != nil {
		return fmt.Errorf("failed to update execution after publication: %w", err)
	}

	return nil
}

func (e *Executor) PublishValidatedArticle(ctx context.Context, job *Job, exec *Execution) error {
	if exec.GeneratedTitle == nil || exec.GeneratedContent == nil {
		return fmt.Errorf("execution missing generated title or content")
	}

	// Get site information
	siteInfo, err := e.siteService.GetSite(ctx, job.SiteID)
	if err != nil {
		return fmt.Errorf("failed to get site: %w", err)
	}

	e.logger.Infof("Publishing validated article for execution %d (job %d)", exec.ID, job.ID)

	// Call the existing publish logic
	return e.publishArticle(ctx, job, exec, siteInfo, *exec.GeneratedTitle, *exec.GeneratedContent)
}

func (e *Executor) getCategoryInfo(ctx context.Context, categoryID int64, siteID int64) (*entities.Category, error) {
	categories, err := e.siteService.GetSiteCategories(ctx, siteID)
	if err != nil {
		return nil, fmt.Errorf("failed to get site categories: %w", err)
	}

	for _, cat := range categories {
		if cat.ID == categoryID {
			return cat, nil
		}
	}

	return nil, fmt.Errorf("category %d not found for site %d", categoryID, siteID)
}
