package handlers

import (
	"Postulator/internal/dto"
	"Postulator/internal/models"
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strings"
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
	siteTopicModel := &models.SiteTopic{
		SiteID:   req.SiteID,
		TopicID:  req.TopicID,
		Priority: req.Priority,
	}

	createdSiteTopic, err := h.repo.CreateSiteTopic(ctx, siteTopicModel)
	if err != nil {
		return nil, fmt.Errorf("failed to create site topic: %w", err)
	}

	return &dto.SiteTopicResponse{
		ID:            createdSiteTopic.ID,
		SiteID:        createdSiteTopic.SiteID,
		TopicID:       createdSiteTopic.TopicID,
		Priority:      createdSiteTopic.Priority,
		UsageCount:    createdSiteTopic.UsageCount,
		LastUsedAt:    createdSiteTopic.LastUsedAt,
		RoundRobinPos: createdSiteTopic.RoundRobinPos,
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
			ID:            siteTopic.ID,
			SiteID:        siteTopic.SiteID,
			TopicID:       siteTopic.TopicID,
			Priority:      siteTopic.Priority,
			UsageCount:    siteTopic.UsageCount,
			LastUsedAt:    siteTopic.LastUsedAt,
			RoundRobinPos: siteTopic.RoundRobinPos,
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
			ID:            siteTopic.ID,
			SiteID:        siteTopic.SiteID,
			TopicID:       siteTopic.TopicID,
			Priority:      siteTopic.Priority,
			UsageCount:    siteTopic.UsageCount,
			LastUsedAt:    siteTopic.LastUsedAt,
			RoundRobinPos: siteTopic.RoundRobinPos,
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
		Priority: req.Priority,
	}

	updatedSiteTopic, err := h.repo.UpdateSiteTopic(ctx, siteTopicModel)
	if err != nil {
		return nil, fmt.Errorf("failed to update site topic: %w", err)
	}

	return &dto.SiteTopicResponse{
		ID:            updatedSiteTopic.ID,
		SiteID:        updatedSiteTopic.SiteID,
		TopicID:       updatedSiteTopic.TopicID,
		Priority:      updatedSiteTopic.Priority,
		UsageCount:    updatedSiteTopic.UsageCount,
		LastUsedAt:    updatedSiteTopic.LastUsedAt,
		RoundRobinPos: updatedSiteTopic.RoundRobinPos,
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
			ID:            result.SiteTopic.ID,
			SiteID:        result.SiteTopic.SiteID,
			TopicID:       result.SiteTopic.TopicID,
			Priority:      result.SiteTopic.Priority,
			UsageCount:    result.SiteTopic.UsageCount,
			LastUsedAt:    result.SiteTopic.LastUsedAt,
			RoundRobinPos: result.SiteTopic.RoundRobinPos,
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

// TopicsImport imports topics from file content with support for txt, csv, jsonl formats
func (h *Handler) TopicsImport(siteID int64, req dto.TopicsImportRequest) (interface{}, error) {
	ctx := h.fastCtx()

	if siteID <= 0 {
		return nil, fmt.Errorf("invalid site ID")
	}

	if req.FileContent == "" {
		return nil, fmt.Errorf("file content is required")
	}

	if req.FileFormat == "" {
		return nil, fmt.Errorf("file format is required")
	}

	// Parse topics from file content
	topics, parseErrors := h.parseTopicsFromContent(req.FileContent, req.FileFormat)

	if req.PreviewOnly {
		return h.generateImportPreview(ctx, siteID, topics, parseErrors)
	}

	return h.executeImport(ctx, siteID, topics, parseErrors)
}

// TopicsReassign reassigns topics from one site to another
func (h *Handler) TopicsReassign(req dto.TopicsReassignRequest) (*dto.ReassignResult, error) {
	ctx := h.fastCtx()

	if req.FromSiteID <= 0 {
		return nil, fmt.Errorf("invalid from site ID")
	}

	if req.ToSiteID <= 0 {
		return nil, fmt.Errorf("invalid to site ID")
	}

	if req.FromSiteID == req.ToSiteID {
		return nil, fmt.Errorf("from and to site IDs cannot be the same")
	}

	// Validate that both sites exist
	_, err := h.repo.GetSite(ctx, req.FromSiteID)
	if err != nil {
		return nil, fmt.Errorf("from site not found: %w", err)
	}

	_, err = h.repo.GetSite(ctx, req.ToSiteID)
	if err != nil {
		return nil, fmt.Errorf("to site not found: %w", err)
	}

	// Get topics to reassign for counting
	var totalTopics int
	if len(req.TopicIDs) > 0 {
		totalTopics = len(req.TopicIDs)
	} else {
		// Count all topics for the site
		result, err := h.repo.GetSiteTopics(ctx, req.FromSiteID, 1000, 0) // Large limit to get all
		if err != nil {
			return nil, fmt.Errorf("failed to get site topics count: %w", err)
		}
		totalTopics = result.Total
	}

	// Perform reassignment
	err = h.repo.ReassignTopicsToSite(ctx, req.FromSiteID, req.ToSiteID, req.TopicIDs)
	if err != nil {
		return &dto.ReassignResult{
			FromSiteID:       req.FromSiteID,
			ToSiteID:         req.ToSiteID,
			ProcessedTopics:  totalTopics,
			ReassignedTopics: 0,
			SkippedTopics:    totalTopics,
			ErrorCount:       1,
			ErrorMessages:    []string{err.Error()},
		}, nil
	}

	return &dto.ReassignResult{
		FromSiteID:       req.FromSiteID,
		ToSiteID:         req.ToSiteID,
		ProcessedTopics:  totalTopics,
		ReassignedTopics: totalTopics,
		SkippedTopics:    0,
		ErrorCount:       0,
		ErrorMessages:    []string{},
	}, nil
}

// parseTopicsFromContent parses topics from different file formats
func (h *Handler) parseTopicsFromContent(content, format string) ([]dto.ImportTopicItem, []string) {
	var topics []dto.ImportTopicItem
	var errors []string

	content = strings.TrimSpace(content)
	if content == "" {
		errors = append(errors, "file content is empty")
		return topics, errors
	}

	switch strings.ToLower(format) {
	case "txt":
		topics, errors = h.parseTxtContent(content)
	case "csv":
		topics, errors = h.parseCsvContent(content)
	case "jsonl":
		topics, errors = h.parseJsonlContent(content)
	default:
		errors = append(errors, fmt.Sprintf("unsupported file format: %s", format))
	}

	return topics, errors
}

// parseTxtContent parses plain text format (one title per line)
func (h *Handler) parseTxtContent(content string) ([]dto.ImportTopicItem, []string) {
	var topics []dto.ImportTopicItem
	var errors []string

	lines := strings.Split(content, "\n")
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if len(line) > 255 {
			errors = append(errors, fmt.Sprintf("line %d: title too long (max 255 characters)", i+1))
			continue
		}

		topics = append(topics, dto.ImportTopicItem{
			Title:  line,
			Status: "new",
		})
	}

	return topics, errors
}

// parseCsvContent parses CSV format (title, keywords, category, tags)
func (h *Handler) parseCsvContent(content string) ([]dto.ImportTopicItem, []string) {
	var topics []dto.ImportTopicItem
	var errors []string

	reader := csv.NewReader(strings.NewReader(content))
	reader.TrimLeadingSpace = true

	lineNum := 0
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			errors = append(errors, fmt.Sprintf("CSV parsing error at line %d: %s", lineNum+1, err.Error()))
			lineNum++
			continue
		}

		lineNum++

		if len(record) == 0 || (len(record) == 1 && strings.TrimSpace(record[0]) == "") {
			continue
		}

		if len(record) < 1 {
			errors = append(errors, fmt.Sprintf("line %d: missing title", lineNum))
			continue
		}

		title := strings.TrimSpace(record[0])
		if title == "" {
			errors = append(errors, fmt.Sprintf("line %d: title cannot be empty", lineNum))
			continue
		}

		if len(title) > 255 {
			errors = append(errors, fmt.Sprintf("line %d: title too long (max 255 characters)", lineNum))
			continue
		}

		topic := dto.ImportTopicItem{
			Title:  title,
			Status: "new",
		}

		if len(record) > 1 {
			topic.Keywords = strings.TrimSpace(record[1])
		}
		if len(record) > 2 {
			topic.Category = strings.TrimSpace(record[2])
		}
		if len(record) > 3 {
			topic.Tags = strings.TrimSpace(record[3])
		}

		topics = append(topics, topic)
	}

	return topics, errors
}

