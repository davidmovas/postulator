package main

import (
	"Postulator/internal/dto"
	"Postulator/internal/repository"
)

// Site Management Methods - Wails API bindings

// CreateSite creates a new WordPress site
func (a *App) CreateSite(req dto.CreateSiteRequest) (*dto.SiteResponse, error) {
	return a.handlers.CreateSite(req)
}

// GetSite retrieves a single site by ID
func (a *App) GetSite(siteID int64) (*dto.SiteResponse, error) {
	return a.handlers.GetSite(siteID)
}

// GetSites retrieves all sites with pagination
func (a *App) GetSites(pagination dto.PaginationRequest) (*dto.SiteListResponse, error) {
	return a.handlers.GetSites(pagination)
}

// UpdateSite updates an existing site
func (a *App) UpdateSite(req dto.UpdateSiteRequest) (*dto.SiteResponse, error) {
	return a.handlers.UpdateSite(req)
}

// DeleteSite deletes a site
func (a *App) DeleteSite(siteID int64) error {
	return a.handlers.DeleteSite(siteID)
}

// ActivateSite activates a site
func (a *App) ActivateSite(siteID int64) error {
	return a.handlers.ActivateSite(siteID)
}

// DeactivateSite deactivates a site
func (a *App) DeactivateSite(siteID int64) error {
	return a.handlers.DeactivateSite(siteID)
}

// SetSiteCheckStatus updates the check status of a site
func (a *App) SetSiteCheckStatus(siteID int64, status string) error {
	return a.handlers.SetSiteCheckStatus(siteID, status)
}

// TestSiteConnection tests connection to a WordPress site
func (a *App) TestSiteConnection(req dto.TestSiteConnectionRequest) (*dto.TestConnectionResponse, error) {
	return a.handlers.TestSiteConnection(req)
}

// Topic Management Methods

// CreateTopic creates a new topic
func (a *App) CreateTopic(req dto.CreateTopicRequest) (*dto.TopicResponse, error) {
	return a.handlers.CreateTopic(req)
}

// GetTopics retrieves all topics with pagination
func (a *App) GetTopics(pagination dto.PaginationRequest) (*dto.TopicListResponse, error) {
	return a.handlers.GetTopics(pagination)
}

// Schedule Management Methods

// CreateSchedule creates a new posting schedule
func (a *App) CreateSchedule(req dto.CreateScheduleRequest) (*dto.ScheduleResponse, error) {
	return a.handlers.CreateSchedule(req)
}

// GetSchedules retrieves all schedules with pagination
func (a *App) GetSchedules(pagination dto.PaginationRequest) (*dto.ScheduleListResponse, error) {
	return a.handlers.GetSchedules(pagination)
}

// Article Management Methods

// CreateArticle creates a new article manually
func (a *App) CreateArticle(req dto.CreateArticleManualRequest) (*dto.ArticleResponse, error) {
	return a.handlers.CreateArticle(req)
}

// GetArticles retrieves all articles with pagination
func (a *App) GetArticles(pagination dto.PaginationRequest) (*dto.ArticleListResponse, error) {
	return a.handlers.GetArticles(pagination)
}

// PreviewArticle generates a preview of an article without saving
func (a *App) PreviewArticle(req dto.PreviewArticleRequest) (*dto.PreviewArticleResponse, error) {
	return a.handlers.PreviewArticle(req)
}

// PostingJob Management Methods

// GetPostingJobs retrieves all posting jobs with pagination
func (a *App) GetPostingJobs(pagination dto.PaginationRequest) (*dto.PostingJobListResponse, error) {
	return a.handlers.GetPostingJobs(pagination)
}

// Dashboard Methods

// GetDashboard retrieves dashboard data
func (a *App) GetDashboard() (*dto.DashboardResponse, error) {
	return a.handlers.GetDashboard()
}

// Settings Management Methods

// GetSettings retrieves all settings
func (a *App) GetSettings() (*dto.SettingsResponse, error) {
	return a.handlers.GetSettings()
}

// GetSetting retrieves a setting value
func (a *App) GetSetting(key string) (string, error) {
	setting, err := repository.GetSetting(key)
	if err != nil {
		return "", err
	}
	return setting.Value, err
}

// SetSetting sets a setting value
func (a *App) SetSetting(key, value string) error {
	return repository.SetSetting(key, value)
}

// UpdateSetting updates a setting
func (a *App) UpdateSetting(req dto.SettingRequest) (*dto.SettingResponse, error) {
	return a.handlers.UpdateSetting(req)
}
