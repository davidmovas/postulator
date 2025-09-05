package handlers

import (
	"Postulator/internal/dto"
	"Postulator/internal/models"
	"fmt"
)

// Topic Handlers

// CreateTopic creates a new topic
func (h *Handler) CreateTopic(req dto.CreateTopicRequest) (*dto.TopicResponse, error) {
	ctx := h.fastCtx()

	// Validate request
	if err := dto.ValidateRequired(req.Title, "title"); err != nil {
		return nil, err
	}

	// Convert to model and create
	topic := req.ToModel()
	createdTopic, err := h.repo.CreateTopic(ctx, topic)
	if err != nil {
		return nil, fmt.Errorf("failed to create topic: %w", err)
	}

	return dto.TopicToResponse(createdTopic), nil
}

// GetTopic retrieves a single topic by ID
func (h *Handler) GetTopic(topicID int64) (*dto.TopicResponse, error) {
	ctx := h.fastCtx()

	if topicID <= 0 {
		return nil, fmt.Errorf("invalid topic ID")
	}

	topic, err := h.repo.GetTopic(ctx, topicID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve topic: %w", err)
	}

	return dto.TopicToResponse(topic), nil
}

// GetTopics retrieves all topics with pagination
func (h *Handler) GetTopics(pagination dto.PaginationRequest) (*dto.TopicListResponse, error) {
	ctx := h.fastCtx()

	// Set defaults
	if pagination.Page <= 0 {
		pagination.Page = 1
	}
	if pagination.Limit <= 0 {
		pagination.Limit = 10
	}

	offset := (pagination.Page - 1) * pagination.Limit

	result, err := h.repo.GetTopics(ctx, pagination.Limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve topics: %w", err)
	}

	// Convert to response DTOs
	topicResponses := make([]*dto.TopicResponse, len(result.Data))
	for i, topic := range result.Data {
		topicResponses[i] = dto.TopicToResponse(topic)
	}

	totalPages := (result.Total + pagination.Limit - 1) / pagination.Limit

	response := &dto.TopicListResponse{
		Topics: topicResponses,
		Pagination: &dto.PaginationResponse{
			Page:       pagination.Page,
			Limit:      pagination.Limit,
			Total:      int64(result.Total),
			TotalPages: totalPages,
		},
	}

	return response, nil
}

// GetTopicsBySiteID retrieves topics associated with a specific site
func (h *Handler) GetTopicsBySiteID(siteID int64, pagination dto.PaginationRequest) (*dto.TopicListResponse, error) {
	ctx := h.fastCtx()

	if siteID <= 0 {
		return nil, fmt.Errorf("invalid site ID")
	}

	// Set defaults
	if pagination.Page <= 0 {
		pagination.Page = 1
	}
	if pagination.Limit <= 0 {
		pagination.Limit = 10
	}

	offset := (pagination.Page - 1) * pagination.Limit

	result, err := h.repo.GetTopicsBySiteID(ctx, siteID, pagination.Limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve topics by site: %w", err)
	}

	// Convert to response DTOs
	topicResponses := make([]*dto.TopicResponse, len(result.Data))
	for i, topic := range result.Data {
		topicResponses[i] = dto.TopicToResponse(topic)
	}

	totalPages := (result.Total + pagination.Limit - 1) / pagination.Limit

	response := &dto.TopicListResponse{
		Topics: topicResponses,
		Pagination: &dto.PaginationResponse{
			Page:       pagination.Page,
			Limit:      pagination.Limit,
			Total:      int64(result.Total),
			TotalPages: totalPages,
		},
	}

	return response, nil
}

// UpdateTopic updates an existing topic
func (h *Handler) UpdateTopic(req dto.UpdateTopicRequest) (*dto.TopicResponse, error) {
	ctx := h.fastCtx()

	// Validate request
	if req.ID <= 0 {
		return nil, fmt.Errorf("invalid topic ID")
	}
	if err := dto.ValidateRequired(req.Title, "title"); err != nil {
		return nil, err
	}

	// Convert to model and update
	topic := req.ToModel()
	updatedTopic, err := h.repo.UpdateTopic(ctx, topic)
	if err != nil {
		return nil, fmt.Errorf("failed to update topic: %w", err)
	}

	return dto.TopicToResponse(updatedTopic), nil
}

