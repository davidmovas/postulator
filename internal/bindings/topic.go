package bindings

import (
	"Postulator/internal/dto"
)

// CreateTopic creates a new topic
func (b *Binder) CreateTopic(req dto.CreateTopicRequest) (*dto.TopicResponse, error) {
	return b.handler.CreateTopic(req)
}

// GetTopic retrieves a single topic by ID
func (b *Binder) GetTopic(topicID int64) (*dto.TopicResponse, error) {
	return b.handler.GetTopic(topicID)
}

// GetTopics retrieves all topics with pagination
func (b *Binder) GetTopics(pagination dto.PaginationRequest) (*dto.TopicListResponse, error) {
	return b.handler.GetTopics(pagination)
}

// GetTopicsBySiteID retrieves topics associated with a specific site
func (b *Binder) GetTopicsBySiteID(siteID int64, pagination dto.PaginationRequest) (*dto.TopicListResponse, error) {
	return b.handler.GetTopicsBySiteID(siteID, pagination)
}

// UpdateTopic updates an existing topic
func (b *Binder) UpdateTopic(req dto.UpdateTopicRequest) (*dto.TopicResponse, error) {
	return b.handler.UpdateTopic(req)
}

// DeleteTopic deletes a topic by ID
func (b *Binder) DeleteTopic(topicID int64) error {
	return b.handler.DeleteTopic(topicID)
}

// SiteTopic operations

// CreateSiteTopic creates a new site-topic relationship
func (b *Binder) CreateSiteTopic(req dto.CreateSiteTopicRequest) (*dto.SiteTopicResponse, error) {
	return b.handler.CreateSiteTopic(req)
}

// GetSiteTopics retrieves topics for a specific site
func (b *Binder) GetSiteTopics(siteID int64, pagination dto.PaginationRequest) (*dto.SiteTopicListResponse, error) {
	return b.handler.GetSiteTopics(siteID, pagination)
}

// GetTopicSites retrieves sites associated with a specific topic
func (b *Binder) GetTopicSites(topicID int64, pagination dto.PaginationRequest) (*dto.SiteTopicListResponse, error) {
	return b.handler.GetTopicSites(topicID, pagination)
}

// UpdateSiteTopic updates an existing site-topic relationship
func (b *Binder) UpdateSiteTopic(req dto.UpdateSiteTopicRequest) (*dto.SiteTopicResponse, error) {
	return b.handler.UpdateSiteTopic(req)
}

// DeleteSiteTopic deletes a site-topic relationship by ID
func (b *Binder) DeleteSiteTopic(siteTopicID int64) error {
	return b.handler.DeleteSiteTopic(siteTopicID)
}

// DeleteSiteTopicBySiteAndTopic deletes a site-topic relationship by site and topic IDs
func (b *Binder) DeleteSiteTopicBySiteAndTopic(siteID int64, topicID int64) error {
	return b.handler.DeleteSiteTopicBySiteAndTopic(siteID, topicID)
}

// Topic Selection and Statistics

// SelectTopicForSite selects a topic for a site using a specific strategy
func (b *Binder) SelectTopicForSite(req dto.TopicSelectionRequest) (*dto.TopicSelectionResponse, error) {
	return b.handler.SelectTopicForSite(req)
}

// GetTopicStats retrieves topic statistics for a site
func (b *Binder) GetTopicStats(siteID int64) (*dto.TopicStatsResponse, error) {
	return b.handler.GetTopicStats(siteID)
}

// GetTopicUsageHistory retrieves usage history for a specific topic on a site
func (b *Binder) GetTopicUsageHistory(siteID int64, topicID int64, pagination dto.PaginationRequest) (*dto.TopicUsageListResponse, error) {
	return b.handler.GetTopicUsageHistory(siteID, topicID, pagination)
}

// GetSiteUsageHistory retrieves usage history for all topics on a site
func (b *Binder) GetSiteUsageHistory(siteID int64, pagination dto.PaginationRequest) (*dto.TopicUsageListResponse, error) {
	return b.handler.GetSiteUsageHistory(siteID, pagination)
}

// CheckStrategyAvailability checks if a topic selection strategy is available for a site
func (b *Binder) CheckStrategyAvailability(siteID int64, strategy string) (*dto.StrategyAvailabilityResponse, error) {
	return b.handler.CheckStrategyAvailability(siteID, strategy)
}

// Bulk Operations

// TopicsImport imports topics from file content with support for txt, csv, jsonl formats
func (b *Binder) TopicsImport(siteID int64, req dto.TopicsImportRequest) (interface{}, error) {
	return b.handler.TopicsImport(siteID, req)
}

// TopicsReassign reassigns topics from one site to another
func (b *Binder) TopicsReassign(req dto.TopicsReassignRequest) (*dto.ReassignResult, error) {
	return b.handler.TopicsReassign(req)
}
