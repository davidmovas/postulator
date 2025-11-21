package entities

import "time"

type ExecutionStatus string

const (
	ExecutionStatusPending           ExecutionStatus = "pending"
	ExecutionStatusGenerating        ExecutionStatus = "generating"
	ExecutionStatusPendingValidation ExecutionStatus = "pending_validation"
	ExecutionStatusValidated         ExecutionStatus = "validated"
	ExecutionStatusPublishing        ExecutionStatus = "publishing"
	ExecutionStatusPublished         ExecutionStatus = "published"
	ExecutionStatusRejected          ExecutionStatus = "rejected"
	ExecutionStatusFailed            ExecutionStatus = "failed"
)

type Execution struct {
	ID        int64
	JobID     int64
	SiteID    int64
	TopicID   int64
	ArticleID *int64

	PromptID     int64
	AIProviderID int64
	AIModel      string
	CategoryIDs  []int64

	Status           ExecutionStatus
	ErrorMessage     *string
	GenerationTimeMs *int
	TokensUsed       *int
	CostUSD          *float64

	StartedAt   time.Time
	GeneratedAt *time.Time
	ValidatedAt *time.Time
	PublishedAt *time.Time
	CompletedAt *time.Time
}

type Metrics struct {
	TotalExecutions      int
	SuccessfulExecutions int
	FailedExecutions     int
	RejectedExecutions   int
	AverageTimeMs        int
	TotalTokens          int
	TotalCost            float64
}