// DeleteTopic deletes a topic
func (h *Handler) DeleteTopic(topicID int64) error {
	ctx := h.fastCtx()

	if topicID <= 0 {
		return fmt.Errorf("invalid topic ID")
	}

	err := h.repo.DeleteTopic(ctx, topicID)
	if err != nil {
		return fmt.Errorf("failed to delete topic: %w", err)
	}

	return nil
}

// ActivateTopic activates a topic
func (h *Handler) ActivateTopic(topicID int64) error {
	ctx := h.fastCtx()

	if topicID <= 0 {
		return fmt.Errorf("invalid topic ID")
	}

	err := h.repo.ActivateTopic(ctx, topicID)
	if err != nil {
		return fmt.Errorf("failed to activate topic: %w", err)
	}

	return nil
}

// DeactivateTopic deactivates a topic
func (h *Handler) DeactivateTopic(topicID int64) error {
	ctx := h.fastCtx()

	if topicID <= 0 {
		return fmt.Errorf("invalid topic ID")
	}

	err := h.repo.DeactivateTopic(ctx, topicID)
	if err != nil {
		return fmt.Errorf("failed to deactivate topic: %w", err)
	}

	return nil
}

// GetActiveTopics retrieves all active topics
func (h *Handler) GetActiveTopics() ([]*dto.TopicResponse, error) {
	ctx := h.fastCtx()

	topics, err := h.repo.GetActiveTopics(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve active topics: %w", err)
	}

	// Convert to response DTOs
	topicResponses := make([]*dto.TopicResponse, len(topics))
	for i, topic := range topics {
		topicResponses[i] = dto.TopicToResponse(topic)
	}

	return topicResponses, nil
}

// SiteTopic Handlers

// CreateSiteTopic creates a new site-topic association
func (h *Handler) CreateSiteTopic(req dto.CreateSiteTopicRequest) (*dto.SiteTopicResponse, error) {
	ctx := h.fastCtx()

	// Validate request
	if req.SiteID <= 0 {
		return nil, fmt.Errorf("invalid site ID")
	}
	if req.TopicID <= 0 {
		return nil, fmt.Errorf("invalid topic ID")
	}

	// Convert to model and create
	siteTopic := &dto.CreateSiteTopicRequest{
		SiteID:   req.SiteID,
		TopicID:  req.TopicID,
		Priority: req.Priority,
		IsActive: req.IsActive,
	}

	siteTopicModel := &models.SiteTopic{
		SiteID:   siteTopic.SiteID,
		TopicID:  siteTopic.TopicID,
		IsActive: siteTopic.IsActive,
	}

	createdSiteTopic, err := h.repo.CreateSiteTopic(ctx, siteTopicModel)
	if err != nil {
		return nil, fmt.Errorf("failed to create site topic: %w", err)
	}

	return &dto.SiteTopicResponse{
		ID:       createdSiteTopic.ID,
		SiteID:   createdSiteTopic.SiteID,
		TopicID:  createdSiteTopic.TopicID,
		Priority: req.Priority,
		IsActive: createdSiteTopic.IsActive,
	}, nil
}

// GetSiteTopics retrieves all topics for a specific site
func (h *Handler) GetSiteTopics(siteID int64, pagination dto.PaginationRequest) (*dto.SiteTopicListResponse, error) {
	ctx := h.fastCtx()

	if siteID <= 0 {
		return nil, fmt.Errorf("invalid site ID")
	}

	// Set defaults
	if pagination.Page <= 0 {
		pagination.Page = 1
	}
	if pagination.Limit <= 0 {
		pagination.Limit = 10
	}

	offset := (pagination.Page - 1) * pagination.Limit

	result, err := h.repo.GetSiteTopics(ctx, siteID, pagination.Limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve site topics: %w", err)
	}

	// Convert to response DTOs
	siteTopicResponses := make([]*dto.SiteTopicResponse, len(result.Data))
	for i, siteTopic := range result.Data {
		siteTopicResponses[i] = &dto.SiteTopicResponse{
			ID:       siteTopic.ID,
			SiteID:   siteTopic.SiteID,
			TopicID:  siteTopic.TopicID,
			Priority: 1, // Default priority as it's not in the model
			IsActive: siteTopic.IsActive,
		}
	}

	totalPages := (result.Total + pagination.Limit - 1) / pagination.Limit

	response := &dto.SiteTopicListResponse{
		SiteTopics: siteTopicResponses,
		Pagination: &dto.PaginationResponse{
			Page:       pagination.Page,
			Limit:      pagination.Limit,
			Total:      int64(result.Total),
			TotalPages: totalPages,
		},
	}

	return response, nil
}

