package dto

import (
	"github.com/davidmovas/postulator/internal/domain/entities"
)

type HealthCheckHistory struct {
	ID             int64  `json:"id"`
	SiteID         int64  `json:"siteId"`
	CheckedAt      string `json:"checkedAt"`
	Status         string `json:"status"`
	ResponseTimeMs int    `json:"responseTimeMs"`
	StatusCode     int    `json:"statusCode"`
	ErrorMessage   string `json:"errorMessage"`
}

type AutoCheckResult struct {
	Unhealthy []*Site `json:"unhealthy"`
	Recovered []*Site `json:"recovered"`
}

func NewHealthHistory(h *entities.HealthCheckHistory) *HealthCheckHistory {
	return (&HealthCheckHistory{}).FromEntity(h)
}

func (d *HealthCheckHistory) FromEntity(e *entities.HealthCheckHistory) *HealthCheckHistory {
	if e == nil {
		return d
	}
	d.ID = e.ID
	d.SiteID = e.SiteID
	d.CheckedAt = TimeToString(e.CheckedAt)
	d.Status = string(e.Status)
	d.ResponseTimeMs = e.ResponseTimeMs
	d.StatusCode = e.StatusCode
	d.ErrorMessage = e.ErrorMessage
	return d
}

func NewHealthHistoryList(items []*entities.HealthCheckHistory) []*HealthCheckHistory {
	res := make([]*HealthCheckHistory, 0, len(items))
	for _, it := range items {
		res = append(res, NewHealthHistory(it))
	}
	return res
}

func SitesToDTO(items []*entities.Site) []*Site {
	res := make([]*Site, 0, len(items))
	for _, s := range items {
		res = append(res, NewSite(s))
	}
	return res
}
