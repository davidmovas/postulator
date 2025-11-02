package entities

import "time"

type Category struct {
	ID           int64
	SiteID       int64
	WPCategoryID int
	Name         string
	Slug         *string
	Description  *string
	Count        int
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Statistics struct {
	CategoryID        int64
	Date              time.Time
	ArticlesPublished int
	TotalWords        int
}
