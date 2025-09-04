package handlers

import (
	"context"
	"fmt"
	"log"
	"time"

	"Postulator/internal/dto"
	"Postulator/internal/repository"
	"Postulator/internal/services/gpt"
	"Postulator/internal/services/pipeline"
	"Postulator/internal/services/wordpress"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// Handler contains all Wails API handlers
type Handler struct {
	repos      *repository.RepositoryContainer
	gptService *gpt.Service
	wpService  *wordpress.Service
	pipeline   *pipeline.Service
	appContext context.Context
}

// NewHandler creates a new handler instance
func NewHandler(repos *repository.RepositoryContainer, gptService *gpt.Service, wpService *wordpress.Service, pipeline *pipeline.Service, appContext context.Context) *Handler {
	return &Handler{
		repos:      repos,
		gptService: gptService,
		wpService:  wpService,
		pipeline:   pipeline,
		appContext: appContext,
	}
}

// Site Handlers

// CreateSite creates a new WordPress site
func (h *Handler) CreateSite(req dto.CreateSiteRequest) *dto.BaseResponse {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Validate request
	if err := h.validateSiteRequest(req.Name, req.URL, req.Username, req.Password); err != nil {
		return dto.ErrorResponse(err)
	}

	// Convert to model and create
	site := req.ToModel()
	if err := h.repos.Site.Create(ctx, site); err != nil {
		return dto.ErrorMessageResponse("Failed to create site", err)
	}

	// Test connection in background
	go func() {
		if err := h.wpService.TestConnection(context.Background(), site); err != nil {
			h.repos.Site.UpdateStatus(context.Background(), site.ID, "error")
		} else {
			h.repos.Site.UpdateStatus(context.Background(), site.ID, "connected")
		}
	}()

	response := dto.SiteToResponse(site)
	h.emitEvent("site:created", response)

	return dto.SuccessMessageResponse("Site created successfully", response)
}

// GetSites retrieves all sites with pagination
func (h *Handler) GetSites(pagination dto.PaginationRequest) *dto.BaseResponse {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Set defaults
	if pagination.Page <= 0 {
		pagination.Page = 1
	}
	if pagination.Limit <= 0 {
		pagination.Limit = 10
	}

	offset := (pagination.Page - 1) * pagination.Limit

	sites, err := h.repos.Site.List(ctx, pagination.Limit, offset)
	if err != nil {
		return dto.ErrorMessageResponse("Failed to retrieve sites", err)
	}

	total, err := h.repos.Site.Count(ctx)
	if err != nil {
		return dto.ErrorMessageResponse("Failed to count sites", err)
	}

	// Convert to response DTOs
	siteResponses := make([]*dto.SiteResponse, len(sites))
	for i, site := range sites {
		siteResponses[i] = dto.SiteToResponse(site)
	}

	response := &dto.SiteListResponse{
		Sites: siteResponses,
		Pagination: &dto.PaginationResponse{
			Page:       pagination.Page,
			Limit:      pagination.Limit,
			Total:      total,
			TotalPages: int((total + int64(pagination.Limit) - 1) / int64(pagination.Limit)),
		},
	}

	return dto.SuccessResponse(response)
}

// UpdateSite updates an existing site
func (h *Handler) UpdateSite(req dto.UpdateSiteRequest) *dto.BaseResponse {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Validate request
	if err := h.validateSiteRequest(req.Name, req.URL, req.Username, req.Password); err != nil {
		return dto.ErrorResponse(err)
	}

	site := req.ToModel()
	if err := h.repos.Site.Update(ctx, site); err != nil {
		return dto.ErrorMessageResponse("Failed to update site", err)
	}

	// Test connection in background if site is active
	if req.IsActive {
		go func() {
			if err := h.wpService.TestConnection(context.Background(), site); err != nil {
				h.repos.Site.UpdateStatus(context.Background(), site.ID, "error")
			} else {
				h.repos.Site.UpdateStatus(context.Background(), site.ID, "connected")
			}
		}()
	}

	response := dto.SiteToResponse(site)
	h.emitEvent("site:updated", response)

	return dto.SuccessMessageResponse("Site updated successfully", response)
}

// DeleteSite deletes a site
func (h *Handler) DeleteSite(siteID int64) *dto.BaseResponse {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := h.repos.Site.Delete(ctx, siteID); err != nil {
		return dto.ErrorMessageResponse("Failed to delete site", err)
	}

	h.emitEvent("site:deleted", map[string]interface{}{"site_id": siteID})
	return dto.SuccessMessageResponse("Site deleted successfully", nil)
}

// TestSiteConnection tests connection to a WordPress site
func (h *Handler) TestSiteConnection(req dto.TestSiteConnectionRequest) *dto.BaseResponse {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	site, err := h.repos.Site.GetByID(ctx, req.SiteID)
	if err != nil {
		return dto.ErrorMessageResponse("Failed to get site", err)
	}
	if site == nil {
		return dto.ErrorResponse(fmt.Errorf("site not found"))
	}

	start := time.Now()
	err = h.wpService.TestConnection(ctx, site)

	response := &dto.TestConnectionResponse{
		Success:   err == nil,
		Timestamp: time.Now(),
	}

	if err != nil {
		response.Status = "error"
		response.Message = "Connection failed"
		response.Details = err.Error()
		// Update site status
		h.repos.Site.UpdateStatus(ctx, req.SiteID, "error")
	} else {
		response.Status = "connected"
		response.Message = fmt.Sprintf("Connection successful (%.2fs)", time.Since(start).Seconds())
		// Update site status
		h.repos.Site.UpdateStatus(ctx, req.SiteID, "connected")
	}

	return dto.SuccessResponse(response)
}

// Topic Handlers

// CreateTopic creates a new topic
func (h *Handler) CreateTopic(req dto.CreateTopicRequest) *dto.BaseResponse {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Validate request
	if err := dto.ValidateRequired(req.Title, "title"); err != nil {
		return dto.ErrorResponse(err)
	}

	topic := req.ToModel()
	if err := h.repos.Topic.Create(ctx, topic); err != nil {
		return dto.ErrorMessageResponse("Failed to create topic", err)
	}

	response := dto.TopicToResponse(topic)
	h.emitEvent("topic:created", response)

	return dto.SuccessMessageResponse("Topic created successfully", response)
}

// GetTopics retrieves all topics with pagination
func (h *Handler) GetTopics(pagination dto.PaginationRequest) *dto.BaseResponse {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if pagination.Page <= 0 {
		pagination.Page = 1
	}
	if pagination.Limit <= 0 {
		pagination.Limit = 10
	}

	offset := (pagination.Page - 1) * pagination.Limit

	topics, err := h.repos.Topic.List(ctx, pagination.Limit, offset)
	if err != nil {
		return dto.ErrorMessageResponse("Failed to retrieve topics", err)
	}

	total, err := h.repos.Topic.Count(ctx)
	if err != nil {
		return dto.ErrorMessageResponse("Failed to count topics", err)
	}

	// Convert to response DTOs
	topicResponses := make([]*dto.TopicResponse, len(topics))
	for i, topic := range topics {
		topicResponses[i] = dto.TopicToResponse(topic)
	}

	response := &dto.TopicListResponse{
		Topics: topicResponses,
		Pagination: &dto.PaginationResponse{
			Page:       pagination.Page,
			Limit:      pagination.Limit,
			Total:      total,
			TotalPages: int((total + int64(pagination.Limit) - 1) / int64(pagination.Limit)),
		},
	}

	return dto.SuccessResponse(response)
}

// Schedule Handlers

// CreateSchedule creates a new posting schedule
func (h *Handler) CreateSchedule(req dto.CreateScheduleRequest) *dto.BaseResponse {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Validate cron expression
	if err := dto.ValidateCronExpression(req.CronExpr); err != nil {
		return dto.ErrorResponse(err)
	}

	schedule := req.ToModel()
	if err := h.repos.Schedule.Create(ctx, schedule); err != nil {
		return dto.ErrorMessageResponse("Failed to create schedule", err)
	}

	response := dto.ScheduleToResponse(schedule)
	h.emitEvent("schedule:created", response)

	return dto.SuccessMessageResponse("Schedule created successfully", response)
}

// GetSchedules retrieves all schedules with pagination
func (h *Handler) GetSchedules(pagination dto.PaginationRequest) *dto.BaseResponse {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if pagination.Page <= 0 {
		pagination.Page = 1
	}
	if pagination.Limit <= 0 {
		pagination.Limit = 10
	}

	offset := (pagination.Page - 1) * pagination.Limit

	schedules, err := h.repos.Schedule.List(ctx, pagination.Limit, offset)
	if err != nil {
		return dto.ErrorMessageResponse("Failed to retrieve schedules", err)
	}

	total, err := h.repos.Schedule.Count(ctx)
	if err != nil {
		return dto.ErrorMessageResponse("Failed to count schedules", err)
	}

	// Convert to response DTOs
	scheduleResponses := make([]*dto.ScheduleResponse, len(schedules))
	for i, schedule := range schedules {
		scheduleResponses[i] = dto.ScheduleToResponse(schedule)
	}

	response := &dto.ScheduleListResponse{
		Schedules: scheduleResponses,
		Pagination: &dto.PaginationResponse{
			Page:       pagination.Page,
			Limit:      pagination.Limit,
			Total:      total,
			TotalPages: int((total + int64(pagination.Limit) - 1) / int64(pagination.Limit)),
		},
	}

	return dto.SuccessResponse(response)
}

// Article Handlers

// CreateArticle creates a new article manually
func (h *Handler) CreateArticle(req dto.CreateArticleManualRequest) *dto.BaseResponse {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	pipelineReq := pipeline.CreateArticleRequest{
		SiteID:   req.SiteID,
		TopicID:  req.TopicID,
		Publish:  req.Publish,
		Metadata: req.Metadata,
	}

	// Run the pipeline asynchronously
	go h.pipeline.ProcessCreateArticleJob(ctx, pipelineReq)

	return dto.SuccessMessageResponse("Article creation started", nil)
}

// GetArticles retrieves all articles with pagination
func (h *Handler) GetArticles(pagination dto.PaginationRequest) *dto.BaseResponse {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if pagination.Page <= 0 {
		pagination.Page = 1
	}
	if pagination.Limit <= 0 {
		pagination.Limit = 10
	}

	offset := (pagination.Page - 1) * pagination.Limit

	articles, err := h.repos.Article.List(ctx, pagination.Limit, offset)
	if err != nil {
		return dto.ErrorMessageResponse("Failed to retrieve articles", err)
	}

	total, err := h.repos.Article.Count(ctx)
	if err != nil {
		return dto.ErrorMessageResponse("Failed to count articles", err)
	}

	// Convert to response DTOs
	articleResponses := make([]*dto.ArticleResponse, len(articles))
	for i, article := range articles {
		articleResponses[i] = dto.ArticleToResponse(article)
	}

	response := &dto.ArticleListResponse{
		Articles: articleResponses,
		Pagination: &dto.PaginationResponse{
			Page:       pagination.Page,
			Limit:      pagination.Limit,
			Total:      total,
			TotalPages: int((total + int64(pagination.Limit) - 1) / int64(pagination.Limit)),
		},
	}

	return dto.SuccessResponse(response)
}

// PreviewArticle generates a preview of an article without saving
func (h *Handler) PreviewArticle(req dto.PreviewArticleRequest) *dto.BaseResponse {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	gptResponse, err := h.pipeline.GeneratePreviewArticle(ctx, req.SiteID, req.TopicID)
	if err != nil {
		return dto.ErrorMessageResponse("Failed to generate article preview", err)
	}

	response := &dto.PreviewArticleResponse{
		Title:      gptResponse.Title,
		Content:    gptResponse.Content,
		Excerpt:    gptResponse.Excerpt,
		Keywords:   gptResponse.Keywords,
		Tags:       gptResponse.Tags,
		Category:   gptResponse.Category,
		TokensUsed: gptResponse.TokensUsed,
		Model:      gptResponse.Model,
	}

	return dto.SuccessResponse(response)
}

// PostingJob Handlers

// GetPostingJobs retrieves all posting jobs with pagination
func (h *Handler) GetPostingJobs(pagination dto.PaginationRequest) *dto.BaseResponse {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if pagination.Page <= 0 {
		pagination.Page = 1
	}
	if pagination.Limit <= 0 {
		pagination.Limit = 10
	}

	offset := (pagination.Page - 1) * pagination.Limit

	jobs, err := h.repos.PostingJob.List(ctx, pagination.Limit, offset)
	if err != nil {
		return dto.ErrorMessageResponse("Failed to retrieve posting jobs", err)
	}

	total, err := h.repos.PostingJob.Count(ctx)
	if err != nil {
		return dto.ErrorMessageResponse("Failed to count posting jobs", err)
	}

	// Convert to response DTOs
	jobResponses := make([]*dto.PostingJobResponse, len(jobs))
	for i, job := range jobs {
		jobResponses[i] = dto.PostingJobToResponse(job)
	}

	response := &dto.PostingJobListResponse{
		Jobs: jobResponses,
		Pagination: &dto.PaginationResponse{
			Page:       pagination.Page,
			Limit:      pagination.Limit,
			Total:      total,
			TotalPages: int((total + int64(pagination.Limit) - 1) / int64(pagination.Limit)),
		},
	}

	return dto.SuccessResponse(response)
}

// Dashboard Handlers

// GetDashboard retrieves dashboard data
func (h *Handler) GetDashboard() *dto.BaseResponse {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	stats := &dto.DashboardStats{}

	// Get statistics
	if total, err := h.repos.Site.Count(ctx); err == nil {
		stats.TotalSites = total
	}

	if active, err := h.repos.Site.GetActive(ctx); err == nil {
		stats.ActiveSites = int64(len(active))
	}

	if total, err := h.repos.Topic.Count(ctx); err == nil {
		stats.TotalTopics = total
	}

	if total, err := h.repos.Article.Count(ctx); err == nil {
		stats.TotalArticles = total
	}

	if published, err := h.repos.Article.GetByStatus(ctx, "published", 0, 0); err == nil {
		stats.PublishedArticles = int64(len(published))
	}

	if pending, err := h.repos.PostingJob.GetPending(ctx); err == nil {
		stats.PendingJobs = int64(len(pending))
	}

	if running, err := h.repos.PostingJob.GetRunning(ctx); err == nil {
		stats.RunningJobs = int64(len(running))
	}

	// Get recent activities (simplified)
	recentActivities := []*dto.RecentActivity{}

	// Get upcoming schedules
	upcomingSchedules := []*dto.ScheduleResponse{}
	if schedules, err := h.repos.Schedule.GetActive(ctx); err == nil && len(schedules) > 0 {
		for _, schedule := range schedules[:min(5, len(schedules))] {
			upcomingSchedules = append(upcomingSchedules, dto.ScheduleToResponse(schedule))
		}
	}

	response := &dto.DashboardResponse{
		Stats:             stats,
		RecentActivities:  recentActivities,
		UpcomingSchedules: upcomingSchedules,
	}

	return dto.SuccessResponse(response)
}

// Settings Handlers

// GetSettings retrieves all settings
func (h *Handler) GetSettings() *dto.BaseResponse {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	settings, err := h.repos.Setting.GetAll(ctx)
	if err != nil {
		return dto.ErrorMessageResponse("Failed to retrieve settings", err)
	}

	settingResponses := make([]*dto.SettingResponse, len(settings))
	for i, setting := range settings {
		settingResponses[i] = &dto.SettingResponse{
			Key:       setting.Key,
			Value:     setting.Value,
			UpdatedAt: setting.UpdatedAt,
		}
	}

	response := &dto.SettingsResponse{
		Settings: settingResponses,
	}

	return dto.SuccessResponse(response)
}

// UpdateSetting updates a setting
func (h *Handler) UpdateSetting(req dto.SettingRequest) *dto.BaseResponse {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := h.repos.Setting.Set(ctx, req.Key, req.Value); err != nil {
		return dto.ErrorMessageResponse("Failed to update setting", err)
	}

	h.emitEvent("setting:updated", map[string]interface{}{
		"key":   req.Key,
		"value": req.Value,
	})

	return dto.SuccessMessageResponse("Setting updated successfully", nil)
}

// Utility functions

func (h *Handler) validateSiteRequest(name, url, username, password string) error {
	if err := dto.ValidateRequired(name, "name"); err != nil {
		return err
	}
	if err := dto.ValidateRequired(url, "url"); err != nil {
		return err
	}
	if err := dto.ValidateURL(url); err != nil {
		return err
	}
	if err := dto.ValidateRequired(username, "username"); err != nil {
		return err
	}
	if err := dto.ValidateRequired(password, "password"); err != nil {
		return err
	}
	return nil
}

func (h *Handler) emitEvent(eventName string, data interface{}) {
	if h.appContext != nil {
		runtime.EventsEmit(h.appContext, eventName, data)
		log.Printf("Event emitted: %s", eventName)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
