package dto

import (
	"github.com/davidmovas/postulator/internal/domain/entities"
)

type Site struct {
	ID              int64   `json:"id"`
	Name            string  `json:"name"`
	URL             string  `json:"url"`
	WPUsername      string  `json:"wpUsername"`
	Status          string  `json:"status"`
	LastHealthCheck *string `json:"lastHealthCheck,omitempty"`
	HealthStatus    string  `json:"healthStatus"`
	CreatedAt       string  `json:"createdAt"`
	UpdatedAt       string  `json:"updatedAt"`
}

type Category struct {
	ID           int64   `json:"id"`
	SiteID       int64   `json:"siteId"`
	WPCategoryID int     `json:"wpCategoryId"`
	Name         string  `json:"name"`
	Slug         *string `json:"slug,omitempty"`
	Count        int     `json:"count"`
	CreatedAt    string  `json:"createdAt"`
}

func FromSite(e *entities.Site) *Site {
	if e == nil {
		return nil
	}
	var last *string
	if e.LastHealthCheck != nil {
		v := e.LastHealthCheck.UTC().Format(timeLayout)
		last = &v
	}
	return &Site{
		ID:              e.ID,
		Name:            e.Name,
		URL:             e.URL,
		WPUsername:      e.WPUsername,
		Status:          string(e.Status),
		LastHealthCheck: last,
		HealthStatus:    string(e.HealthStatus),
		CreatedAt:       e.CreatedAt.UTC().Format(timeLayout),
		UpdatedAt:       e.UpdatedAt.UTC().Format(timeLayout),
	}
}

func FromSites(items []*entities.Site) []*Site {
	out := make([]*Site, 0, len(items))
	for _, it := range items {
		out = append(out, FromSite(it))
	}
	return out
}

func FromCategory(e *entities.Category) *Category {
	if e == nil {
		return nil
	}
	var slug *string
	if e.Slug != nil {
		v := *e.Slug
		slug = &v
	}
	return &Category{
		ID:           e.ID,
		SiteID:       e.SiteID,
		WPCategoryID: e.WPCategoryID,
		Name:         e.Name,
		Slug:         slug,
		Count:        e.Count,
		CreatedAt:    e.CreatedAt.UTC().Format(timeLayout),
	}
}

func FromCategories(items []*entities.Category) []*Category {
	out := make([]*Category, 0, len(items))
	for _, it := range items {
		out = append(out, FromCategory(it))
	}
	return out
}
