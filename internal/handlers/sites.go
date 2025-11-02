package handlers

import (
	"github.com/davidmovas/postulator/internal/domain/sites"
	"github.com/davidmovas/postulator/internal/dto"
	"github.com/davidmovas/postulator/pkg/ctx"
)

type SitesHandler struct {
	service sites.Service
}

func NewSitesHandler(service sites.Service) *SitesHandler {
	return &SitesHandler{
		service: service,
	}
}

func (h *SitesHandler) CreateSite(site *dto.Site) *dto.Response[string] {
	entity, err := site.ToEntity()
	if err != nil {
		return fail[string](err)
	}

	if err = h.service.CreateSite(ctx.FastCtx(), entity); err != nil {
		return fail[string](err)
	}

	return ok("Site created successfully")
}

func (h *SitesHandler) GetSite(id int64) *dto.Response[*dto.Site] {
	site, err := h.service.GetSite(ctx.FastCtx(), id)
	if err != nil {
		return fail[*dto.Site](err)
	}

	return ok(dto.NewSite(site))
}

func (h *SitesHandler) GetSiteWithPassword(id int64) *dto.Response[*dto.Site] {
	site, err := h.service.GetSiteWithPassword(ctx.FastCtx(), id)
	if err != nil {
		return fail[*dto.Site](err)
	}

	return ok(dto.NewSite(site))
}

func (h *SitesHandler) ListSites() *dto.Response[[]*dto.Site] {
	listSites, err := h.service.ListSites(ctx.FastCtx())
	if err != nil {
		return fail[[]*dto.Site](err)
	}

	var dtoSites []*dto.Site
	for _, site := range listSites {
		dtoSites = append(dtoSites, dto.NewSite(site))
	}

	return ok(dtoSites)
}

func (h *SitesHandler) UpdateSite(site *dto.Site) *dto.Response[string] {
	entity, err := site.ToEntity()
	if err != nil {
		return fail[string](err)
	}

	if err = h.service.UpdateSite(ctx.FastCtx(), entity); err != nil {
		return fail[string](err)
	}

	return ok("Site updated successfully")
}

func (h *SitesHandler) UpdateSitePassword(id int64, password string) *dto.Response[string] {
	if err := h.service.UpdateSitePassword(ctx.FastCtx(), id, password); err != nil {
		return fail[string](err)
	}

	return ok("Site password updated successfully")
}

func (h *SitesHandler) DeleteSite(id int64) *dto.Response[string] {
	if err := h.service.DeleteSite(ctx.FastCtx(), id); err != nil {
		return fail[string](err)
	}

	return ok("Site deleted successfully")
}

func (h *SitesHandler) CheckHealth(siteID int64) *dto.Response[string] {
	if err := h.service.CheckHealth(ctx.FastCtx(), siteID); err != nil {
		return fail[string](err)
	}

	return ok("Site health checked")
}

func (h *SitesHandler) CheckAllHealth() *dto.Response[string] {
	if err := h.service.CheckAllHealth(ctx.FastCtx()); err != nil {
		return fail[string](err)
	}

	return ok("All sites health checked")
}
