package graph

import "github.com/davidmovas/postulator/internal/kernel/dto"

type CreateEntityRequest struct {
	SiteID   string        `json:"siteId"`
	Name     string        `json:"name" description:"Subject, 2-4 words, title case"`
	Kind     string        `json:"kind" enum:"hub,product,topic,category,custom" description:"hub roots a subject, category groups topics, topic is one subject, product is sold, custom is other"`
	Intent   string        `json:"intent,omitempty" description:"Reader's goal, one sentence"`
	Keywords []dto.Keyword `json:"keywords,omitempty" description:"Search phrases, sorted by volume"`
	Anchors  []Anchor      `json:"anchors,omitempty" description:"Link texts other pages may use"`
	ParentID string        `json:"parentId,omitempty" description:"Parent entity id; omit for a root"`
	Source   string        `json:"source,omitempty" enum:"import,user,ai" description:"Who asked; default user"`
}

type CreateEntityResponse struct {
	Entity Entity `json:"entity"`
}

type EntityInput struct {
	Name       string        `json:"name" description:"Subject, 2-4 words, title case"`
	Kind       string        `json:"kind" enum:"hub,product,topic,category,custom" description:"hub roots a subject, category groups topics, topic is one subject, product is sold, custom is other"`
	Intent     string        `json:"intent,omitempty" description:"Reader's goal, one sentence"`
	Keywords   []dto.Keyword `json:"keywords,omitempty" description:"Search phrases, sorted by volume"`
	Anchors    []Anchor      `json:"anchors,omitempty" description:"Link texts other pages may use"`
	ParentName string        `json:"parentName,omitempty" description:"Exact parent name, in this list or on the site; omit for a root"`
}

type CreateEntitiesRequest struct {
	SiteID   string        `json:"siteId"`
	Entities []EntityInput `json:"entities" description:"The tree, parents before children; written or refused whole"`
	Source   string        `json:"source,omitempty" enum:"import,user,ai" description:"Who asked; default ai"`
}

type CreateEntitiesResponse struct {
	Entities []Entity `json:"entities"`
	Edges    []Edge   `json:"edges"`
}

type UpdateEntityRequest struct {
	ID       string         `json:"id" description:"Entity id"`
	Name     *string        `json:"name,omitempty" description:"New name"`
	Kind     *string        `json:"kind,omitempty" enum:"hub,product,topic,category,custom" description:"New kind"`
	Intent   *string        `json:"intent,omitempty" description:"New reader goal"`
	Keywords *[]dto.Keyword `json:"keywords,omitempty" description:"Whole new keyword list"`
}

type UpdateEntityResponse struct {
	Entity Entity `json:"entity"`
}

type DeleteEntityRequest struct {
	ID string `json:"id" description:"Entity id"`
}

type DeleteEntityResponse struct{}

type GetEntityRequest struct {
	ID string `json:"id" description:"Entity id"`
}

type GetEntityResponse struct {
	Entity Entity `json:"entity"`
}

type ListEntitiesRequest struct {
	dto.ListRequest
	SiteID           string `json:"siteId"`
	Kind             string `json:"kind,omitempty" enum:"hub,product,topic,category,custom" description:"Only this kind"`
	HasCanonicalPage *bool  `json:"hasCanonicalPage,omitempty" description:"Only entities with, or without, a page"`
	NamePrefix       string `json:"namePrefix,omitempty" description:"Only names starting with this"`
}

type SetAnchorsRequest struct {
	EntityID string   `json:"entityId" description:"Entity id"`
	Anchors  []Anchor `json:"anchors,omitempty" description:"Whole new list; omit to clear all"`
}

type SetAnchorsResponse struct {
	Entity Entity `json:"entity"`
}

type AddEdgeRequest struct {
	SiteID       string  `json:"siteId"`
	FromEntityID string  `json:"fromEntityId" description:"Start entity id; the child of a parent edge"`
	ToEntityID   string  `json:"toEntityId" description:"End entity id; the parent of a parent edge"`
	Kind         string  `json:"kind" enum:"parent,related" description:"parent builds the tree, no cycles; related is an undirected sibling link"`
	Weight       float64 `json:"weight,omitempty" minimum:"0" maximum:"1" description:"Closeness; a parent edge takes 1, omit for none"`
	Source       string  `json:"source,omitempty" enum:"import,user,ai" description:"Who asked; default user"`
	Status       string  `json:"status,omitempty" enum:"approved,proposed,rejected" description:"Default approved; proposed leaves it to the user"`
	Reason       string  `json:"reason,omitempty" description:"Why they belong together, one sentence"`
}

type AddEdgeResponse struct {
	Edge Edge `json:"edge"`
}

type ApproveEdgeRequest struct {
	ID string `json:"id" description:"Proposed edge id"`
}

type ApproveEdgeResponse struct {
	Edge Edge `json:"edge"`
}

type RejectEdgeRequest struct {
	ID string `json:"id" description:"Proposed edge id"`
}

type RejectEdgeResponse struct {
	Edge Edge `json:"edge"`
}

type MoveEntityRequest struct {
	EntityID    string `json:"entityId" description:"Entity id"`
	NewParentID string `json:"newParentId" description:"New parent entity id"`
	KeepBoth    bool   `json:"keepBoth,omitempty" description:"Keep the old parent too, giving two parents"`
}

type MoveEntityResponse struct {
	Edge           Edge     `json:"edge"`
	RemovedEdgeIDs []string `json:"removedEdgeIds"`
}

type DeleteEdgeRequest struct {
	ID string `json:"id" description:"Edge id"`
}

type DeleteEdgeResponse struct{}

type ListEdgesRequest struct {
	dto.ListRequest
	SiteID   string `json:"siteId"`
	Kind     string `json:"kind,omitempty" enum:"parent,related" description:"Only this kind"`
	Status   string `json:"status,omitempty" enum:"approved,proposed,rejected" description:"Only this state"`
	EntityID string `json:"entityId,omitempty" description:"Only edges touching this entity"`
}

type LoadGraphRequest struct {
	SiteID string `json:"siteId"`
}

type LoadGraphResponse struct {
	Entities []Entity     `json:"entities"`
	Edges    []Edge       `json:"edges"`
	Pages    []EntityPage `json:"pages"`
}

type RecomputeScoresRequest struct {
	SiteID string `json:"siteId"`
}

type RecomputeScoresResponse struct {
	Scores map[string]float64 `json:"scores"`
}
