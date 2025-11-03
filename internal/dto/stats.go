package dto

import "github.com/davidmovas/postulator/internal/domain/entities"

type SiteStats struct {
	ID                   int64  `json:"id"`
	SiteID               int64  `json:"siteId"`
	Date                 string `json:"date"`
	ArticlesPublished    int    `json:"articlesPublished"`
	ArticlesFailed       int    `json:"articlesFailed"`
	TotalWords           int    `json:"totalWords"`
	InternalLinksCreated int    `json:"internalLinksCreated"`
	ExternalLinksCreated int    `json:"externalLinksCreated"`
}

func NewSiteStats(entity *entities.SiteStats) *SiteStats {
	s := &SiteStats{}
	return s.FromEntity(entity)
}

func (d *SiteStats) ToEntity() (*entities.SiteStats, error) {
	date, err := StringToTime(d.Date)
	if err != nil {
		return nil, err
	}

	return &entities.SiteStats{
		ID:                   d.ID,
		SiteID:               d.SiteID,
		Date:                 date,
		ArticlesPublished:    d.ArticlesPublished,
		ArticlesFailed:       d.ArticlesFailed,
		TotalWords:           d.TotalWords,
		InternalLinksCreated: d.InternalLinksCreated,
		ExternalLinksCreated: d.ExternalLinksCreated,
	}, nil
}

func (d *SiteStats) FromEntity(entity *entities.SiteStats) *SiteStats {
	d.ID = entity.ID
	d.SiteID = entity.SiteID
	d.Date = TimeToString(entity.Date)
	d.ArticlesPublished = entity.ArticlesPublished
	d.ArticlesFailed = entity.ArticlesFailed
	d.TotalWords = entity.TotalWords
	d.InternalLinksCreated = entity.InternalLinksCreated
	d.ExternalLinksCreated = entity.ExternalLinksCreated
	return d
}

type DashboardSummary struct {
	TotalSites     int `json:"totalSites"`
	ActiveSites    int `json:"activeSites"`
	UnhealthySites int `json:"unhealthySites"`

	TotalJobs  int `json:"totalJobs"`
	ActiveJobs int `json:"activeJobs"`
	PausedJobs int `json:"pausedJobs"`

	PendingValidations    int `json:"pendingValidations"`
	ExecutionsToday       int `json:"executionsToday"`
	FailedExecutionsToday int `json:"failedExecutionsToday"`
}

func NewDashboardSummary(entity *entities.DashboardSummary) *DashboardSummary {
	d := &DashboardSummary{}
	return d.FromEntity(entity)
}

func (d *DashboardSummary) FromEntity(entity *entities.DashboardSummary) *DashboardSummary {
	d.TotalSites = entity.TotalSites
	d.ActiveSites = entity.ActiveSites
	d.UnhealthySites = entity.UnhealthySites
	d.TotalJobs = entity.TotalJobs
	d.ActiveJobs = entity.ActiveJobs
	d.PausedJobs = entity.PausedJobs
	d.PendingValidations = entity.PendingValidations
	d.ExecutionsToday = entity.ExecutionsToday
	d.FailedExecutionsToday = entity.FailedExecutionsToday
	return d
}