// GetTopicSites retrieves all sites for a specific topic
func (h *Handler) GetTopicSites(topicID int64, pagination dto.PaginationRequest) (*dto.SiteTopicListResponse, error) {
	ctx := h.fastCtx()

	if topicID <= 0 {
		return nil, fmt.Errorf("invalid topic ID")
	}

	// Set defaults
	if pagination.Page <= 0 {
		pagination.Page = 1
	}
	if pagination.Limit <= 0 {
		pagination.Limit = 10
	}

	offset := (pagination.Page - 1) * pagination.Limit

	result, err := h.repo.GetTopicSites(ctx, topicID, pagination.Limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve topic sites: %w", err)
	}

	// Convert to response DTOs
	siteTopicResponses := make([]*dto.SiteTopicResponse, len(result.Data))
	for i, siteTopic := range result.Data {
		siteTopicResponses[i] = &dto.SiteTopicResponse{
			ID:       siteTopic.ID,
			SiteID:   siteTopic.SiteID,
			TopicID:  siteTopic.TopicID,
			Priority: 1, // Default priority as it's not in the model
			IsActive: siteTopic.IsActive,
		}
	}

	totalPages := (result.Total + pagination.Limit - 1) / pagination.Limit

	response := &dto.SiteTopicListResponse{
		SiteTopics: siteTopicResponses,
		Pagination: &dto.PaginationResponse{
			Page:       pagination.Page,
			Limit:      pagination.Limit,
			Total:      int64(result.Total),
			TotalPages: totalPages,
		},
	}

	return response, nil
}

// UpdateSiteTopic updates a site-topic association
func (h *Handler) UpdateSiteTopic(req dto.UpdateSiteTopicRequest) (*dto.SiteTopicResponse, error) {
	ctx := h.fastCtx()

	// Validate request
	if req.ID <= 0 {
		return nil, fmt.Errorf("invalid site topic ID")
	}
	if req.SiteID <= 0 {
		return nil, fmt.Errorf("invalid site ID")
	}
	if req.TopicID <= 0 {
		return nil, fmt.Errorf("invalid topic ID")
	}

	// Convert to model and update
	siteTopicModel := &models.SiteTopic{
		ID:       req.ID,
		SiteID:   req.SiteID,
		TopicID:  req.TopicID,
		IsActive: req.IsActive,
	}

	updatedSiteTopic, err := h.repo.UpdateSiteTopic(ctx, siteTopicModel)
	if err != nil {
		return nil, fmt.Errorf("failed to update site topic: %w", err)
	}

	return &dto.SiteTopicResponse{
		ID:       updatedSiteTopic.ID,
		SiteID:   updatedSiteTopic.SiteID,
		TopicID:  updatedSiteTopic.TopicID,
		Priority: req.Priority,
		IsActive: updatedSiteTopic.IsActive,
	}, nil
}

// DeleteSiteTopic deletes a site-topic association
func (h *Handler) DeleteSiteTopic(siteTopicID int64) error {
	ctx := h.fastCtx()

	if siteTopicID <= 0 {
		return fmt.Errorf("invalid site topic ID")
	}

	err := h.repo.DeleteSiteTopic(ctx, siteTopicID)
	if err != nil {
		return fmt.Errorf("failed to delete site topic: %w", err)
	}

	return nil
}

// DeleteSiteTopicBySiteAndTopic deletes site-topic association by site and topic IDs
func (h *Handler) DeleteSiteTopicBySiteAndTopic(siteID int64, topicID int64) error {
	ctx := h.fastCtx()

	if siteID <= 0 {
		return fmt.Errorf("invalid site ID")
	}
	if topicID <= 0 {
		return fmt.Errorf("invalid topic ID")
	}

	err := h.repo.DeleteSiteTopicBySiteAndTopic(ctx, siteID, topicID)
	if err != nil {
		return fmt.Errorf("failed to delete site topic: %w", err)
	}

	return nil
}