// parseJsonlContent parses JSONL format (one JSON object per line)
func (h *Handler) parseJsonlContent(content string) ([]dto.ImportTopicItem, []string) {
	var topics []dto.ImportTopicItem
	var errors []string

	lines := strings.Split(content, "\n")
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var jsonTopic struct {
			Title    string `json:"title"`
			Keywords string `json:"keywords,omitempty"`
			Category string `json:"category,omitempty"`
			Tags     string `json:"tags,omitempty"`
		}

		err := json.Unmarshal([]byte(line), &jsonTopic)
		if err != nil {
			errors = append(errors, fmt.Sprintf("line %d: invalid JSON: %s", i+1, err.Error()))
			continue
		}

		if jsonTopic.Title == "" {
			errors = append(errors, fmt.Sprintf("line %d: title cannot be empty", i+1))
			continue
		}

		if len(jsonTopic.Title) > 255 {
			errors = append(errors, fmt.Sprintf("line %d: title too long (max 255 characters)", i+1))
			continue
		}

		topics = append(topics, dto.ImportTopicItem{
			Title:    jsonTopic.Title,
			Keywords: jsonTopic.Keywords,
			Category: jsonTopic.Category,
			Tags:     jsonTopic.Tags,
			Status:   "new",
		})
	}

	return topics, errors
}

