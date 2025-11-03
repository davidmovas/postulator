package execution

import (
	"context"
	"time"

	"github.com/davidmovas/postulator/internal/domain/entities"
)

type Repository interface {
	Create(ctx context.Context, exec *entities.Execution) error
	GetByID(ctx context.Context, id int64) (*entities.Execution, error)
	GetByJobID(ctx context.Context, jobID int64, limit, offset int) ([]*entities.Execution, int, error)
	GetPendingValidation(ctx context.Context) ([]*entities.Execution, error)
	GetByStatus(ctx context.Context, status entities.Status) ([]*entities.Execution, error)
	Update(ctx context.Context, exec *entities.Execution) error
	Delete(ctx context.Context, id int64) error

	CountByJob(ctx context.Context, jobID int64) (int, error)
	GetTotalCost(ctx context.Context, from, to time.Time) (float64, error)
	GetTotalTokens(ctx context.Context, from, to time.Time) (int, error)
	GetAverageGenerationTime(ctx context.Context, jobID int64) (int, error)
}

type Service interface {
	CreateExecution(ctx context.Context, exec *entities.Execution) error
	GetExecution(ctx context.Context, id int64) (*entities.Execution, error)
	ListExecutions(ctx context.Context, jobID int64, limit, offset int) ([]*entities.Execution, int, error)
	GetPendingValidations(ctx context.Context) ([]*entities.Execution, error)

	UpdateStatus(ctx context.Context, id int64, status entities.ExecutionStatus) error

	ApproveExecution(ctx context.Context, id int64) error
	RejectExecution(ctx context.Context, id int64) error

	GetJobMetrics(ctx context.Context, jobID int64) (*entities.Metrics, error)
}
