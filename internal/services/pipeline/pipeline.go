package pipeline

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"Postulator/internal/models"
	"Postulator/internal/repository"
	"Postulator/internal/services/gpt"
	"Postulator/internal/services/wordpress"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// Service orchestrates the article generation and publishing pipeline
type Service struct {
	repos      *repository.Repository
	gptService *gpt.Service
	wpService  *wordpress.Service
	appContext context.Context
	activeJobs map[int64]*JobStatus
	mutex      sync.RWMutex
	workers    chan struct{}
	config     Config
}

// Config holds pipeline configuration
type Config struct {
	MaxWorkers       int
	JobTimeout       time.Duration
	RetryCount       int
	RetryDelay       time.Duration
	MinContentWords  int
	MaxDailyPosts    int
	WordPressTimeout time.Duration
	GPTTimeout       time.Duration
	FailureThreshold int
}

// NewService creates a new pipeline service
func NewService(config Config, repos *repository.Container, gptService *gpt.Service, wpService *wordpress.Service, appContext context.Context) *Service {
	if config.MaxWorkers == 0 {
		config.MaxWorkers = 5
	}
	if config.JobTimeout == 0 {
		config.JobTimeout = 15 * time.Minute
	}
	if config.RetryCount == 0 {
		config.RetryCount = 3
	}
	if config.RetryDelay == 0 {
		config.RetryDelay = 5 * time.Minute
	}
	if config.MinContentWords == 0 {
		config.MinContentWords = 500
	}
	if config.MaxDailyPosts == 0 {
		config.MaxDailyPosts = 10
	}

	return &Service{
		repos:      repos,
		gptService: gptService,
		wpService:  wpService,
		appContext: appContext,
		activeJobs: make(map[int64]*JobStatus),
		workers:    make(chan struct{}, config.MaxWorkers),
		config:     config,
	}
}

