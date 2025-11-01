package linking

import (
	"context"
	"time"
)

type TaskStatus string

const (
	StatusPending   TaskStatus = "pending"
	StatusAnalyzing TaskStatus = "analyzing"
	StatusReady     TaskStatus = "ready"
	StatusApplying  TaskStatus = "applying"
	StatusApplied   TaskStatus = "applied"
	StatusFailed    TaskStatus = "failed"
)

type ProposalStatus string

const (
	ProposalPending  ProposalStatus = "pending"
	ProposalApproved ProposalStatus = "approved"
	ProposalRejected ProposalStatus = "rejected"
	ProposalModified ProposalStatus = "modified"
)

type Task struct {
	ID                 int64
	Name               string
	SiteIDs            []int64
	ArticleIDs         []int64
	MaxLinksPerArticle int
	MinLinkDistance    int
	PromptID           *int64
	AIProviderID       int64
	Status             TaskStatus
	ErrorMessage       *string
	CreatedAt          time.Time
	StartedAt          *time.Time
	CompletedAt        *time.Time
	AppliedAt          *time.Time
}

type Proposal struct {
	ID              int64
	TaskID          int64
	SourceArticleID int64
	TargetArticleID int64
	AnchorText      string
	Position        int
	Confidence      *float64
	Status          ProposalStatus
	CreatedAt       time.Time
}

type Link struct {
	ID              int64
	ArticleID       int64
	LinkType        string
	TargetArticleID *int64
	URL             string
	AnchorText      string
	Position        *int
	TaskID          *int64
	CreatedAt       time.Time
}

type Repository interface {
	CreateTask(ctx context.Context, task *Task) error
	GetTaskByID(ctx context.Context, id int64) (*Task, error)
	GetAllTasks(ctx context.Context) ([]*Task, error)
	GetPendingTasks(ctx context.Context) ([]*Task, error)
	UpdateTask(ctx context.Context, task *Task) error
	DeleteTask(ctx context.Context, id int64) error
}

type ProposalRepository interface {
	CreateProposal(ctx context.Context, proposal *Proposal) error
	CreateBatch(ctx context.Context, proposals []*Proposal) error
	GetByID(ctx context.Context, id int64) (*Proposal, error)
	GetByTaskID(ctx context.Context, taskID int64) ([]*Proposal, error)
	Update(ctx context.Context, proposal *Proposal) error
	Delete(ctx context.Context, id int64) error
	DeleteByTaskID(ctx context.Context, taskID int64) error

	CountByStatus(ctx context.Context, taskID int64, status ProposalStatus) (int, error)
}

type LinkRepository interface {
	CreateLink(ctx context.Context, link *Link) error
	GetByArticleID(ctx context.Context, articleID int64) ([]*Link, error)
	GetByTaskID(ctx context.Context, taskID int64) ([]*Link, error)
	Update(ctx context.Context, link *Link) error
	Delete(ctx context.Context, id int64) error
	DeleteByArticleID(ctx context.Context, articleID int64) error
}

type Service interface {
	CreateTask(ctx context.Context, task *Task) error
	GetTask(ctx context.Context, id int64) (*Task, error)
	ListTasks(ctx context.Context) ([]*Task, error)
	DeleteTask(ctx context.Context, id int64) error

	AnalyzeTask(ctx context.Context, id int64) error

	GetProposals(ctx context.Context, taskID int64) ([]*Proposal, error)
	UpdateProposal(ctx context.Context, proposal *Proposal) error
	ApproveProposal(ctx context.Context, id int64) error
	RejectProposal(ctx context.Context, id int64) error

	ApplyTask(ctx context.Context, taskID int64) error

	GetTaskGraph(ctx context.Context, taskID int64) (*LinkGraph, error)

	GetArticleLinks(ctx context.Context, articleID int64) ([]*Link, error)
	DeleteLink(ctx context.Context, id int64) error
}

type LinkGraph struct {
	Nodes []*GraphNode
	Edges []*GraphEdge
}

type GraphNode struct {
	ArticleID int64
	Title     string
	URL       string
}

type GraphEdge struct {
	SourceID   int64
	TargetID   int64
	AnchorText string
	Status     ProposalStatus
	Confidence *float64
}