// generateImportPreview generates preview of import without creating topics
func (h *Handler) generateImportPreview(ctx context.Context, siteID int64, topics []dto.ImportTopicItem, parseErrors []string) (*dto.TopicsImportPreview, error) {
	preview := &dto.TopicsImportPreview{
		SiteID:        siteID,
		TotalLines:    len(topics) + len(parseErrors),
		ValidTopics:   0,
		Duplicates:    0,
		Errors:        len(parseErrors),
		Topics:        make([]dto.ImportTopicItem, 0),
		ErrorMessages: parseErrors,
	}

	// Check for duplicates within file and against database
	titleMap := make(map[string]bool)
	for _, topic := range topics {
		// Check for duplicates within the file
		if titleMap[topic.Title] {
			topic.Status = "duplicate"
			topic.Error = "duplicate within file"
		} else {
			titleMap[topic.Title] = true

			// Check against database
			existingTopic, err := h.repo.GetTopicByTitle(ctx, topic.Title)
			if err != nil && err != sql.ErrNoRows {
				topic.Status = "error"
				topic.Error = fmt.Sprintf("database check failed: %s", err.Error())
				preview.Errors++
			} else if existingTopic != nil {
				topic.Status = "exists"
				topic.Error = "topic already exists in database"
			} else {
				topic.Status = "new"
				preview.ValidTopics++
			}
		}

		if topic.Status == "duplicate" {
			preview.Duplicates++
		}

		preview.Topics = append(preview.Topics, topic)
	}

	return preview, nil
}

// executeImport performs the actual import of topics
func (h *Handler) executeImport(ctx context.Context, siteID int64, topics []dto.ImportTopicItem, parseErrors []string) (*dto.TopicsImportResult, error) {
	result := &dto.TopicsImportResult{
		SiteID:         siteID,
		TotalProcessed: len(topics),
		CreatedTopics:  0,
		ReusedTopics:   0,
		SkippedTopics:  0,
		ErrorCount:     len(parseErrors),
		Topics:         make([]dto.ImportTopicItem, 0),
		ErrorMessages:  parseErrors,
	}

	// Check for duplicates and prepare topics for creation
	titleMap := make(map[string]bool)
	var topicsToCreate []*models.Topic
	var topicsToReuse []*models.Topic

	for _, topic := range topics {
		// Skip duplicates within file
		if titleMap[topic.Title] {
			topic.Status = "duplicate"
			topic.Error = "duplicate within file"
			result.SkippedTopics++
		} else {
			titleMap[topic.Title] = true

			// Check if topic exists in database
			existingTopic, err := h.repo.GetTopicByTitle(ctx, topic.Title)
			if err != nil && err != sql.ErrNoRows {
				topic.Status = "error"
				topic.Error = fmt.Sprintf("database check failed: %s", err.Error())
				result.ErrorCount++
			} else if existingTopic != nil {
				// Topic exists, reuse it
				topic.Status = "exists"
				topicsToReuse = append(topicsToReuse, existingTopic)
				result.ReusedTopics++
			} else {
				// New topic to create
				topicsToCreate = append(topicsToCreate, &models.Topic{
					Title:    topic.Title,
					Keywords: topic.Keywords,
					Category: topic.Category,
					Tags:     topic.Tags,
				})
				topic.Status = "new"
			}
		}

		result.Topics = append(result.Topics, topic)
	}

	// Create new topics with site binding
	if len(topicsToCreate) > 0 {
		createdTopics, err := h.repo.BulkCreateTopicsWithSiteBinding(ctx, siteID, topicsToCreate)
		if err != nil {
			result.ErrorMessages = append(result.ErrorMessages, fmt.Sprintf("bulk create failed: %s", err.Error()))
			result.ErrorCount++
			result.SkippedTopics += len(topicsToCreate)
		} else {
			result.CreatedTopics = len(createdTopics)
		}
	}

	// Create site bindings for existing topics (reused ones)
	for _, existingTopic := range topicsToReuse {
		_, err := h.repo.CreateSiteTopic(ctx, &models.SiteTopic{
			SiteID:   siteID,
			TopicID:  existingTopic.ID,
			Priority: 1,
		})
		if err != nil {
			// If binding already exists, that's ok
			if !strings.Contains(err.Error(), "UNIQUE constraint") {
				result.ErrorMessages = append(result.ErrorMessages, fmt.Sprintf("failed to bind topic '%s': %s", existingTopic.Title, err.Error()))
				result.ErrorCount++
			}
		}
	}

	return result, nil
}