// JobStatus represents the status of a pipeline job
type JobStatus struct {
	ID          int64             `json:"id"`
	Type        string            `json:"type"`
	Status      string            `json:"status"`
	Progress    int               `json:"progress"`
	SiteID      int64             `json:"site_id"`
	ArticleID   int64             `json:"article_id,omitempty"`
	StartedAt   time.Time         `json:"started_at"`
	CompletedAt time.Time         `json:"completed_at,omitempty"`
	Error       string            `json:"error,omitempty"`
	RetryCount  int               `json:"retry_count"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// CreateArticleRequest represents a request to create an article
type CreateArticleRequest struct {
	SiteID   int64             `json:"site_id"`
	TopicID  int64             `json:"topic_id"`
	Publish  bool              `json:"publish"`
	Priority int               `json:"priority"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// CreateArticleResponse represents the result of article creation
type CreateArticleResponse struct {
	Article     *models.Article `json:"article"`
	WordPressID int64           `json:"wordpress_id,omitempty"`
	URL         string          `json:"url,omitempty"`
	Status      string          `json:"status"`
}

// ProcessScheduledJob processes a scheduled posting job
func (s *Service) ProcessScheduledJob(ctx context.Context, siteID int64, postsCount int) error {
	log.Printf("Processing scheduled job for site %d, posts count: %d", siteID, postsCount)

	// Check daily limit
	if postsCount > s.config.MaxDailyPosts {
		postsCount = s.config.MaxDailyPosts
	}

	// Get site information
	site, err := s.repos.Site.GetByID(ctx, siteID)
	if err != nil {
		return fmt.Errorf("failed to get site: %w", err)
	}
	if site == nil {
		return fmt.Errorf("site %d not found", siteID)
	}

	// Check if site is active
	if !site.IsActive {
		return fmt.Errorf("site %d is not active", siteID)
	}

	// Get active topics for this site
	siteTopics, err := s.repos.SiteTopic.GetBySiteID(ctx, siteID)
	if err != nil {
		return fmt.Errorf("failed to get site topics: %w", err)
	}

	if len(siteTopics) == 0 {
		return fmt.Errorf("no active topics found for site %d", siteID)
	}

	// Check recent posts to avoid over-posting
	recent, err := s.repos.Article.GetRecentBySite(ctx, siteID, 1)
	if err != nil {
		log.Printf("Warning: failed to check recent posts for site %d: %v", siteID, err)
	}

	if len(recent) >= postsCount {
		log.Printf("Site %d already has %d recent posts, skipping", siteID, len(recent))
		return nil
	}

	// Reduce posts count based on recent posts
	remainingPosts := postsCount - len(recent)
	if remainingPosts <= 0 {
		return nil
	}

	// Create posts
	for i := 0; i < remainingPosts && i < len(siteTopics); i++ {
		siteTopic := siteTopics[i%len(siteTopics)]

		req := CreateArticleRequest{
			SiteID:   siteID,
			TopicID:  siteTopic.TopicID,
			Publish:  true,
			Priority: siteTopic.Priority,
			Metadata: map[string]string{
				"scheduled": "true",
				"batch_id":  fmt.Sprintf("batch_%d_%d", siteID, time.Now().Unix()),
			},
		}

		go s.ProcessCreateArticleJob(ctx, req)
	}

	s.emitEvent("pipeline:scheduled_job_started", map[string]interface{}{
		"site_id":     siteID,
		"posts_count": remainingPosts,
	})

	return nil
}

// ProcessCreateArticleJob processes a single article creation job
func (s *Service) ProcessCreateArticleJob(ctx context.Context, req CreateArticleRequest) {
	// Acquire worker slot
	s.workers <- struct{}{}
	defer func() { <-s.workers }()

	// Create job status
	jobID := time.Now().UnixNano()
	status := &JobStatus{
		ID:        jobID,
		Type:      "create_article",
		Status:    "running",
		Progress:  0,
		SiteID:    req.SiteID,
		StartedAt: time.Now(),
		Metadata:  req.Metadata,
	}

	s.setJobStatus(jobID, status)
	defer s.removeJobStatus(jobID)

	// Create posting job in database
	postingJob := &models.PostingJob{
		Type:      "scheduled",
		SiteID:    req.SiteID,
		Status:    "running",
		Progress:  0,
		StartedAt: time.Now(),
		CreatedAt: time.Now(),
	}

	if err := s.repos.PostingJob.Create(ctx, postingJob); err != nil {
		log.Printf("Failed to create posting job: %v", err)
		return
	}

	status.ID = postingJob.ID
	s.updateJobProgress(jobID, 10, "Job created")

	// Execute the pipeline
	response, err := s.executeArticlePipeline(ctx, req, status)
	if err != nil {
		s.handleJobError(ctx, postingJob.ID, status, err)
		return
	}

	// Mark job as completed
	status.Status = "completed"
	status.Progress = 100
	status.CompletedAt = time.Now()
	status.ArticleID = response.Article.ID

	if err := s.repos.PostingJob.Complete(ctx, postingJob.ID); err != nil {
		log.Printf("Failed to mark posting job as complete: %v", err)
	}

	s.emitEvent("pipeline:job_completed", map[string]interface{}{
		"job_id":     status.ID,
		"article_id": response.Article.ID,
		"site_id":    req.SiteID,
		"url":        response.URL,
	})

	log.Printf("Article creation job completed: article_id=%d, url=%s", response.Article.ID, response.URL)
}

// executeArticlePipeline executes the complete article creation and publishing pipeline
func (s *Service) executeArticlePipeline(ctx context.Context, req CreateArticleRequest, status *JobStatus) (*CreateArticleResponse, error) {
	// Step 1: Get site and topic information
	s.updateJobProgress(status.ID, 15, "Loading site and topic data")

	site, err := s.repos.Site.GetByID(ctx, req.SiteID)
	if err != nil {
		return nil, fmt.Errorf("failed to get site: %w", err)
	}

	topic, err := s.repos.Topic.GetByID(ctx, req.TopicID)
	if err != nil {
		return nil, fmt.Errorf("failed to get topic: %w", err)
	}

	// Step 2: Generate article content using GPT
	s.updateJobProgress(status.ID, 25, "Generating article content")

	// Build prompt from topic and site
	prompt := topic.Prompt
	if prompt == "" {
		prompt = fmt.Sprintf("Write an article about: %s\n\nDescription: %s\nKeywords: %s\nCategory: %s\nTarget Tags: %s\nWebsite: %s",
			topic.Title, topic.Description, topic.Keywords, topic.Category, topic.Tags, site.URL)
	}

	gptReq := gpt.GenerateArticleRequest{
		Title:  topic.Title,
		Prompt: prompt,
	}

	gptCtx, cancel := context.WithTimeout(ctx, s.config.GPTTimeout)
	defer cancel()

	gptResponse, err := s.gptService.GenerateArticle(gptCtx, gptReq)
	if err != nil {
		return nil, fmt.Errorf("failed to generate article: %w", err)
	}

	// Step 3: Create article in database
	s.updateJobProgress(status.ID, 50, "Saving article to database")

	article := &models.Article{
		SiteID:    req.SiteID,
		TopicID:   req.TopicID,
		Title:     gptResponse.Article.Title,
		Content:   s.truncateContent(gptResponse.Article.Content),
		Excerpt:   gptResponse.Article.Excerpt,
		Keywords:  strings.Join(gptResponse.Article.Keywords, ", "),
		Tags:      strings.Join(gptResponse.Article.Tags, ", "),
		Category:  gptResponse.Article.Category,
		Status:    "generated",
		GPTModel:  gptResponse.Model,
		Tokens:    gptResponse.TokensUsed,
		CreatedAt: time.Now(),
	}

	if err := s.repos.Article.Create(ctx, article); err != nil {
		return nil, fmt.Errorf("failed to save article: %w", err)
	}

	// Update job with article ID
	status.ArticleID = article.ID
	s.repos.PostingJob.UpdateProgress(ctx, status.ID, 60)

	response := &CreateArticleResponse{
		Article: article,
		Status:  "generated",
	}

	// Step 4: Publish to WordPress if requested
	if req.Publish {
		s.updateJobProgress(status.ID, 70, "Publishing to WordPress")

		// Use full content for WordPress
		fullArticle := *article
		fullArticle.Content = gptResponse.Article.Content

		wpReq := wordpress.CreatePostRequest{
			Site:    site,
			Article: &fullArticle,
			Publish: true,
		}

		wpCtx, cancel := context.WithTimeout(ctx, s.config.WordPressTimeout)
		defer cancel()

		wpResponse, err := s.wpService.CreatePost(wpCtx, wpReq)
		if err != nil {
			// Article was created but publishing failed
			article.Status = "failed"
			s.repos.Article.UpdateStatus(ctx, article.ID, "failed")
			return nil, fmt.Errorf("failed to publish article: %w", err)
		}

		// Update article with WordPress information
		article.WordPressID = wpResponse.WordPressID
		article.Status = "published"
		article.PublishedAt = wpResponse.PublishedAt

		if err := s.repos.Article.SetWordPressID(ctx, article.ID, wpResponse.WordPressID); err != nil {
			log.Printf("Warning: failed to update WordPress ID: %v", err)
		}

		if err := s.repos.Article.UpdateStatus(ctx, article.ID, "published"); err != nil {
			log.Printf("Warning: failed to update article status: %v", err)
		}

		response.WordPressID = wpResponse.WordPressID
		response.URL = wpResponse.URL
		response.Status = "published"

		s.updateJobProgress(status.ID, 90, "Article published successfully")
	}

	return response, nil
}

// truncateContent truncates content to store only first N words
func (s *Service) truncateContent(content string) string {
	if len(content) <= s.config.MinContentWords*6 { // Rough estimate: 6 chars per word
		return content
	}

	// Return first 200 characters as preview
	if len(content) > 200 {
		return content[:200] + "..."
	}
	return content
}

// handleJobError handles job errors with retry logic
func (s *Service) handleJobError(ctx context.Context, jobID int64, status *JobStatus, err error) {
	status.Error = err.Error()
	status.Status = "failed"
	status.CompletedAt = time.Now()
	status.RetryCount++

	log.Printf("Job %d failed: %v (retry %d/%d)", jobID, err, status.RetryCount, s.config.RetryCount)

	// Update posting job in database
	s.repos.PostingJob.SetError(ctx, jobID, err.Error())

	if status.RetryCount < s.config.RetryCount {
		// Schedule retry
		go func() {
			time.Sleep(s.config.RetryDelay)
			log.Printf("Retrying job %d (attempt %d)", jobID, status.RetryCount+1)
			// Here you would re-queue the job for retry
		}()

		s.emitEvent("pipeline:job_retry", map[string]interface{}{
			"job_id":      jobID,
			"retry_count": status.RetryCount,
			"error":       err.Error(),
		})
	} else {
		s.emitEvent("pipeline:job_failed", map[string]interface{}{
			"job_id": jobID,
			"error":  err.Error(),
		})
	}
}

// updateJobProgress updates job progress and emits events
func (s *Service) updateJobProgress(jobID int64, progress int, message string) {
	s.mutex.Lock()
	if status, exists := s.activeJobs[jobID]; exists {
		status.Progress = progress
	}
	s.mutex.Unlock()

	// Update in database
	s.repos.PostingJob.UpdateProgress(context.Background(), jobID, progress)

	// Emit event
	s.emitEvent("pipeline:job_progress", map[string]interface{}{
		"job_id":   jobID,
		"progress": progress,
		"message":  message,
	})

	log.Printf("Job %d progress: %d%% - %s", jobID, progress, message)
}

// setJobStatus sets job status in memory
func (s *Service) setJobStatus(jobID int64, status *JobStatus) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.activeJobs[jobID] = status
}

