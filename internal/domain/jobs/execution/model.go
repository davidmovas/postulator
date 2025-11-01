package execution

import (
	"context"
	"time"
)

type Status string

const (
	StatusPending           Status = "pending"
	StatusGenerating        Status = "generating"
	StatusPendingValidation Status = "pending_validation"
	StatusValidated         Status = "validated"
	StatusPublishing        Status = "publishing"
	StatusPublished         Status = "published"
	StatusRejected          Status = "rejected"
	StatusFailed            Status = "failed"
)

type Execution struct {
	ID        int64
	JobID     int64
	TopicID   int64
	ArticleID *int64

	PromptID     int64
	AIProviderID int64
	AIModel      string
	CategoryID   int64

	Status           Status
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

type Repository interface {
	Create(ctx context.Context, exec *Execution) error
	GetByID(ctx context.Context, id int64) (*Execution, error)
	GetByJobID(ctx context.Context, jobID int64, limit, offset int) ([]*Execution, int, error)
	GetPendingValidation(ctx context.Context) ([]*Execution, error)
	GetByStatus(ctx context.Context, status Status) ([]*Execution, error)
	Update(ctx context.Context, exec *Execution) error
	Delete(ctx context.Context, id int64) error

	CountByJob(ctx context.Context, jobID int64) (int, error)
	GetTotalCost(ctx context.Context, from, to time.Time) (float64, error)
	GetTotalTokens(ctx context.Context, from, to time.Time) (int, error)
	GetAverageGenerationTime(ctx context.Context, jobID int64) (int, error)
}

type Service interface {
	CreateExecution(ctx context.Context, exec *Execution) error
	GetExecution(ctx context.Context, id int64) (*Execution, error)
	ListExecutions(ctx context.Context, jobID int64, limit, offset int) ([]*Execution, int, error)
	GetPendingValidations(ctx context.Context) ([]*Execution, error)

	UpdateStatus(ctx context.Context, id int64, status Status) error

	ApproveExecution(ctx context.Context, id int64) error
	RejectExecution(ctx context.Context, id int64) error

	GetJobMetrics(ctx context.Context, jobID int64) (*Metrics, error)
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
