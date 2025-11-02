package entities

import "time"

type SiteStats struct {
	ID                   int64
	SiteID               int64
	Date                 time.Time
	ArticlesPublished    int
	ArticlesFailed       int
	TotalWords           int
	InternalLinksCreated int
	ExternalLinksCreated int
}

type DashboardSummary struct {
	TotalSites     int
	ActiveSites    int
	UnhealthySites int

	TotalArticles      int
	TotalArticlesToday int
	TotalWordsToday    int

	TotalJobs  int
	ActiveJobs int
	PausedJobs int

	PendingValidations    int
	ExecutionsToday       int
	FailedExecutionsToday int
}
