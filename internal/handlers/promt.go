package handlers

import (
	"Postulator/internal/dto"
	"fmt"
)

// Prompt Handlers

// CreatePrompt creates a new prompt
func (h *Handler) CreatePrompt(req dto.CreatePromptRequest) (*dto.PromptResponse, error) {
	ctx := h.fastCtx()

	// Validate request
	if err := dto.ValidateRequired(req.Name, "name"); err != nil {
		return nil, err
	}
	if err := dto.ValidateRequired(req.System, "system"); err != nil {
		return nil, err
	}
	if err := dto.ValidateRequired(req.User, "user"); err != nil {
		return nil, err
	}

	// Convert to model and create
	prompt := req.ToModel()
	createdPrompt, err := h.repo.CreatePrompt(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to create prompt: %w", err)
	}

	return dto.PromptToResponse(createdPrompt), nil
}

// GetPrompt retrieves a single prompt by ID
func (h *Handler) GetPrompt(promptID int64) (*dto.PromptResponse, error) {
	ctx := h.fastCtx()

	if promptID <= 0 {
		return nil, fmt.Errorf("invalid prompt ID")
	}

	prompt, err := h.repo.GetPrompt(ctx, promptID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve prompt: %w", err)
	}

	return dto.PromptToResponse(prompt), nil
}

// GetPrompts retrieves all prompts with pagination
func (h *Handler) GetPrompts(pagination dto.PaginationRequest) (*dto.PromptListResponse, error) {
	ctx := h.fastCtx()

	// Set defaults
	if pagination.Page <= 0 {
		pagination.Page = 1
	}
	if pagination.Limit <= 0 {
		pagination.Limit = 10
	}

	offset := (pagination.Page - 1) * pagination.Limit

	result, err := h.repo.GetPrompts(ctx, pagination.Limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve prompts: %w", err)
	}

	// Convert to response DTOs
	promptResponses := make([]*dto.PromptResponse, len(result.Data))
	for i, prompt := range result.Data {
		promptResponses[i] = dto.PromptToResponse(prompt)
	}

	totalPages := (result.Total + pagination.Limit - 1) / pagination.Limit

	response := &dto.PromptListResponse{
		Prompts: promptResponses,
		Pagination: &dto.PaginationResponse{
			Page:       pagination.Page,
			Limit:      pagination.Limit,
			Total:      int64(result.Total),
			TotalPages: totalPages,
		},
	}

	return response, nil
}

// UpdatePrompt updates an existing prompt
func (h *Handler) UpdatePrompt(req dto.UpdatePromptRequest) (*dto.PromptResponse, error) {
	ctx := h.fastCtx()

	// Validate request
	if req.ID <= 0 {
		return nil, fmt.Errorf("invalid prompt ID")
	}
	if err := dto.ValidateRequired(req.Name, "name"); err != nil {
		return nil, err
	}
	if err := dto.ValidateRequired(req.Content, "content"); err != nil {
		return nil, err
	}

	// Convert to model and update
	prompt := req.ToModel()
	updatedPrompt, err := h.repo.UpdatePrompt(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to update prompt: %w", err)
	}

	return dto.PromptToResponse(updatedPrompt), nil
}

// DeletePrompt deletes a prompt by ID
func (h *Handler) DeletePrompt(promptID int64) error {
	ctx := h.fastCtx()

	if promptID <= 0 {
		return fmt.Errorf("invalid prompt ID")
	}

	err := h.repo.DeletePrompt(ctx, promptID)
	if err != nil {
		return fmt.Errorf("failed to delete prompt: %w", err)
	}

	return nil
}

// GetDefaultPrompt retrieves the default prompt
func (h *Handler) GetDefaultPrompt() (*dto.PromptResponse, error) {
	ctx := h.fastCtx()

	prompt, err := h.repo.GetDefaultPrompt(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve default prompt: %w", err)
	}

	return dto.PromptToResponse(prompt), nil
}

// SetDefaultPrompt sets a prompt as the default
func (h *Handler) SetDefaultPrompt(req dto.SetDefaultPromptRequest) error {
	ctx := h.fastCtx()

	if req.ID <= 0 {
		return fmt.Errorf("invalid prompt ID")
	}

	err := h.repo.SetDefaultPrompt(ctx, req.ID)
	if err != nil {
		return fmt.Errorf("failed to set default prompt: %w", err)
	}

	return nil
}

