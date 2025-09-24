package dto

import (
	"fmt"
	"strings"
	"time"

	"Postulator/internal/models"
)

// Pagination request
type PaginationRequest struct {
	Page  int `json:"page"`
	Limit int `json:"limit"`
}

// Pagination response
type PaginationResponse struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

// Generic paginated response wrapper
type PaginatedResponse[T any] struct {
	Data       []T                 `json:"data"`
	Pagination *PaginationResponse `json:"pagination"`
}

// Site DTOs
type CreateSiteRequest struct {
	Name     string `json:"name" validate:"required,min=1,max=100"`
	URL      string `json:"url" validate:"required,url"`
	Username string `json:"username" validate:"required,min=1,max=100"`
	Password string `json:"password" validate:"required,min=1"`
	IsActive bool   `json:"is_active"`
	Strategy string `json:"strategy,omitempty"`
}

type UpdateSiteRequest struct {
	ID       int64  `json:"id" validate:"required,min=1"`
	Name     string `json:"name" validate:"required,min=1,max=100"`
	URL      string `json:"url" validate:"required,url"`
	Username string `json:"username" validate:"required,min=1,max=100"`
	Password string `json:"password" validate:"required,min=1"`
	IsActive bool   `json:"is_active"`
	Strategy string `json:"strategy,omitempty"`
}

type SiteResponse struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	URL       string    `json:"url"`
	Username  string    `json:"username"`
	Password  string    `json:"password"`
	IsActive  bool      `json:"is_active"`
	LastCheck time.Time `json:"last_check"`
	Status    string    `json:"status"`
	Strategy  string    `json:"strategy"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type SiteListResponse struct {
	Sites      []*SiteResponse     `json:"sites"`
	Pagination *PaginationResponse `json:"pagination"`
}

// Topic DTOs
type CreateTopicRequest struct {
	Title    string `json:"title" validate:"required,min=1,max=200"`
	Keywords string `json:"keywords,omitempty"`
	Category string `json:"category,omitempty"`
	Tags     string `json:"tags,omitempty"`
}

type UpdateTopicRequest struct {
	ID       int64  `json:"id" validate:"required,min=1"`
	Title    string `json:"title" validate:"required,min=1,max=200"`
	Keywords string `json:"keywords,omitempty"`
	Category string `json:"category,omitempty"`
	Tags     string `json:"tags,omitempty"`
}

type TopicResponse struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	Keywords  string    `json:"keywords"`
	Category  string    `json:"category"`
	Tags      string    `json:"tags"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type TopicListResponse struct {
	Topics     []*TopicResponse    `json:"topics"`
	Pagination *PaginationResponse `json:"pagination"`
}

// Schedule DTOs
type CreateScheduleRequest struct {
	SiteID      int64  `json:"site_id" validate:"required,min=1"`
	CronExpr    string `json:"cron_expr" validate:"required"`
	PostsPerDay int    `json:"posts_per_day" validate:"min=1,max=50"`
	IsActive    bool   `json:"is_active"`
}

type UpdateScheduleRequest struct {
	ID          int64  `json:"id" validate:"required,min=1"`
	SiteID      int64  `json:"site_id" validate:"required,min=1"`
	CronExpr    string `json:"cron_expr" validate:"required"`
	PostsPerDay int    `json:"posts_per_day" validate:"min=1,max=50"`
	IsActive    bool   `json:"is_active"`
}

