package app

import (
	"Postulator/internal/domain/entities"
	"Postulator/internal/dto"
	"Postulator/pkg/errors"
	"context"
)

// CreateSite creates a new site using provided DTO fields
func (a *App) CreateSite(site *dto.Site) *dto.Response[string] {
	if site == nil {
		return dtoErr[string](errors.Validation("site payload is required"))
	}
	e := &entities.Site{
		Name:       site.Name,
		URL:        site.URL,
		WPUsername: site.WPUsername,
		WPPassword: "", // password handled via SetSitePassword for secure storage
		Status:     entities.Status(site.Status),
	}
	if err := a.siteSvc.CreateSite(context.Background(), e); err != nil {
		return dtoErr[string](asAppErr(err))
	}
	return &dto.Response[string]{Success: true, Data: "created"}
}

// SetSitePassword securely sets/updates the WordPress password for the site
func (a *App) SetSitePassword(siteID int64, password string) *dto.Response[string] {
	if siteID == 0 {
		return dtoErr[string](errors.Validation("invalid site id"))
	}
	if password == "" {
		return dtoErr[string](errors.Validation("password cannot be empty"))
	}

	if err := a.siteSvc.UpdateSitePassword(context.Background(), siteID, password); err != nil {
		return dtoErr[string](asAppErr(err))
	}

	return &dto.Response[string]{Success: true, Data: "password updated"}
}

// GetSite returns site by ID
func (a *App) GetSite(id int64) *dto.Response[*dto.Site] {
	s, err := a.siteSvc.GetSite(context.Background(), id)
	if err != nil {
		return dtoErr[*dto.Site](asAppErr(err))
	}
	return &dto.Response[*dto.Site]{Success: true, Data: dto.FromSite(s)}
}

// ListSites lists all sites
func (a *App) ListSites() *dto.Response[[]*dto.Site] {
	if a.siteSvc == nil {
		panic("site service is nil")
	}

	sites, err := a.siteSvc.ListSites(context.Background())
	if err != nil {
		return dtoErr[[]*dto.Site](asAppErr(err))
	}
	return &dto.Response[[]*dto.Site]{Success: true, Data: dto.FromSites(sites)}
}

// UpdateSite updates a site
func (a *App) UpdateSite(site *dto.Site) *dto.Response[string] {
	if site == nil {
		return dtoErr[string](errors.Validation("site payload is required"))
	}
	e := &entities.Site{
		ID:         site.ID,
		Name:       site.Name,
		URL:        site.URL,
		WPUsername: site.WPUsername,
		Status:     entities.Status(site.Status),
	}
	if err := a.siteSvc.UpdateSite(context.Background(), e); err != nil {
		return dtoErr[string](asAppErr(err))
	}
	return &dto.Response[string]{Success: true, Data: "updated"}
}

// DeleteSite removes a site by ID
func (a *App) DeleteSite(id int64) *dto.Response[string] {
	if err := a.siteSvc.DeleteSite(context.Background(), id); err != nil {
		return dtoErr[string](asAppErr(err))
	}
	return &dto.Response[string]{Success: true, Data: "deleted"}
}

// CheckHealth performs a WordPress health check and updates status
func (a *App) CheckHealth(siteID int64) *dto.Response[string] {
	if err := a.siteSvc.CheckHealth(context.Background(), siteID); err != nil {
		return dtoErr[string](asAppErr(err))
	}
	return &dto.Response[string]{Success: true, Data: "checked"}
}

// SyncCategories fetches categories from WP and stores them
func (a *App) SyncCategories(siteID int64) *dto.Response[string] {
	if err := a.siteSvc.SyncCategories(context.Background(), siteID); err != nil {
		return dtoErr[string](asAppErr(err))
	}
	return &dto.Response[string]{Success: true, Data: "synced"}
}

// GetSiteCategories returns categories of a site
func (a *App) GetSiteCategories(siteID int64) *dto.Response[[]*dto.Category] {
	cats, err := a.siteSvc.GetSiteCategories(context.Background(), siteID)
	if err != nil {
		return dtoErr[[]*dto.Category](asAppErr(err))
	}
	return &dto.Response[[]*dto.Category]{Success: true, Data: dto.FromCategories(cats)}
}
