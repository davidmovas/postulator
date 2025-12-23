package dto

import (
	"time"

	"github.com/davidmovas/postulator/internal/domain/linking"
)

// LinkPlan DTOs

type LinkPlan struct {
	ID         int64     `json:"id"`
	SitemapID  int64     `json:"sitemapId"`
	SiteID     int64     `json:"siteId"`
	Name       string    `json:"name"`
	Status     string    `json:"status"`
	ProviderID *int64    `json:"providerId,omitempty"`
	PromptID   *int64    `json:"promptId,omitempty"`
	Error      *string   `json:"error,omitempty"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

func NewLinkPlan(p *linking.LinkPlan) *LinkPlan {
	return &LinkPlan{
		ID:         p.ID,
		SitemapID:  p.SitemapID,
		SiteID:     p.SiteID,
		Name:       p.Name,
		Status:     string(p.Status),
		ProviderID: p.ProviderID,
		PromptID:   p.PromptID,
		Error:      p.Error,
		CreatedAt:  p.CreatedAt,
		UpdatedAt:  p.UpdatedAt,
	}
}

func NewLinkPlans(plans []*linking.LinkPlan) []*LinkPlan {
	result := make([]*LinkPlan, len(plans))
	for i, p := range plans {
		result[i] = NewLinkPlan(p)
	}
	return result
}

// PlannedLink DTOs

type PlannedLink struct {
	ID            int64      `json:"id"`
	PlanID        int64      `json:"planId"`
	SourceNodeID  int64      `json:"sourceNodeId"`
	TargetNodeID  int64      `json:"targetNodeId"`
	AnchorText    *string    `json:"anchorText,omitempty"`
	AnchorContext *string    `json:"anchorContext,omitempty"`
	Status        string     `json:"status"`
	Source        string     `json:"source"`
	Position      *int       `json:"position,omitempty"`
	Confidence    *float64   `json:"confidence,omitempty"`
	Error         *string    `json:"error,omitempty"`
	AppliedAt     *time.Time `json:"appliedAt,omitempty"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}

func NewPlannedLink(l *linking.PlannedLink) *PlannedLink {
	return &PlannedLink{
		ID:            l.ID,
		PlanID:        l.PlanID,
		SourceNodeID:  l.SourceNodeID,
		TargetNodeID:  l.TargetNodeID,
		AnchorText:    l.AnchorText,
		AnchorContext: l.AnchorContext,
		Status:        string(l.Status),
		Source:        string(l.Source),
		Position:      l.Position,
		Confidence:    l.Confidence,
		Error:         l.Error,
		AppliedAt:     l.AppliedAt,
		CreatedAt:     l.CreatedAt,
		UpdatedAt:     l.UpdatedAt,
	}
}

func NewPlannedLinks(links []*linking.PlannedLink) []*PlannedLink {
	result := make([]*PlannedLink, len(links))
	for i, l := range links {
		result[i] = NewPlannedLink(l)
	}
	return result
}

// LinkGraph DTOs

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
	ID           int64    `json:"id"`
	SourceNodeID int64    `json:"sourceNodeId"`
	TargetNodeID int64    `json:"targetNodeId"`
	AnchorText   *string  `json:"anchorText,omitempty"`
	Status       string   `json:"status"`
	Source       string   `json:"source"`
	Confidence   *float64 `json:"confidence,omitempty"`
}

func NewLinkGraph(g *linking.LinkGraph) *LinkGraph {
	nodes := make([]*GraphNode, len(g.Nodes))
	for i, n := range g.Nodes {
		nodes[i] = &GraphNode{
			NodeID:            n.NodeID,
			Title:             n.Title,
			Slug:              n.Slug,
			Path:              n.Path,
			HasContent:        n.HasContent,
			OutgoingLinkCount: n.OutgoingLinkCount,
			IncomingLinkCount: n.IncomingLinkCount,
		}
	}

	edges := make([]*GraphEdge, len(g.Edges))
	for i, e := range g.Edges {
		edges[i] = &GraphEdge{
			ID:           e.ID,
			SourceNodeID: e.SourceNodeID,
			TargetNodeID: e.TargetNodeID,
			AnchorText:   e.AnchorText,
			Status:       string(e.Status),
			Source:       string(e.Source),
			Confidence:   e.Confidence,
		}
	}

	return &LinkGraph{
		Nodes: nodes,
		Edges: edges,
	}
}

// Request DTOs

type CreateLinkPlanRequest struct {
	SitemapID int64  `json:"sitemapId"`
	SiteID    int64  `json:"siteId"`
	Name      string `json:"name"`
}

type AddLinkRequest struct {
	PlanID       int64 `json:"planId"`
	SourceNodeID int64 `json:"sourceNodeId"`
	TargetNodeID int64 `json:"targetNodeId"`
}

type UpdateLinkRequest struct {
	ID            int64   `json:"id"`
	AnchorText    *string `json:"anchorText,omitempty"`
	AnchorContext *string `json:"anchorContext,omitempty"`
}

type SuggestLinksRequest struct {
	PlanID     int64   `json:"planId"`
	ProviderID int64   `json:"providerId"`
	PromptID   *int64  `json:"promptId,omitempty"`
	NodeIDs    []int64 `json:"nodeIds,omitempty"`
	Feedback   string  `json:"feedback,omitempty"`
}

type ApplyLinksRequest struct {
	PlanID  int64   `json:"planId"`
	LinkIDs []int64 `json:"linkIds"`
}