// removeJobStatus removes job status from memory
func (s *Service) removeJobStatus(jobID int64) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	delete(s.activeJobs, jobID)
}

// GetActiveJobs returns currently active jobs
func (s *Service) GetActiveJobs() map[int64]*JobStatus {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	result := make(map[int64]*JobStatus)
	for id, status := range s.activeJobs {
		result[id] = status
	}
	return result
}

// GetJobStatus returns status of a specific job
func (s *Service) GetJobStatus(jobID int64) (*JobStatus, bool) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	status, exists := s.activeJobs[jobID]
	return status, exists
}

// TestSiteConnection tests connection to a WordPress site
func (s *Service) TestSiteConnection(ctx context.Context, siteID int64) error {
	site, err := s.repos.Site.GetByID(ctx, siteID)
	if err != nil {
		return fmt.Errorf("failed to get site: %w", err)
	}

	if site == nil {
		return fmt.Errorf("site not found")
	}

	return s.wpService.TestConnection(ctx, site)
}

// GeneratePreviewArticle generates an article preview without saving or publishing
func (s *Service) GeneratePreviewArticle(ctx context.Context, siteID, topicID int64) (*gpt.GenerateArticleResponse, error) {
	site, err := s.repos.Site.GetByID(ctx, siteID)
	if err != nil {
		return nil, fmt.Errorf("failed to get site: %w", err)
	}

	topic, err := s.repos.Topic.GetByID(ctx, topicID)
	if err != nil {
		return nil, fmt.Errorf("failed to get topic: %w", err)
	}

	// Build prompt from topic and site
	prompt := topic.Prompt
	if prompt == "" {
		prompt = fmt.Sprintf("Write an article about: %s\n\nDescription: %s\nKeywords: %s\nCategory: %s\nTarget Tags: %s\nWebsite: %s",
			topic.Title, topic.Description, topic.Keywords, topic.Category, topic.Tags, site.URL)
	}

	req := gpt.GenerateArticleRequest{
		Title:  topic.Title,
		Prompt: prompt,
	}

	return s.gptService.GenerateArticle(ctx, req)
}

// emitEvent emits an event to the frontend
func (s *Service) emitEvent(eventName string, data interface{}) {
	if s.appContext != nil {
		runtime.EventsEmit(s.appContext, eventName, data)
	}
}

// GetPipelineStats returns pipeline statistics
func (s *Service) GetPipelineStats() PipelineStats {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	stats := PipelineStats{
		ActiveJobs: len(s.activeJobs),
		MaxWorkers: s.config.MaxWorkers,
		Available:  s.config.MaxWorkers - len(s.activeJobs),
	}

	for _, status := range s.activeJobs {
		switch status.Status {
		case "running":
			stats.RunningJobs++
		case "failed":
			stats.FailedJobs++
		case "completed":
			stats.CompletedJobs++
		}
	}

	return stats
}

// PipelineStats contains pipeline statistics
type PipelineStats struct {
	ActiveJobs    int `json:"active_jobs"`
	RunningJobs   int `json:"running_jobs"`
	CompletedJobs int `json:"completed_jobs"`
	FailedJobs    int `json:"failed_jobs"`
	MaxWorkers    int `json:"max_workers"`
	Available     int `json:"available_workers"`
}
