package job

import "time"

type ScheduleType string

const (
	ScheduleManual  ScheduleType = "manual"
	ScheduleOnce    ScheduleType = "once"
	ScheduleDaily   ScheduleType = "daily"
	ScheduleWeekly  ScheduleType = "weekly"
	ScheduleMonthly ScheduleType = "monthly"
)

type Status string

const (
	StatusActive    Status = "active"
	StatusPaused    Status = "paused"
	StatusCompleted Status = "completed"
	StatusError     Status = "error"
)

type Job struct {
	ID           int64
	Name         string
	SiteID       int64
	TopicID      int64
	PromptID     int64
	AIProviderID int64

	MaxWords           int
	CustomPlaceholders map[string]string

	RequiresValidation bool

	ScheduleType ScheduleType
	ScheduleTime *time.Time
	ScheduleDay  *int

	JitterEnabled bool
	JitterMinutes int

	Status    Status
	LastRunAt *time.Time
	NextRunAt *time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
}

type ExecutionStatus string

const (
	ExecutionPendingValidation ExecutionStatus = "pending_validation"
	ExecutionValidated         ExecutionStatus = "validated"
	ExecutionPublished         ExecutionStatus = "published"
	ExecutionFailed            ExecutionStatus = "failed"
)

type Execution struct {
	ID      int64
	JobID   int64
	TitleID int64

	GeneratedTitle   string
	GeneratedContent string

	Status       ExecutionStatus
	ErrorMessage string

	WPPostID  *int
	WPPostURL string

	StartedAt   time.Time
	ValidatedAt *time.Time
	PublishedAt *time.Time
}
