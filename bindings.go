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

// GetTopic retrieves a single topic by ID
func (a *App) GetTopic(topicID int64) (*dto.TopicResponse, error) {
	return a.handlers.GetTopic(topicID)
}

// GetTopics retrieves all topics with pagination
func (a *App) GetTopics(pagination dto.PaginationRequest) (*dto.TopicListResponse, error) {
	return a.handlers.GetTopics(pagination)
}

// GetTopicsBySiteID retrieves topics associated with a specific site
func (a *App) GetTopicsBySiteID(siteID int64, pagination dto.PaginationRequest) (*dto.TopicListResponse, error) {
	return a.handlers.GetTopicsBySiteID(siteID, pagination)
}

// UpdateTopic updates an existing topic
func (a *App) UpdateTopic(req dto.UpdateTopicRequest) (*dto.TopicResponse, error) {
	return a.handlers.UpdateTopic(req)
}

// DeleteTopic deletes a topic
func (a *App) DeleteTopic(topicID int64) error {
	return a.handlers.DeleteTopic(topicID)
}

// ActivateTopic activates a topic
func (a *App) ActivateTopic(topicID int64) error {
	return a.handlers.ActivateTopic(topicID)
}

// DeactivateTopic deactivates a topic
func (a *App) DeactivateTopic(topicID int64) error {
	return a.handlers.DeactivateTopic(topicID)
}

// GetActiveTopics retrieves all active topics
func (a *App) GetActiveTopics() ([]*dto.TopicResponse, error) {
	return a.handlers.GetActiveTopics()
}

// SiteTopic Management Methods

// CreateSiteTopic creates a new site-topic association
func (a *App) CreateSiteTopic(req dto.CreateSiteTopicRequest) (*dto.SiteTopicResponse, error) {
	return a.handlers.CreateSiteTopic(req)
}

// GetSiteTopics retrieves all topics for a specific site
func (a *App) GetSiteTopics(siteID int64, pagination dto.PaginationRequest) (*dto.SiteTopicListResponse, error) {
	return a.handlers.GetSiteTopics(siteID, pagination)
}

// GetTopicSites retrieves all sites for a specific topic
func (a *App) GetTopicSites(topicID int64, pagination dto.PaginationRequest) (*dto.SiteTopicListResponse, error) {
	return a.handlers.GetTopicSites(topicID, pagination)
}

// UpdateSiteTopic updates a site-topic association
func (a *App) UpdateSiteTopic(req dto.UpdateSiteTopicRequest) (*dto.SiteTopicResponse, error) {
	return a.handlers.UpdateSiteTopic(req)
}

// DeleteSiteTopic deletes a site-topic association
func (a *App) DeleteSiteTopic(siteTopicID int64) error {
	return a.handlers.DeleteSiteTopic(siteTopicID)
}

// DeleteSiteTopicBySiteAndTopic deletes site-topic association by site and topic IDs
func (a *App) DeleteSiteTopicBySiteAndTopic(siteID int64, topicID int64) error {
	return a.handlers.DeleteSiteTopicBySiteAndTopic(siteID, topicID)
}

// ActivateSiteTopic activates a site-topic association
func (a *App) ActivateSiteTopic(siteTopicID int64) error {
	return a.handlers.ActivateSiteTopic(siteTopicID)
}

// DeactivateSiteTopic deactivates a site-topic association
func (a *App) DeactivateSiteTopic(siteTopicID int64) error {
	return a.handlers.DeactivateSiteTopic(siteTopicID)
}

// Topic Strategy and Selection Methods

// SelectTopicForSite selects a topic for article generation using the specified strategy
func (a *App) SelectTopicForSite(req dto.TopicSelectionRequest) (*dto.TopicSelectionResponse, error) {
	return a.handlers.SelectTopicForSite(req)
}

// GetTopicStats retrieves topic statistics for a site
func (a *App) GetTopicStats(siteID int64) (*dto.TopicStatsResponse, error) {
	return a.handlers.GetTopicStats(siteID)
}

// GetTopicUsageHistory retrieves usage history for a specific topic on a site
func (a *App) GetTopicUsageHistory(siteID int64, topicID int64, pagination dto.PaginationRequest) (*dto.TopicUsageListResponse, error) {
	return a.handlers.GetTopicUsageHistory(siteID, topicID, pagination)
}

// GetSiteUsageHistory retrieves all topic usage history for a site
func (a *App) GetSiteUsageHistory(siteID int64, pagination dto.PaginationRequest) (*dto.TopicUsageListResponse, error) {
	return a.handlers.GetSiteUsageHistory(siteID, pagination)
}

// CheckStrategyAvailability checks if more topics are available for a strategy
func (a *App) CheckStrategyAvailability(siteID int64, strategy string) (*dto.StrategyAvailabilityResponse, error) {
	return a.handlers.CheckStrategyAvailability(siteID, strategy)
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
