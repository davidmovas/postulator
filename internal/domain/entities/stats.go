package entities

import "time"

type Statistics struct {
	ID                int64
	SiteID            int64
	Date              time.Time
	ArticlesGenerated int
	ArticlesPublished int
	ArticlesFailed    int
}
