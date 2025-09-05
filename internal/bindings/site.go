package bindings

import (
	"Postulator/internal/dto"
)

// GetSites retrieves all sites with pagination
func (b *Binder) GetSites(pagination dto.PaginationRequest) (*dto.SiteListResponse, error) {
	return b.handler.GetSites(pagination)
}

// GetSite retrieves a single site by ID
func (b *Binder) GetSite(siteID int64) (*dto.SiteResponse, error) {
	return b.handler.GetSite(siteID)
}

// CreateSite creates a new WordPress site
func (b *Binder) CreateSite(req dto.CreateSiteRequest) (*dto.SiteResponse, error) {
	return b.handler.CreateSite(req)
}

// UpdateSite updates an existing site
func (b *Binder) UpdateSite(req dto.UpdateSiteRequest) (*dto.SiteResponse, error) {
	return b.handler.UpdateSite(req)
}

// ActivateSite activates a site
func (b *Binder) ActivateSite(siteID int64) error {
	return b.handler.ActivateSite(siteID)
}

// DeactivateSite deactivates a site
func (b *Binder) DeactivateSite(siteID int64) error {
	return b.handler.DeactivateSite(siteID)
}

// SetSiteCheckStatus updates the last check status of a site
func (b *Binder) SetSiteCheckStatus(siteID int64, status string) error {
	return b.handler.SetSiteCheckStatus(siteID, status)
}

// DeleteSite deletes a site by ID
func (b *Binder) DeleteSite(siteID int64) error {
	return b.handler.DeleteSite(siteID)
}

// TestSiteConnection tests the connection to a WordPress site
func (b *Binder) TestSiteConnection(req dto.TestSiteConnectionRequest) (*dto.TestConnectionResponse, error) {
	return b.handler.TestSiteConnection(req)
}
