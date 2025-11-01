package jobs

import (
	"context"
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

type Status string

const (
	StatusActive    Status = "active"
	StatusPaused    Status = "paused"
	StatusCompleted Status = "completed"
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
	Status             Status
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
	Value int    `json:"value"`
	Unit  string `json:"unit"`
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

type Repository interface {
	Create(ctx context.Context, job *Job) error
	GetByID(ctx context.Context, id int64) (*Job, error)
	GetAll(ctx context.Context) ([]*Job, error)
	GetActive(ctx context.Context) ([]*Job, error)
	GetDue(ctx context.Context, before time.Time) ([]*Job, error)
	Update(ctx context.Context, job *Job) error
	Delete(ctx context.Context, id int64) error

	SetCategories(ctx context.Context, jobID int64, categoryIDs []int64) error
	GetCategories(ctx context.Context, jobID int64) ([]int64, error)

	SetTopics(ctx context.Context, jobID int64, topicIDs []int64) error
	GetTopics(ctx context.Context, jobID int64) ([]int64, error)
}

type StateRepository interface {
	Get(ctx context.Context, jobID int64) (*State, error)
	Update(ctx context.Context, state *State) error
	UpdateNextRun(ctx context.Context, jobID int64, nextRun *time.Time) error
	IncrementExecutions(ctx context.Context, jobID int64, failed bool) error
	UpdateCategoryIndex(ctx context.Context, jobID int64, index int) error
}

type Service interface {
	CreateJob(ctx context.Context, job *Job) error
	GetJob(ctx context.Context, id int64) (*Job, error)
	ListJobs(ctx context.Context) ([]*Job, error)
	UpdateJob(ctx context.Context, job *Job) error
	DeleteJob(ctx context.Context, id int64) error

	PauseJob(ctx context.Context, id int64) error
	ResumeJob(ctx context.Context, id int64) error

	ExecuteManually(ctx context.Context, jobID int64) error
}
