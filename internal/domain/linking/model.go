package linking

import (
	"context"
	"time"
)

type PlanStatus string

const (
	PlanStatusDraft      PlanStatus = "draft"
	PlanStatusSuggesting PlanStatus = "suggesting"
	PlanStatusReady      PlanStatus = "ready"
	PlanStatusApplying   PlanStatus = "applying"
	PlanStatusApplied    PlanStatus = "applied"
	PlanStatusFailed     PlanStatus = "failed"
)

type LinkStatus string

const (
	LinkStatusPlanned  LinkStatus = "planned"
	LinkStatusApproved LinkStatus = "approved"
	LinkStatusRejected LinkStatus = "rejected"
	LinkStatusApplying LinkStatus = "applying"
	LinkStatusApplied  LinkStatus = "applied"
	LinkStatusFailed   LinkStatus = "failed"
)

type LinkSource string

const (
	LinkSourceAI     LinkSource = "ai"
	LinkSourceManual LinkSource = "manual"
)

type LinkPlan struct {
	ID         int64
	SitemapID  int64
	SiteID     int64
	Name       string
	Status     PlanStatus
	ProviderID *int64
	PromptID   *int64
	Error      *string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type PlannedLink struct {
	ID            int64
	PlanID        int64
	SourceNodeID  int64
	TargetNodeID  int64
	AnchorText    *string
	AnchorContext *string
	Status        LinkStatus
	Source        LinkSource
	Position      *int
	Confidence    *float64
	Error         *string
	AppliedAt     *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type AppliedLinkInfo struct {
	LinkID      int64
	AnchorText  string
	Position    int
	TargetURL   string
	TargetTitle string
}

type PlanRepository interface {
	Create(ctx context.Context, plan *LinkPlan) error
	GetByID(ctx context.Context, id int64) (*LinkPlan, error)
	GetBySitemapID(ctx context.Context, sitemapID int64) (*LinkPlan, error)
	GetActiveBySitemapID(ctx context.Context, sitemapID int64) (*LinkPlan, error)
	List(ctx context.Context, siteID int64) ([]*LinkPlan, error)
	Update(ctx context.Context, plan *LinkPlan) error
	Delete(ctx context.Context, id int64) error
}

type LinkRepository interface {
	Create(ctx context.Context, link *PlannedLink) error
	CreateBatch(ctx context.Context, links []*PlannedLink) error
	GetByID(ctx context.Context, id int64) (*PlannedLink, error)
	GetByPlanID(ctx context.Context, planID int64) ([]*PlannedLink, error)
	GetBySourceNodeID(ctx context.Context, planID int64, nodeID int64) ([]*PlannedLink, error)
	GetByTargetNodeID(ctx context.Context, planID int64, nodeID int64) ([]*PlannedLink, error)
	GetByNodePair(ctx context.Context, planID int64, sourceID int64, targetID int64) (*PlannedLink, error)
	Update(ctx context.Context, link *PlannedLink) error
	UpdateStatus(ctx context.Context, id int64, status LinkStatus, errorMsg *string) error
	Delete(ctx context.Context, id int64) error
	DeleteByPlanID(ctx context.Context, planID int64) error
	CountByStatus(ctx context.Context, planID int64, status LinkStatus) (int, error)
}

type Service interface {
	CreatePlan(ctx context.Context, sitemapID int64, siteID int64, name string) (*LinkPlan, error)
	GetPlan(ctx context.Context, id int64) (*LinkPlan, error)
	GetPlanBySitemap(ctx context.Context, sitemapID int64) (*LinkPlan, error)
	GetActivePlan(ctx context.Context, sitemapID int64) (*LinkPlan, error)
	ListPlans(ctx context.Context, siteID int64) ([]*LinkPlan, error)
	UpdatePlan(ctx context.Context, plan *LinkPlan) error
	DeletePlan(ctx context.Context, id int64) error

	AddLink(ctx context.Context, planID int64, sourceNodeID int64, targetNodeID int64) (*PlannedLink, error)
	RemoveLink(ctx context.Context, linkID int64) error
	UpdateLink(ctx context.Context, link *PlannedLink) error
	GetLinks(ctx context.Context, planID int64) ([]*PlannedLink, error)
	GetLinksByNode(ctx context.Context, planID int64, nodeID int64) ([]*PlannedLink, error)
	ApproveLink(ctx context.Context, linkID int64) error
	RejectLink(ctx context.Context, linkID int64) error
	ApproveAndApplyLink(ctx context.Context, linkID int64) error

	SuggestLinks(ctx context.Context, config SuggestLinksConfig) error
	ApplyLinks(ctx context.Context, planID int64, linkIDs []int64, providerID int64) (*ApplyResult, error)

	GetLinkGraph(ctx context.Context, planID int64) (*LinkGraph, error)
	GetOrCreateActivePlan(ctx context.Context, sitemapID int64, siteID int64) (*LinkPlan, error)
}

type LinkGraph struct {
	Nodes []*GraphNode `json:"nodes"`
	Edges []*GraphEdge `json:"edges"`
}

type GraphNode struct {
	NodeID            int64  `json:"nodeId"`
	Title             string `json:"title"`
	Slug              string `json:"slug"`
	Path              string `json:"path"`
	HasContent        bool   `json:"hasContent"`
	OutgoingLinkCount int    `json:"outgoingLinkCount"`
	IncomingLinkCount int    `json:"incomingLinkCount"`
}

type GraphEdge struct {
	ID           int64      `json:"id"`
	SourceNodeID int64      `json:"sourceNodeId"`
	TargetNodeID int64      `json:"targetNodeId"`
	AnchorText   *string    `json:"anchorText"`
	Status       LinkStatus `json:"status"`
	Source       LinkSource `json:"source"`
	Confidence   *float64   `json:"confidence"`
}

type SuggestLinksConfig struct {
	PlanID      int64
	ProviderID  int64
	PromptID    *int64
	NodeIDs     []int64
	Feedback    string
	MaxIncoming int
	MaxOutgoing int
}

type SuggestionRequest struct {
	PlanID     int64
	ProviderID int64
	PromptID   *int64
	NodeIDs    []int64
	Feedback   string
}

type SuggestionResult struct {
	Links       []*SuggestedLink `json:"links"`
	Explanation string           `json:"explanation"`
}

type SuggestedLink struct {
	SourceNodeID int64   `json:"sourceNodeId"`
	TargetNodeID int64   `json:"targetNodeId"`
	AnchorText   string  `json:"anchorText"`
	Reason       string  `json:"reason"`
	Confidence   float64 `json:"confidence"`
}

type ApplyRequest struct {
	PlanID  int64
	LinkIDs []int64
	SiteID  int64
}

type ApplyResult struct {
	TotalLinks   int                `json:"totalLinks"`
	AppliedLinks int                `json:"appliedLinks"`
	FailedLinks  int                `json:"failedLinks"`
	Results      []*AppliedLinkInfo `json:"results"`
}
