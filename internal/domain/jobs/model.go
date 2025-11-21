package jobs

import (
	"context"
	"time"

	"github.com/davidmovas/postulator/internal/domain/entities"
)

type Repository interface {
	Create(ctx context.Context, job *entities.Job) error
	GetByID(ctx context.Context, id int64) (*entities.Job, error)
	GetAll(ctx context.Context) ([]*entities.Job, error)
	GetActive(ctx context.Context) ([]*entities.Job, error)
	GetDue(ctx context.Context, before time.Time) ([]*entities.Job, error)
	Update(ctx context.Context, job *entities.Job) error
	Delete(ctx context.Context, id int64) error

	SetCategories(ctx context.Context, jobID int64, categoryIDs []int64) error
	GetCategories(ctx context.Context, jobID int64) ([]int64, error)

	SetTopics(ctx context.Context, jobID int64, topicIDs []int64) error
	GetTopics(ctx context.Context, jobID int64) ([]int64, error)
}

type StateRepository interface {
	Get(ctx context.Context, jobID int64) (*entities.State, error)
	Update(ctx context.Context, state *entities.State) error
	UpdateNextRun(ctx context.Context, jobID int64, nextRun *time.Time) error
	IncrementExecutions(ctx context.Context, jobID int64, failed bool) error
	UpdateCategoryIndex(ctx context.Context, jobID int64, index int) error
}

type Service interface {
	CreateJob(ctx context.Context, job *entities.Job) error
	GetJob(ctx context.Context, id int64) (*entities.Job, error)
	ListJobs(ctx context.Context) ([]*entities.Job, error)
	UpdateJob(ctx context.Context, job *entities.Job) error
	DeleteJob(ctx context.Context, id int64) error

	PauseJob(ctx context.Context, id int64) error
	ResumeJob(ctx context.Context, id int64) error

	ExecuteManually(ctx context.Context, jobID int64) error
}

type Scheduler interface {
	Start(ctx context.Context) error
	Stop() error
	RestoreState(ctx context.Context) error
	CalculateNextRun(job *entities.Job, lastRun *time.Time) (baseTime time.Time, withJitter time.Time, err error)
	ScheduleJob(ctx context.Context, job *entities.Job) error
	TriggerJob(ctx context.Context, jobID int64) error
}

type Executor interface {
	Execute(ctx context.Context, job *entities.Job) error
	PublishValidatedArticle(ctx context.Context, exec *entities.Execution) error
}