// SitePrompt Handlers

// CreateSitePrompt creates a new site-prompt relationship
func (h *Handler) CreateSitePrompt(req dto.CreateSitePromptRequest) (*dto.SitePromptResponse, error) {
	ctx := h.fastCtx()

	// Validate request
	if req.SiteID <= 0 {
		return nil, fmt.Errorf("invalid site ID")
	}
	if req.PromptID <= 0 {
		return nil, fmt.Errorf("invalid prompt ID")
	}

	// Convert to model and create
	sitePrompt := req.ToModel()
	createdSitePrompt, err := h.repo.CreateSitePrompt(ctx, sitePrompt)
	if err != nil {
		return nil, fmt.Errorf("failed to create site prompt: %w", err)
	}

	return dto.SitePromptToResponse(createdSitePrompt), nil
}

// GetSitePrompt retrieves the prompt for a specific site
func (h *Handler) GetSitePrompt(siteID int64) (*dto.SitePromptResponse, error) {
	ctx := h.fastCtx()

	if siteID <= 0 {
		return nil, fmt.Errorf("invalid site ID")
	}

	sitePrompt, err := h.repo.GetSitePrompt(ctx, siteID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve site prompt: %w", err)
	}

	return dto.SitePromptToResponse(sitePrompt), nil
}

// UpdateSitePrompt updates an existing site-prompt relationship
func (h *Handler) UpdateSitePrompt(req dto.UpdateSitePromptRequest) (*dto.SitePromptResponse, error) {
	ctx := h.fastCtx()

	// Validate request
	if req.ID <= 0 {
		return nil, fmt.Errorf("invalid site prompt ID")
	}
	if req.PromptID <= 0 {
		return nil, fmt.Errorf("invalid prompt ID")
	}

	// Convert to model and update
	sitePrompt := req.ToModel()
	updatedSitePrompt, err := h.repo.UpdateSitePrompt(ctx, sitePrompt)
	if err != nil {
		return nil, fmt.Errorf("failed to update site prompt: %w", err)
	}

	return dto.SitePromptToResponse(updatedSitePrompt), nil
}

// DeleteSitePrompt deletes a site-prompt relationship by ID
func (h *Handler) DeleteSitePrompt(sitePromptID int64) error {
	ctx := h.fastCtx()

	if sitePromptID <= 0 {
		return fmt.Errorf("invalid site prompt ID")
	}

	err := h.repo.DeleteSitePrompt(ctx, sitePromptID)
	if err != nil {
		return fmt.Errorf("failed to delete site prompt: %w", err)
	}

	return nil
}

// DeleteSitePromptBySite deletes all site-prompt relationships for a specific site
func (h *Handler) DeleteSitePromptBySite(siteID int64) error {
	ctx := h.fastCtx()

	if siteID <= 0 {
		return fmt.Errorf("invalid site ID")
	}

	err := h.repo.DeleteSitePromptBySite(ctx, siteID)
	if err != nil {
		return fmt.Errorf("failed to delete site prompts by site: %w", err)
	}

	return nil
}

// GetPromptSites retrieves all sites associated with a specific prompt
func (h *Handler) GetPromptSites(promptID int64, pagination dto.PaginationRequest) (*dto.SitePromptListResponse, error) {
	ctx := h.fastCtx()

	if promptID <= 0 {
		return nil, fmt.Errorf("invalid prompt ID")
	}

	// Set defaults
	if pagination.Page <= 0 {
		pagination.Page = 1
	}
	if pagination.Limit <= 0 {
		pagination.Limit = 10
	}

	offset := (pagination.Page - 1) * pagination.Limit

	result, err := h.repo.GetPromptSites(ctx, promptID, pagination.Limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve prompt sites: %w", err)
	}

	// Convert to response DTOs
	sitePromptResponses := make([]*dto.SitePromptResponse, len(result.Data))
	for i, sitePrompt := range result.Data {
		sitePromptResponses[i] = dto.SitePromptToResponse(sitePrompt)
	}

	totalPages := (result.Total + pagination.Limit - 1) / pagination.Limit

	response := &dto.SitePromptListResponse{
		SitePrompts: sitePromptResponses,
		Pagination: &dto.PaginationResponse{
			Page:       pagination.Page,
			Limit:      pagination.Limit,
			Total:      int64(result.Total),
			TotalPages: totalPages,
		},
	}

	return response, nil
}