type ScheduleResponse struct {
	ID          int64     `json:"id"`
	SiteID      int64     `json:"site_id"`
	SiteName    string    `json:"site_name,omitempty"`
	CronExpr    string    `json:"cron_expr"`
	PostsPerDay int       `json:"posts_per_day"`
	IsActive    bool      `json:"is_active"`
	LastRun     time.Time `json:"last_run"`
	NextRun     time.Time `json:"next_run"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type ScheduleListResponse struct {
	Schedules  []*ScheduleResponse `json:"schedules"`
	Pagination *PaginationResponse `json:"pagination"`
}

// Article DTOs
type ArticleResponse struct {
	ID          int64     `json:"id"`
	SiteID      int64     `json:"site_id"`
	SiteName    string    `json:"site_name,omitempty"`
	TopicID     int64     `json:"topic_id"`
	TopicTitle  string    `json:"topic_title,omitempty"`
	Title       string    `json:"title"`
	Excerpt     string    `json:"excerpt"`
	Keywords    string    `json:"keywords"`
	Tags        string    `json:"tags"`
	Category    string    `json:"category"`
	Status      string    `json:"status"`
	WordPressID int64     `json:"wordpress_id"`
	GPTModel    string    `json:"gpt_model"`
	Tokens      int       `json:"tokens"`
	CreatedAt   time.Time `json:"created_at"`
	PublishedAt time.Time `json:"published_at"`
}

type ArticleListResponse struct {
	Articles   []*ArticleResponse  `json:"articles"`
	Pagination *PaginationResponse `json:"pagination"`
}

type CreateArticleManualRequest struct {
	SiteID       int64             `json:"site_id" validate:"required,min=1"`
	TopicID      int64             `json:"topic_id" validate:"required,min=1"`
	Publish      bool              `json:"publish"`
	CustomPrompt string            `json:"custom_prompt,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// PostingJob DTOs
type PostingJobResponse struct {
	ID          int64     `json:"id"`
	Type        string    `json:"type"`
	SiteID      int64     `json:"site_id"`
	SiteName    string    `json:"site_name,omitempty"`
	ArticleID   int64     `json:"article_id"`
	Status      string    `json:"status"`
	Progress    int       `json:"progress"`
	ErrorMsg    string    `json:"error_msg"`
	StartedAt   time.Time `json:"started_at"`
	CompletedAt time.Time `json:"completed_at"`
	CreatedAt   time.Time `json:"created_at"`
}

type PostingJobListResponse struct {
	Jobs       []*PostingJobResponse `json:"jobs"`
	Pagination *PaginationResponse   `json:"pagination"`
}

// SiteTopic DTOs
type CreateSiteTopicRequest struct {
	SiteID   int64 `json:"site_id" validate:"required,min=1"`
	TopicID  int64 `json:"topic_id" validate:"required,min=1"`
	Priority int   `json:"priority" validate:"min=1,max=10"`
}

type UpdateSiteTopicRequest struct {
	ID       int64 `json:"id" validate:"required,min=1"`
	SiteID   int64 `json:"site_id" validate:"required,min=1"`
	TopicID  int64 `json:"topic_id" validate:"required,min=1"`
	Priority int   `json:"priority" validate:"min=1,max=10"`
}

type SiteTopicResponse struct {
	ID            int64      `json:"id"`
	SiteID        int64      `json:"site_id"`
	SiteName      string     `json:"site_name,omitempty"`
	TopicID       int64      `json:"topic_id"`
	TopicTitle    string     `json:"topic_title,omitempty"`
	Priority      int        `json:"priority"`
	UsageCount    int        `json:"usage_count"`
	LastUsedAt    *time.Time `json:"last_used_at,omitempty"`
	RoundRobinPos int        `json:"round_robin_pos"`
}

type SiteTopicListResponse struct {
	SiteTopics []*SiteTopicResponse `json:"site_topics"`
	Pagination *PaginationResponse  `json:"pagination"`
}

// Dashboard DTOs
type DashboardStats struct {
	TotalSites        int64 `json:"total_sites"`
	ActiveSites       int64 `json:"active_sites"`
	TotalTopics       int64 `json:"total_topics"`
	ActiveTopics      int64 `json:"active_topics"`
	TotalArticles     int64 `json:"total_articles"`
	PublishedArticles int64 `json:"published_articles"`
	PendingJobs       int64 `json:"pending_jobs"`
	RunningJobs       int64 `json:"running_jobs"`
}

type RecentActivity struct {
	Type         string    `json:"type"`
	Description  string    `json:"description"`
	Timestamp    time.Time `json:"timestamp"`
	SiteID       int64     `json:"site_id,omitempty"`
	SiteName     string    `json:"site_name,omitempty"`
	ArticleID    int64     `json:"article_id,omitempty"`
	ArticleTitle string    `json:"article_title,omitempty"`
}

type DashboardResponse struct {
	Stats             *DashboardStats     `json:"stats"`
	RecentActivities  []*RecentActivity   `json:"recent_activities"`
	UpcomingSchedules []*ScheduleResponse `json:"upcoming_schedules"`
}

// Settings DTOs
type SettingRequest struct {
	Key   string `json:"key" validate:"required,min=1"`
	Value string `json:"value"`
}

type SettingResponse struct {
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	UpdatedAt time.Time `json:"updated_at"`
}

type SettingsResponse struct {
	Settings []*SettingResponse `json:"settings"`
}

// GPT Configuration DTOs
type GPTConfigRequest struct {
	Model     string `json:"model"`
	MaxTokens int    `json:"max_tokens" validate:"min=100,max=8000"`
}

type GPTConfigResponse struct {
	Model     string `json:"model"`
	MaxTokens int    `json:"max_tokens"`
}

// Test Connection DTOs
type TestSiteConnectionRequest struct {
	SiteID int64 `json:"site_id" validate:"required,min=1"`
}

type TestConnectionResponse struct {
	Success   bool      `json:"success"`
	Status    string    `json:"status"`
	Message   string    `json:"message"`
	Details   string    `json:"details,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// Preview Article DTOs
type PreviewArticleRequest struct {
	SiteID       int64  `json:"site_id" validate:"required,min=1"`
	TopicID      int64  `json:"topic_id" validate:"required,min=1"`
	CustomPrompt string `json:"custom_prompt,omitempty"`
}

type PreviewArticleResponse struct {
	Title      string `json:"title"`
	Content    string `json:"content"`
	Excerpt    string `json:"excerpt"`
	Keywords   string `json:"keywords"`
	Tags       string `json:"tags"`
	Category   string `json:"category"`
	TokensUsed int    `json:"tokens_used"`
	Model      string `json:"model"`
}

// Prompt DTOs
type CreatePromptRequest struct {
	Name      string `json:"name" validate:"required,min=1,max=100"`
	System    string `json:"system" validate:"required,min=1,max=500"`
	User      string `json:"user"  validate:"required,min=1,max=1000"`
	IsDefault bool   `json:"is_default"`
	IsActive  bool   `json:"is_active"`
}

type UpdatePromptRequest struct {
	ID          int64  `json:"id" validate:"required,min=1"`
	Name        string `json:"name" validate:"required,min=1,max=100"`
	Type        string `json:"type" validate:"required,oneof=system user"`
	Content     string `json:"content" validate:"required,min=1"`
	Description string `json:"description,omitempty"`
	IsDefault   bool   `json:"is_default"`
	IsActive    bool   `json:"is_active"`
}

type PromptResponse struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	System    string    `json:"system"`
	User      string    `json:"user"`
	IsDefault bool      `json:"is_default"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type PromptListResponse struct {
	Prompts    []*PromptResponse   `json:"prompts"`
	Pagination *PaginationResponse `json:"pagination"`
}

type SetDefaultPromptRequest struct {
	ID   int64  `json:"id" validate:"required,min=1"`
	Type string `json:"type" validate:"required,oneof=system user"`
}

// SitePrompt DTOs
type CreateSitePromptRequest struct {
	SiteID   int64 `json:"site_id" validate:"required,min=1"`
	PromptID int64 `json:"prompt_id" validate:"required,min=1"`
}

type UpdateSitePromptRequest struct {
	ID       int64 `json:"id" validate:"required,min=1"`
	SiteID   int64 `json:"site_id" validate:"required,min=1"`
	PromptID int64 `json:"prompt_id" validate:"required,min=1"`
}

type SitePromptResponse struct {
	ID         int64     `json:"id"`
	SiteID     int64     `json:"site_id"`
	SiteName   string    `json:"site_name,omitempty"`
	PromptID   int64     `json:"prompt_id"`
	PromptName string    `json:"prompt_name,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type SitePromptListResponse struct {
	SitePrompts []*SitePromptResponse `json:"site_prompts"`
	Pagination  *PaginationResponse   `json:"pagination"`
}

// Topic Strategy and Selection DTOs
type TopicSelectionRequest struct {
	SiteID   int64  `json:"site_id" validate:"required,min=1"`
	Strategy string `json:"strategy,omitempty"` // "unique", "round_robin", "random"
}

type TopicSelectionResponse struct {
	Topic          *TopicResponse     `json:"topic"`
	SiteTopic      *SiteTopicResponse `json:"site_topic"`
	Strategy       string             `json:"strategy"`
	CanContinue    bool               `json:"can_continue"`
	RemainingCount int                `json:"remaining_count"`
}

type TopicStatsResponse struct {
	SiteID             int64      `json:"site_id"`
	TotalTopics        int        `json:"total_topics"`
	ActiveTopics       int        `json:"active_topics"`
	UsedTopics         int        `json:"used_topics"`
	UnusedTopics       int        `json:"unused_topics"`
	UniqueTopicsLeft   int        `json:"unique_topics_left"`
	RoundRobinPosition int        `json:"round_robin_position"`
	MostUsedTopicID    int64      `json:"most_used_topic_id"`
	MostUsedTopicCount int        `json:"most_used_topic_count"`
	LastUsedTopicID    int64      `json:"last_used_topic_id"`
	LastUsedAt         *time.Time `json:"last_used_at"`
}

type TopicUsageResponse struct {
	ID        int64     `json:"id"`
	SiteID    int64     `json:"site_id"`
	TopicID   int64     `json:"topic_id"`
	ArticleID int64     `json:"article_id"`
	Strategy  string    `json:"strategy"`
	UsedAt    time.Time `json:"used_at"`
	CreatedAt time.Time `json:"created_at"`
}

type TopicUsageListResponse struct {
	UsageHistory []*TopicUsageResponse `json:"usage_history"`
	Pagination   *PaginationResponse   `json:"pagination"`
}

type StrategyAvailabilityResponse struct {
	SiteID         int64  `json:"site_id"`
	Strategy       string `json:"strategy"`
	CanContinue    bool   `json:"can_continue"`
	TotalTopics    int    `json:"total_topics"`
	ActiveTopics   int    `json:"active_topics"`
	UnusedTopics   int    `json:"unused_topics"`
	RemainingCount int    `json:"remaining_count"`
}

// Topics Import structures
type TopicsImportRequest struct {
	SiteID      int64  `json:"site_id"`
	FileContent string `json:"file_content"`
	FileFormat  string `json:"file_format"` // txt, csv, jsonl
	PreviewOnly bool   `json:"preview_only"`
}

type ImportTopicItem struct {
	Title    string `json:"title"`
	Keywords string `json:"keywords,omitempty"`
	Category string `json:"category,omitempty"`
	Tags     string `json:"tags,omitempty"`
	Status   string `json:"status"` // new, duplicate, exists
	Error    string `json:"error,omitempty"`
}

type TopicsImportPreview struct {
	SiteID        int64             `json:"site_id"`
	TotalLines    int               `json:"total_lines"`
	ValidTopics   int               `json:"valid_topics"`
	Duplicates    int               `json:"duplicates"`
	Errors        int               `json:"errors"`
	Topics        []ImportTopicItem `json:"topics"`
	ErrorMessages []string          `json:"error_messages,omitempty"`
}

type TopicsImportResult struct {
	SiteID         int64             `json:"site_id"`
	TotalProcessed int               `json:"total_processed"`
	CreatedTopics  int               `json:"created_topics"`
	ReusedTopics   int               `json:"reused_topics"`
	SkippedTopics  int               `json:"skipped_topics"`
	ErrorCount     int               `json:"error_count"`
	Topics         []ImportTopicItem `json:"topics"`
	ErrorMessages  []string          `json:"error_messages,omitempty"`
}

// Topics Reassign structures
type TopicsReassignRequest struct {
	FromSiteID int64   `json:"from_site_id"`
	ToSiteID   int64   `json:"to_site_id"`
	TopicIDs   []int64 `json:"topic_ids,omitempty"` // empty means all topics
}

type ReassignResult struct {
	FromSiteID       int64    `json:"from_site_id"`
	ToSiteID         int64    `json:"to_site_id"`
	ProcessedTopics  int      `json:"processed_topics"`
	ReassignedTopics int      `json:"reassigned_topics"`
	SkippedTopics    int      `json:"skipped_topics"`
	ErrorCount       int      `json:"error_count"`
	ErrorMessages    []string `json:"error_messages,omitempty"`
}

// Pipeline and Job Management structures
type GeneratePublishRequest struct {
	SiteID   int64  `json:"site_id"`
	TopicID  *int64 `json:"topic_id,omitempty"`  // Optional: if not provided, will use site strategy
	Strategy string `json:"strategy,omitempty"`  // Optional: override site default strategy
	Title    string `json:"title,omitempty"`     // Optional: override topic title
	Tone     string `json:"tone,omitempty"`      // Optional: content tone
	Style    string `json:"style,omitempty"`     // Optional: content style
	MinWords int    `json:"min_words,omitempty"` // Optional: minimum word count
}

type JobResponse struct {
	ID          int64      `json:"id"`
	Type        string     `json:"type"`
	SiteID      int64      `json:"site_id"`
	ArticleID   *int64     `json:"article_id,omitempty"`
	Status      string     `json:"status"`
	Progress    int        `json:"progress"`
	ErrorMsg    string     `json:"error_msg,omitempty"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

type JobListResponse struct {
	Jobs       []*JobResponse      `json:"jobs"`
	Pagination *PaginationResponse `json:"pagination"`
}

// Conversion utilities
func (r *CreateSiteRequest) ToModel() *models.Site {
	return &models.Site{
		Name:      r.Name,
		URL:       r.URL,
		Username:  r.Username,
		Password:  r.Password,
		IsActive:  r.IsActive,
		Status:    "pending",
		Strategy:  r.Strategy,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func (r *UpdateSiteRequest) ToModel() *models.Site {
	return &models.Site{
		ID:        r.ID,
		Name:      r.Name,
		URL:       r.URL,
		Username:  r.Username,
		Password:  r.Password,
		IsActive:  r.IsActive,
		Strategy:  r.Strategy,
		UpdatedAt: time.Now(),
	}
}

func SiteToResponse(site *models.Site) *SiteResponse {
	return &SiteResponse{
		ID:        site.ID,
		Name:      site.Name,
		URL:       site.URL,
		Username:  site.Username,
		Password:  site.Password,
		IsActive:  site.IsActive,
		LastCheck: site.LastCheck,
		Status:    site.Status,
		Strategy:  site.Strategy,
		CreatedAt: site.CreatedAt,
		UpdatedAt: site.UpdatedAt,
	}
}

func (r *CreateTopicRequest) ToModel() *models.Topic {
	return &models.Topic{
		Title:     r.Title,
		Keywords:  r.Keywords,
		Category:  r.Category,
		Tags:      r.Tags,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func (r *UpdateTopicRequest) ToModel() *models.Topic {
	return &models.Topic{
		ID:        r.ID,
		Title:     r.Title,
		Keywords:  r.Keywords,
		Category:  r.Category,
		Tags:      r.Tags,
		UpdatedAt: time.Now(),
	}
}

func TopicToResponse(topic *models.Topic) *TopicResponse {
	return &TopicResponse{
		ID:        topic.ID,
		Title:     topic.Title,
		Keywords:  topic.Keywords,
		Category:  topic.Category,
		Tags:      topic.Tags,
		CreatedAt: topic.CreatedAt,
		UpdatedAt: topic.UpdatedAt,
	}
}

func (r *CreateScheduleRequest) ToModel() *models.Schedule {
	return &models.Schedule{
		SiteID:    r.SiteID,
		CronExpr:  r.CronExpr,
		IsActive:  r.IsActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func ScheduleToResponse(schedule *models.Schedule) *ScheduleResponse {
	return &ScheduleResponse{
		ID:          schedule.ID,
		SiteID:      schedule.SiteID,
		CronExpr:    schedule.CronExpr,
		PostsPerDay: 0, // Not in model, set default
		IsActive:    schedule.IsActive,
		LastRun:     schedule.LastRun,
		NextRun:     schedule.NextRun,
		CreatedAt:   schedule.CreatedAt,
		UpdatedAt:   schedule.UpdatedAt,
	}
}

func ArticleToResponse(article *models.Article) *ArticleResponse {
	return &ArticleResponse{
		ID:          article.ID,
		SiteID:      article.SiteID,
		TopicID:     article.TopicID,
		Title:       article.Title,
		Excerpt:     "", // Not in model, set empty
		Keywords:    article.Keywords,
		Tags:        article.Tags,
		Category:    article.Category,
		Status:      article.Status,
		WordPressID: article.WordPressID,
		GPTModel:    article.GPTModel,
		Tokens:      article.Tokens,
		CreatedAt:   article.CreatedAt,
		PublishedAt: article.PublishedAt,
	}
}

func PostingJobToResponse(job *models.PostingJob) *PostingJobResponse {
	return &PostingJobResponse{
		ID:          job.ID,
		Type:        job.Type,
		SiteID:      job.SiteID,
		ArticleID:   job.ArticleID,
		Status:      job.Status,
		Progress:    job.Progress,
		ErrorMsg:    job.ErrorMsg,
		StartedAt:   job.StartedAt,
		CompletedAt: job.CompletedAt,
		CreatedAt:   job.CreatedAt,
	}
}

func (r *CreatePromptRequest) ToModel() *models.Prompt {
	prompt := &models.Prompt{
		Name:         r.Name,
		SystemPrompt: r.System,
		UserPrompt:   r.User,
		IsDefault:    r.IsDefault,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	return prompt
}

func (r *UpdatePromptRequest) ToModel() *models.Prompt {
	prompt := &models.Prompt{
		ID:        r.ID,
		Name:      r.Name,
		IsDefault: r.IsDefault,
		UpdatedAt: time.Now(),
	}

	// Map content based on type
	if r.Type == "system" {
		prompt.SystemPrompt = r.Content
		prompt.UserPrompt = ""
	} else {
		prompt.SystemPrompt = ""
		prompt.UserPrompt = r.Content
	}

	return prompt
}

func PromptToResponse(prompt *models.Prompt) *PromptResponse {
	return &PromptResponse{
		ID:        prompt.ID,
		Name:      prompt.Name,
		System:    prompt.SystemPrompt,
		User:      prompt.UserPrompt,
		IsDefault: prompt.IsDefault,
		IsActive:  true, // Not in model, set default
		CreatedAt: prompt.CreatedAt,
		UpdatedAt: prompt.UpdatedAt,
	}
}

func (r *CreateSitePromptRequest) ToModel() *models.SitePrompt {
	return &models.SitePrompt{
		SiteID:    r.SiteID,
		PromptID:  r.PromptID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func (r *UpdateSitePromptRequest) ToModel() *models.SitePrompt {
	return &models.SitePrompt{
		ID:        r.ID,
		SiteID:    r.SiteID,
		PromptID:  r.PromptID,
		UpdatedAt: time.Now(),
	}
}

func SitePromptToResponse(sitePrompt *models.SitePrompt) *SitePromptResponse {
	return &SitePromptResponse{
		ID:         sitePrompt.ID,
		SiteID:     sitePrompt.SiteID,
		SiteName:   "", // Not in model, will be populated by handler if needed
		PromptID:   sitePrompt.PromptID,
		PromptName: "", // Not in model, will be populated by handler if needed
		CreatedAt:  sitePrompt.CreatedAt,
		UpdatedAt:  sitePrompt.UpdatedAt,
	}
}

func JobToResponse(job *models.PostingJob) *JobResponse {
	var articleID *int64
	if job.ArticleID != 0 {
		articleID = &job.ArticleID
	}

	var startedAt *time.Time
	if !job.StartedAt.IsZero() {
		startedAt = &job.StartedAt
	}

	var completedAt *time.Time
	if !job.CompletedAt.IsZero() {
		completedAt = &job.CompletedAt
	}

	return &JobResponse{
		ID:          job.ID,
		Type:        job.Type,
		SiteID:      job.SiteID,
		ArticleID:   articleID,
		Status:      job.Status,
		Progress:    job.Progress,
		ErrorMsg:    job.ErrorMsg,
		StartedAt:   startedAt,
		CompletedAt: completedAt,
		CreatedAt:   job.CreatedAt,
	}
}

// Validation utilities
func ValidateRequired(value string, fieldName string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s is required", fieldName)
	}
	return nil
}

func ValidateURL(url string) error {
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return fmt.Errorf("URL must start with http:// or https://")
	}
	return nil
}

func ValidateRange(value, min, max int, fieldName string) error {
	if value < min || value > max {
		return fmt.Errorf("%s must be between %d and %d", fieldName, min, max)
	}
	return nil
}

func ValidateCronExpression(expr string) error {
	// Basic cron expression validation
	parts := strings.Fields(expr)
	if len(parts) != 5 {
		return fmt.Errorf("cron expression must have 5 parts")
	}
	return nil
}
