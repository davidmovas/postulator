package entities

import (
	"encoding/json"
	"time"
)

type TopicStrategy string

const (
	StrategyUnique    TopicStrategy = "unique"
	StrategyVariation TopicStrategy = "reuse_with_variation"
)

type CategoryStrategy string

const (
	CategoryFixed  CategoryStrategy = "fixed"
	CategoryRandom CategoryStrategy = "random"
	CategoryRotate CategoryStrategy = "rotate"
)

type JobStatus string

const (
	JobStatusActive    JobStatus = "active"
	JobStatusPaused    JobStatus = "paused"
	JobStatusCompleted JobStatus = "completed"
)

type Job struct {
	ID                 int64
	Name               string
	SiteID             int64
	PromptID           int64
	AIProviderID       int64
	TopicStrategy      TopicStrategy
	CategoryStrategy   CategoryStrategy
	RequiresValidation bool
	JitterEnabled      bool
	JitterMinutes      int
	Status             JobStatus
	CreatedAt          time.Time
	UpdatedAt          time.Time

	Schedule   *Schedule
	State      *State
	Categories []int64
	Topics     []int64
}

type ScheduleType string

const (
	ScheduleManual   ScheduleType = "manual"
	ScheduleOnce     ScheduleType = "once"
	ScheduleInterval ScheduleType = "interval"
	ScheduleDaily    ScheduleType = "daily"
)

type Schedule struct {
	Type   ScheduleType
	Config json.RawMessage
}

type OnceSchedule struct {
	ExecuteAt time.Time `json:"execute_at"`
}

type IntervalSchedule struct {
	Value   int        `json:"value"`
	Unit    string     `json:"unit"`
	StartAt *time.Time `json:"start_at,omitempty"`
}

type DailySchedule struct {
	Hour     int   `json:"hour"`
	Minute   int   `json:"minute"`
	Weekdays []int `json:"weekdays"`
}

type State struct {
	JobID             int64
	LastRunAt         *time.Time
	NextRunAt         *time.Time
	TotalExecutions   int
	FailedExecutions  int
	LastCategoryIndex int
}
