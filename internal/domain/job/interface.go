package job

import (
	"context"
	"time"
)

type IRepository interface {
	Create(ctx context.Context, job *Job) error
	GetByID(ctx context.Context, id int64) (*Job, error)
	GetAll(ctx context.Context) ([]*Job, error)
	GetActive(ctx context.Context) ([]*Job, error)
	Update(ctx context.Context, job *Job) error
	Delete(ctx context.Context, id int64) error

	GetDueJobs(ctx context.Context, now time.Time) ([]*Job, error)
}

type IExecutionRepository interface {
	Create(ctx context.Context, exec *Execution) error
	GetByID(ctx context.Context, id int64) (*Execution, error)
	GetByJobID(ctx context.Context, jobID int64) ([]*Execution, error)
	GetPendingValidation(ctx context.Context) ([]*Execution, error)
	Update(ctx context.Context, exec *Execution) error
	Delete(ctx context.Context, id int64) error
}

type IService interface {
	CreateJob(ctx context.Context, job *Job) error
	GetJob(ctx context.Context, id int64) (*Job, error)
	ListJobs(ctx context.Context) ([]*Job, error)
	UpdateJob(ctx context.Context, job *Job) error
	DeleteJob(ctx context.Context, id int64) error
	PauseJob(ctx context.Context, id int64) error
	ResumeJob(ctx context.Context, id int64) error

	ExecuteJobManually(ctx context.Context, jobID int64) error
	ValidateExecution(ctx context.Context, execID int64, approved bool) error
	GetPendingValidations(ctx context.Context) ([]*Execution, error)
}

type IScheduler interface {
	Start(ctx context.Context) error
	Stop() error
	RestoreState(ctx context.Context) error
	CalculateNextRun(job *Job, now time.Time) time.Time
}

type IExecutor interface {
	Execute(ctx context.Context, job *Job) error
}
