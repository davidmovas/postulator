package dto

import (
	"fmt"
	"strings"
	"time"

	"Postulator/internal/models"
)

// Base response structure
type BaseResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Error   string      `json:"error,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

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

// Site DTOs
type CreateSiteRequest struct {
	Name     string `json:"name" validate:"required,min=1,max=100"`
	URL      string `json:"url" validate:"required,url"`
	Username string `json:"username" validate:"required,min=1,max=100"`
	Password string `json:"password" validate:"required,min=1"`
	APIKey   string `json:"api_key,omitempty"`
	IsActive bool   `json:"is_active"`
}

type UpdateSiteRequest struct {
	ID       int64  `json:"id" validate:"required,min=1"`
	Name     string `json:"name" validate:"required,min=1,max=100"`
	URL      string `json:"url" validate:"required,url"`
	Username string `json:"username" validate:"required,min=1,max=100"`
	Password string `json:"password" validate:"required,min=1"`
	APIKey   string `json:"api_key,omitempty"`
	IsActive bool   `json:"is_active"`
}

type SiteResponse struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	URL       string    `json:"url"`
	Username  string    `json:"username"`
	IsActive  bool      `json:"is_active"`
	LastCheck time.Time `json:"last_check"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type SiteListResponse struct {
	Sites      []*SiteResponse     `json:"sites"`
	Pagination *PaginationResponse `json:"pagination"`
}

// Topic DTOs
type CreateTopicRequest struct {
	Title       string `json:"title" validate:"required,min=1,max=200"`
	Description string `json:"description,omitempty"`
	Keywords    string `json:"keywords,omitempty"`
	Prompt      string `json:"prompt,omitempty"`
	Category    string `json:"category,omitempty"`
	Tags        string `json:"tags,omitempty"`
	IsActive    bool   `json:"is_active"`
}

type UpdateTopicRequest struct {
	ID          int64  `json:"id" validate:"required,min=1"`
	Title       string `json:"title" validate:"required,min=1,max=200"`
	Description string `json:"description,omitempty"`
	Keywords    string `json:"keywords,omitempty"`
	Prompt      string `json:"prompt,omitempty"`
	Category    string `json:"category,omitempty"`
	Tags        string `json:"tags,omitempty"`
	IsActive    bool   `json:"is_active"`
}

type TopicResponse struct {
	ID          int64     `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Keywords    string    `json:"keywords"`
	Prompt      string    `json:"prompt"`
	Category    string    `json:"category"`
	Tags        string    `json:"tags"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
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
	IsActive bool  `json:"is_active"`
}

type UpdateSiteTopicRequest struct {
	ID       int64 `json:"id" validate:"required,min=1"`
	SiteID   int64 `json:"site_id" validate:"required,min=1"`
	TopicID  int64 `json:"topic_id" validate:"required,min=1"`
	Priority int   `json:"priority" validate:"min=1,max=10"`
	IsActive bool  `json:"is_active"`
}

type SiteTopicResponse struct {
	ID         int64  `json:"id"`
	SiteID     int64  `json:"site_id"`
	SiteName   string `json:"site_name,omitempty"`
	TopicID    int64  `json:"topic_id"`
	TopicTitle string `json:"topic_title,omitempty"`
	Priority   int    `json:"priority"`
	IsActive   bool   `json:"is_active"`
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
	APIKey    string `json:"api_key" validate:"required"`
	Model     string `json:"model"`
	MaxTokens int    `json:"max_tokens" validate:"min=100,max=8000"`
}

type GPTConfigResponse struct {
	Model     string `json:"model"`
	MaxTokens int    `json:"max_tokens"`
	HasAPIKey bool   `json:"has_api_key"`
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

// Conversion utilities
func (r *CreateSiteRequest) ToModel() *models.Site {
	return &models.Site{
		Name:      r.Name,
		URL:       r.URL,
		Username:  r.Username,
		Password:  r.Password,
		APIKey:    r.APIKey,
		IsActive:  r.IsActive,
		Status:    "pending",
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
		APIKey:    r.APIKey,
		IsActive:  r.IsActive,
		UpdatedAt: time.Now(),
	}
}

func SiteToResponse(site *models.Site) *SiteResponse {
	return &SiteResponse{
		ID:        site.ID,
		Name:      site.Name,
		URL:       site.URL,
		Username:  site.Username,
		IsActive:  site.IsActive,
		LastCheck: site.LastCheck,
		Status:    site.Status,
		CreatedAt: site.CreatedAt,
		UpdatedAt: site.UpdatedAt,
	}
}

func (r *CreateTopicRequest) ToModel() *models.Topic {
	return &models.Topic{
		Title:       r.Title,
		Description: r.Description,
		Keywords:    r.Keywords,
		Prompt:      r.Prompt,
		Category:    r.Category,
		Tags:        r.Tags,
		IsActive:    r.IsActive,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

func (r *UpdateTopicRequest) ToModel() *models.Topic {
	return &models.Topic{
		ID:          r.ID,
		Title:       r.Title,
		Description: r.Description,
		Keywords:    r.Keywords,
		Prompt:      r.Prompt,
		Category:    r.Category,
		Tags:        r.Tags,
		IsActive:    r.IsActive,
		UpdatedAt:   time.Now(),
	}
}

func TopicToResponse(topic *models.Topic) *TopicResponse {
	return &TopicResponse{
		ID:          topic.ID,
		Title:       topic.Title,
		Description: topic.Description,
		Keywords:    topic.Keywords,
		Prompt:      topic.Prompt,
		Category:    topic.Category,
		Tags:        topic.Tags,
		IsActive:    topic.IsActive,
		CreatedAt:   topic.CreatedAt,
		UpdatedAt:   topic.UpdatedAt,
	}
}

func (r *CreateScheduleRequest) ToModel() *models.Schedule {
	return &models.Schedule{
		SiteID:      r.SiteID,
		CronExpr:    r.CronExpr,
		PostsPerDay: r.PostsPerDay,
		IsActive:    r.IsActive,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

func ScheduleToResponse(schedule *models.Schedule) *ScheduleResponse {
	return &ScheduleResponse{
		ID:          schedule.ID,
		SiteID:      schedule.SiteID,
		CronExpr:    schedule.CronExpr,
		PostsPerDay: schedule.PostsPerDay,
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
		Excerpt:     article.Excerpt,
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

// Success response helpers
func SuccessResponse(data interface{}) *BaseResponse {
	return &BaseResponse{
		Success: true,
		Data:    data,
	}
}

func SuccessMessageResponse(message string, data interface{}) *BaseResponse {
	return &BaseResponse{
		Success: true,
		Message: message,
		Data:    data,
	}
}

func ErrorResponse(err error) *BaseResponse {
	return &BaseResponse{
		Success: false,
		Error:   err.Error(),
	}
}

func ErrorMessageResponse(message string, err error) *BaseResponse {
	return &BaseResponse{
		Success: false,
		Message: message,
		Error:   err.Error(),
	}
}
