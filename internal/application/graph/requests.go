package graph

import "github.com/davidmovas/postulator/internal/kernel/dto"

type CreateEntityRequest struct {
	SiteID            string   `json:"siteId"`
	Name              string   `json:"name"`
	Kind              string   `json:"kind"`
	Intent            string   `json:"intent"`
	PrimaryKeyword    string   `json:"primaryKeyword"`
	SecondaryKeywords []string `json:"secondaryKeywords"`
	Anchors           []Anchor `json:"anchors"`
	Source            string   `json:"source,omitempty"`
}

type CreateEntityResponse struct {
	Entity Entity `json:"entity"`
}

type UpdateEntityRequest struct {
	ID                string    `json:"id"`
	Name              *string   `json:"name,omitempty"`
	Kind              *string   `json:"kind,omitempty"`
	Intent            *string   `json:"intent,omitempty"`
	PrimaryKeyword    *string   `json:"primaryKeyword,omitempty"`
	SecondaryKeywords *[]string `json:"secondaryKeywords,omitempty"`
}

type UpdateEntityResponse struct {
	Entity Entity `json:"entity"`
}

type DeleteEntityRequest struct {
	ID string `json:"id"`
}

type DeleteEntityResponse struct{}

type GetEntityRequest struct {
	ID string `json:"id"`
}

type GetEntityResponse struct {
	Entity Entity `json:"entity"`
}

type ListEntitiesRequest struct {
	dto.ListRequest
	SiteID           string `json:"siteId"`
	Kind             string `json:"kind,omitempty"`
	HasCanonicalPage *bool  `json:"hasCanonicalPage,omitempty"`
	NamePrefix       string `json:"namePrefix,omitempty"`
}

type SetAnchorsRequest struct {
	EntityID string   `json:"entityId"`
	Anchors  []Anchor `json:"anchors"`
}

type SetAnchorsResponse struct {
	Entity Entity `json:"entity"`
}

type AddEdgeRequest struct {
	SiteID       string  `json:"siteId"`
	FromEntityID string  `json:"fromEntityId"`
	ToEntityID   string  `json:"toEntityId"`
	Kind         string  `json:"kind"`
	Weight       float64 `json:"weight"`
	Source       string  `json:"source,omitempty"`
	Status       string  `json:"status,omitempty"`
	Reason       string  `json:"reason,omitempty" description:"Why the two belong together, one short sentence"`
}

type AddEdgeResponse struct {
	Edge Edge `json:"edge"`
}

type ApproveEdgeRequest struct {
	ID string `json:"id"`
}

type ApproveEdgeResponse struct {
	Edge Edge `json:"edge"`
}

type RejectEdgeRequest struct {
	ID string `json:"id"`
}

type RejectEdgeResponse struct {
	Edge Edge `json:"edge"`
}

type DeleteEdgeRequest struct {
	ID string `json:"id"`
}

type DeleteEdgeResponse struct{}

type ListEdgesRequest struct {
	dto.ListRequest
	SiteID   string `json:"siteId"`
	Kind     string `json:"kind,omitempty"`
	Status   string `json:"status,omitempty"`
	EntityID string `json:"entityId,omitempty"`
}

type LoadGraphRequest struct {
	SiteID string `json:"siteId"`
}

type LoadGraphResponse struct {
	Entities []Entity `json:"entities"`
	Edges    []Edge   `json:"edges"`
}

type RecomputeScoresRequest struct {
	SiteID string `json:"siteId"`
}

type RecomputeScoresResponse struct {
	Scores map[string]float64 `json:"scores"`
}