// ActivateSiteTopic activates a site-topic association
func (h *Handler) ActivateSiteTopic(siteTopicID int64) error {
	ctx := h.fastCtx()

	if siteTopicID <= 0 {
		return fmt.Errorf("invalid site topic ID")
	}

	err := h.repo.ActivateSiteTopic(ctx, siteTopicID)
	if err != nil {
		return fmt.Errorf("failed to activate site topic: %w", err)
	}

	return nil
}

// DeactivateSiteTopic deactivates a site-topic association
func (h *Handler) DeactivateSiteTopic(siteTopicID int64) error {
	ctx := h.fastCtx()

	if siteTopicID <= 0 {
		return fmt.Errorf("invalid site topic ID")
	}

	err := h.repo.DeactivateSiteTopic(ctx, siteTopicID)
	if err != nil {
		return fmt.Errorf("failed to deactivate site topic: %w", err)
	}

	return nil
}

// Topic Strategy and Selection Handlers

// SelectTopicForSite selects a topic for article generation using the specified strategy
func (h *Handler) SelectTopicForSite(req dto.TopicSelectionRequest) (*dto.TopicSelectionResponse, error) {
	ctx := h.fastCtx()

	if req.SiteID <= 0 {
		return nil, fmt.Errorf("invalid site ID")
	}

	// Create models request
	modelReq := &models.TopicSelectionRequest{
		SiteID:   req.SiteID,
		Strategy: models.TopicSelectionStrategy(req.Strategy),
	}

	// Use topic strategy service to select topic
	result, err := h.topicStrategyService.SelectTopicForSite(ctx, modelReq)
	if err != nil {
		return nil, fmt.Errorf("failed to select topic: %w", err)
	}

	// Convert to response DTO
	response := &dto.TopicSelectionResponse{
		Topic: dto.TopicToResponse(result.Topic),
		SiteTopic: &dto.SiteTopicResponse{
			ID:         result.SiteTopic.ID,
			SiteID:     result.SiteTopic.SiteID,
			TopicID:    result.SiteTopic.TopicID,
			Priority:   result.SiteTopic.Priority,
			IsActive:   result.SiteTopic.IsActive,
			Strategy:   result.SiteTopic.Strategy,
			UsageCount: result.SiteTopic.UsageCount,
			LastUsedAt: result.SiteTopic.LastUsedAt,
		},
		Strategy:       result.Strategy,
		CanContinue:    result.CanContinue,
		RemainingCount: result.RemainingCount,
	}

	return response, nil
}

// GetTopicStats retrieves topic statistics for a site
func (h *Handler) GetTopicStats(siteID int64) (*dto.TopicStatsResponse, error) {
	ctx := h.fastCtx()

	if siteID <= 0 {
		return nil, fmt.Errorf("invalid site ID")
	}

	stats, err := h.repo.GetTopicStats(ctx, siteID)
	if err != nil {
		return nil, fmt.Errorf("failed to get topic stats: %w", err)
	}

	response := &dto.TopicStatsResponse{
		SiteID:             stats.SiteID,
		TotalTopics:        stats.TotalTopics,
		ActiveTopics:       stats.ActiveTopics,
		UsedTopics:         stats.UsedTopics,
		UnusedTopics:       stats.UnusedTopics,
		UniqueTopicsLeft:   stats.UniqueTopicsLeft,
		RoundRobinPosition: stats.RoundRobinPosition,
		MostUsedTopicID:    stats.MostUsedTopicID,
		MostUsedTopicCount: stats.MostUsedTopicCount,
		LastUsedTopicID:    stats.LastUsedTopicID,
		LastUsedAt:         stats.LastUsedAt,
	}

	return response, nil
}

