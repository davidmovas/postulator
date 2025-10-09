package entities

import "time"

type TopicStrategy string

const (
	StrategyUnique             TopicStrategy = "unique"
	StrategyReuseWithVariation TopicStrategy = "reuse_with_variation"
)

type Topic struct {
	ID        int64
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Title struct {
	ID        int64
	TopicID   int64
	Title     string
	CreatedAt time.Time
}

type SiteTopic struct {
	ID         int64
	SiteID     int64
	TopicID    int64
	CategoryID *int64
	Strategy   TopicStrategy
	CreatedAt  time.Time
}

type UsedTitle struct {
	ID      int64
	SiteID  int64
	TitleID int64
	UsedAt  time.Time
}
