package bindings

import (
	"Postulator/internal/dto"
)

// CreatePrompt creates a new prompt
func (b *Binder) CreatePrompt(req dto.CreatePromptRequest) (*dto.PromptResponse, error) {
	return b.handler.CreatePrompt(req)
}

// GetPrompt retrieves a single prompt by ID
func (b *Binder) GetPrompt(promptID int64) (*dto.PromptResponse, error) {
	return b.handler.GetPrompt(promptID)
}

// GetPrompts retrieves all prompts with pagination
func (b *Binder) GetPrompts(pagination dto.PaginationRequest) (*dto.PromptListResponse, error) {
	return b.handler.GetPrompts(pagination)
}

// UpdatePrompt updates an existing prompt
func (b *Binder) UpdatePrompt(req dto.UpdatePromptRequest) (*dto.PromptResponse, error) {
	return b.handler.UpdatePrompt(req)
}

// DeletePrompt deletes a prompt by ID
func (b *Binder) DeletePrompt(promptID int64) error {
	return b.handler.DeletePrompt(promptID)
}

// GetDefaultPrompt retrieves the default prompt
func (b *Binder) GetDefaultPrompt() (*dto.PromptResponse, error) {
	return b.handler.GetDefaultPrompt()
}

// SetDefaultPrompt sets a prompt as the default
func (b *Binder) SetDefaultPrompt(req dto.SetDefaultPromptRequest) error {
	return b.handler.SetDefaultPrompt(req)
}

// SitePrompt operations

// CreateSitePrompt creates a new site-prompt relationship
func (b *Binder) CreateSitePrompt(req dto.CreateSitePromptRequest) (*dto.SitePromptResponse, error) {
	return b.handler.CreateSitePrompt(req)
}

// GetSitePrompt retrieves the prompt for a specific site
func (b *Binder) GetSitePrompt(siteID int64) (*dto.SitePromptResponse, error) {
	return b.handler.GetSitePrompt(siteID)
}

// UpdateSitePrompt updates an existing site-prompt relationship
func (b *Binder) UpdateSitePrompt(req dto.UpdateSitePromptRequest) (*dto.SitePromptResponse, error) {
	return b.handler.UpdateSitePrompt(req)
}

// DeleteSitePrompt deletes a site-prompt relationship by ID
func (b *Binder) DeleteSitePrompt(sitePromptID int64) error {
	return b.handler.DeleteSitePrompt(sitePromptID)
}

// DeleteSitePromptBySite deletes all site-prompt relationships for a specific site
func (b *Binder) DeleteSitePromptBySite(siteID int64) error {
	return b.handler.DeleteSitePromptBySite(siteID)
}

// ActivateSitePrompt activates a site-prompt relationship
func (b *Binder) ActivateSitePrompt(sitePromptID int64) error {
	return b.handler.ActivateSitePrompt(sitePromptID)
}

// DeactivateSitePrompt deactivates a site-prompt relationship
func (b *Binder) DeactivateSitePrompt(sitePromptID int64) error {
	return b.handler.DeactivateSitePrompt(sitePromptID)
}

// GetPromptSites retrieves all sites associated with a specific prompt
func (b *Binder) GetPromptSites(promptID int64, pagination dto.PaginationRequest) (*dto.SitePromptListResponse, error) {
	return b.handler.GetPromptSites(promptID, pagination)
}
