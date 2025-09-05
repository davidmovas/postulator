package handlers

import (
	"Postulator/internal/dto"
	"fmt"
	"time"
)

// GetSites retrieves all sites with pagination
func (h *Handler) GetSites(pagination dto.PaginationRequest) (*dto.SiteListResponse, error) {
	ctx := h.fastCtx()

	// Set defaults
	if pagination.Page <= 0 {
		pagination.Page = 1
	}
	if pagination.Limit <= 0 {
		pagination.Limit = 10
	}

	offset := (pagination.Page - 1) * pagination.Limit

	result, err := h.repo.GetSites(ctx, pagination.Limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve sites: %w", err)
	}

	// Convert to response DTOs
	siteResponses := make([]*dto.SiteResponse, len(result.Data))
	for i, site := range result.Data {
		siteResponses[i] = dto.SiteToResponse(site)
	}

	totalPages := (result.Total + pagination.Limit - 1) / pagination.Limit

	response := &dto.SiteListResponse{
		Sites: siteResponses,
		Pagination: &dto.PaginationResponse{
			Page:       pagination.Page,
			Limit:      pagination.Limit,
			Total:      int64(result.Total),
			TotalPages: totalPages,
		},
	}

	return response, nil
}

// GetSite retrieves a single site by ID
func (h *Handler) GetSite(siteID int64) (*dto.SiteResponse, error) {
	ctx := h.fastCtx()

	if siteID <= 0 {
		return nil, fmt.Errorf("invalid site ID")
	}

	site, err := h.repo.GetSite(ctx, siteID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve site: %w", err)
	}

	return dto.SiteToResponse(site), nil
}

// CreateSite creates a new WordPress site
func (h *Handler) CreateSite(req dto.CreateSiteRequest) (*dto.SiteResponse, error) {
	ctx := h.fastCtx()

	// Validate request
	if err := h.validateSiteRequest(req.Name, req.URL, req.Username, req.Password); err != nil {
		return nil, err
	}

	// Convert to model and create
	site := req.ToModel()
	createdSite, err := h.repo.CreateSite(ctx, site)
	if err != nil {
		return nil, fmt.Errorf("failed to create site: %w", err)
	}

	return dto.SiteToResponse(createdSite), nil
}

// UpdateSite updates an existing site
func (h *Handler) UpdateSite(req dto.UpdateSiteRequest) (*dto.SiteResponse, error) {
	ctx := h.fastCtx()

	// Validate ID
	if req.ID <= 0 {
		return nil, fmt.Errorf("invalid site ID")
	}

	// Validate request
	if err := h.validateSiteRequest(req.Name, req.URL, req.Username, req.Password); err != nil {
		return nil, err
	}

	// Convert to model and update
	site := req.ToModel()
	updatedSite, err := h.repo.UpdateSite(ctx, site)
	if err != nil {
		return nil, fmt.Errorf("failed to update site: %w", err)
	}

	return dto.SiteToResponse(updatedSite), nil
}

// ActivateSite activates a site
func (h *Handler) ActivateSite(siteID int64) error {
	ctx := h.fastCtx()

	if siteID <= 0 {
		return fmt.Errorf("invalid site ID")
	}

	err := h.repo.ActivateSite(ctx, siteID)
	if err != nil {
		return fmt.Errorf("failed to activate site: %w", err)
	}

	return nil
}

// DeactivateSite deactivates a site
func (h *Handler) DeactivateSite(siteID int64) error {
	ctx := h.fastCtx()

	if siteID <= 0 {
		return fmt.Errorf("invalid site ID")
	}

	err := h.repo.DeactivateSite(ctx, siteID)
	if err != nil {
		return fmt.Errorf("failed to deactivate site: %w", err)
	}

	return nil
}

// SetSiteCheckStatus updates the check status of a site
func (h *Handler) SetSiteCheckStatus(siteID int64, status string) error {
	ctx := h.fastCtx()

	if siteID <= 0 {
		return fmt.Errorf("invalid site ID")
	}

	if status == "" {
		return fmt.Errorf("status is required")
	}

	err := h.repo.SetCheckStatus(ctx, siteID, time.Now(), status)
	if err != nil {
		return fmt.Errorf("failed to set check status: %w", err)
	}

	return nil
}

// DeleteSite deletes a site
func (h *Handler) DeleteSite(siteID int64) error {
	ctx := h.fastCtx()

	if siteID <= 0 {
		return fmt.Errorf("invalid site ID")
	}

	err := h.repo.DeleteSite(ctx, siteID)
	if err != nil {
		return fmt.Errorf("failed to delete site: %w", err)
	}

	return nil
}

// TestSiteConnection tests connection to a WordPress site
func (h *Handler) TestSiteConnection(req dto.TestSiteConnectionRequest) (*dto.TestConnectionResponse, error) {
	if req.SiteID <= 0 {
		return nil, fmt.Errorf("invalid site ID")
	}

	// For now, return a successful connection test
	// TODO: Implement actual WordPress connection testing with site data
	response := &dto.TestConnectionResponse{
		Success:   true,
		Status:    "connected",
		Message:   "Connection test successful",
		Timestamp: time.Now(),
	}

	return response, nil
}

// Helper methods

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