// GetTopicUsageHistory retrieves usage history for a specific topic on a site
func (h *Handler) GetTopicUsageHistory(siteID int64, topicID int64, pagination dto.PaginationRequest) (*dto.TopicUsageListResponse, error) {
	ctx := h.fastCtx()

	if siteID <= 0 {
		return nil, fmt.Errorf("invalid site ID")
	}
	if topicID <= 0 {
		return nil, fmt.Errorf("invalid topic ID")
	}

	// Set defaults
	if pagination.Page <= 0 {
		pagination.Page = 1
	}
	if pagination.Limit <= 0 {
		pagination.Limit = 10
	}

	offset := (pagination.Page - 1) * pagination.Limit

	result, err := h.repo.GetTopicUsageHistory(ctx, siteID, topicID, pagination.Limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve topic usage history: %w", err)
	}

	// Convert to response DTOs
	usageResponses := make([]*dto.TopicUsageResponse, len(result.Data))
	for i, usage := range result.Data {
		usageResponses[i] = &dto.TopicUsageResponse{
			ID:        usage.ID,
			SiteID:    usage.SiteID,
			TopicID:   usage.TopicID,
			ArticleID: usage.ArticleID,
			Strategy:  usage.Strategy,
			UsedAt:    usage.UsedAt,
			CreatedAt: usage.CreatedAt,
		}
	}

	totalPages := (result.Total + pagination.Limit - 1) / pagination.Limit

	response := &dto.TopicUsageListResponse{
		UsageHistory: usageResponses,
		Pagination: &dto.PaginationResponse{
			Page:       pagination.Page,
			Limit:      pagination.Limit,
			Total:      int64(result.Total),
			TotalPages: totalPages,
		},
	}

	return response, nil
}

// GetSiteUsageHistory retrieves all topic usage history for a site
func (h *Handler) GetSiteUsageHistory(siteID int64, pagination dto.PaginationRequest) (*dto.TopicUsageListResponse, error) {
	ctx := h.fastCtx()

	if siteID <= 0 {
		return nil, fmt.Errorf("invalid site ID")
	}

	// Set defaults
	if pagination.Page <= 0 {
		pagination.Page = 1
	}
	if pagination.Limit <= 0 {
		pagination.Limit = 10
	}

	offset := (pagination.Page - 1) * pagination.Limit

	result, err := h.repo.GetSiteUsageHistory(ctx, siteID, pagination.Limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve site usage history: %w", err)
	}

	// Convert to response DTOs
	usageResponses := make([]*dto.TopicUsageResponse, len(result.Data))
	for i, usage := range result.Data {
		usageResponses[i] = &dto.TopicUsageResponse{
			ID:        usage.ID,
			SiteID:    usage.SiteID,
			TopicID:   usage.TopicID,
			ArticleID: usage.ArticleID,
			Strategy:  usage.Strategy,
			UsedAt:    usage.UsedAt,
			CreatedAt: usage.CreatedAt,
		}
	}

	totalPages := (result.Total + pagination.Limit - 1) / pagination.Limit

	response := &dto.TopicUsageListResponse{
		UsageHistory: usageResponses,
		Pagination: &dto.PaginationResponse{
			Page:       pagination.Page,
			Limit:      pagination.Limit,
			Total:      int64(result.Total),
			TotalPages: totalPages,
		},
	}

	return response, nil
}

// CheckStrategyAvailability checks if more topics are available for a strategy
func (h *Handler) CheckStrategyAvailability(siteID int64, strategy string) (*dto.StrategyAvailabilityResponse, error) {
	ctx := h.fastCtx()

	if siteID <= 0 {
		return nil, fmt.Errorf("invalid site ID")
	}
	if strategy == "" {
		return nil, fmt.Errorf("strategy is required")
	}

	canContinue, err := h.topicStrategyService.CanContinueWithStrategy(ctx, siteID, strategy)
	if err != nil {
		return nil, fmt.Errorf("failed to check strategy availability: %w", err)
	}

	// Get stats to provide more information
	stats, err := h.repo.GetTopicStats(ctx, siteID)
	if err != nil {
		return nil, fmt.Errorf("failed to get topic stats: %w", err)
	}

	response := &dto.StrategyAvailabilityResponse{
		SiteID:         siteID,
		Strategy:       strategy,
		CanContinue:    canContinue,
		TotalTopics:    stats.TotalTopics,
		ActiveTopics:   stats.ActiveTopics,
		UnusedTopics:   stats.UnusedTopics,
		RemainingCount: stats.UniqueTopicsLeft, // For unique strategy
	}

	return response, nil
}
