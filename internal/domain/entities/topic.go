package entities

import "time"

type TopicStrategy string

const (
	StrategyUnique    TopicStrategy = "unique"
	StrategyVariation TopicStrategy = "variation"
)

type Topic struct {
	ID        int64
	Title     string
	CreatedAt time.Time
}

type SiteTopic struct {
	ID         int64
	SiteID     int64
	TopicID    int64
	CategoryID int64
	Strategy   TopicStrategy
	CreatedAt  time.Time
}

type UsedTopic struct {
	ID      int64
	SiteID  int64
	TopicID int64
	UsedAt  time.Time
}
