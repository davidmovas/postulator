package graph

import "github.com/davidmovas/postulator/internal/kernel/dto"

type CreateEntityRequest struct {
	SiteID            string   `json:"siteId"`
	Name              string   `json:"name" description:"What the entity is about, two to four words in title case"`
	Kind              string   `json:"kind" enum:"hub,product,topic,category,custom" description:"Where the entity sits in the graph: a hub is a subject root, a category groups topics, a topic is one subject, a product is a thing sold, custom is anything else"`
	Intent            string   `json:"intent,omitempty" description:"What a reader wants from the page, one short sentence"`
	PrimaryKeyword    string   `json:"primaryKeyword,omitempty" description:"The phrase a reader searches to reach the page"`
	SecondaryKeywords []string `json:"secondaryKeywords,omitempty" description:"Further phrases the page can rank for, at most four"`
	Anchors           []Anchor `json:"anchors,omitempty" description:"The link texts another page may point here with; leave it out and the anchors can be set later"`
	Source            string   `json:"source,omitempty" enum:"import,user,ai" description:"Who asked for the entity; leave it out and it is recorded as user"`
}

type CreateEntityResponse struct {
	Entity Entity `json:"entity"`
}

type EntityInput struct {
	Name              string   `json:"name" description:"What the entity is about, two to four words in title case"`
	Kind              string   `json:"kind" enum:"hub,product,topic,category,custom" description:"Where the entity sits in the graph: a hub is a subject root, a category groups topics, a topic is one subject, a product is a thing sold, custom is anything else"`
	Intent            string   `json:"intent,omitempty" description:"What a reader wants from the page, one short sentence"`
	PrimaryKeyword    string   `json:"primaryKeyword,omitempty" description:"The phrase a reader searches to reach the page"`
	SecondaryKeywords []string `json:"secondaryKeywords,omitempty" description:"Further phrases the page can rank for, at most four"`
	Anchors           []Anchor `json:"anchors,omitempty" description:"The link texts another page may point here with"`
	ParentName        string   `json:"parentName,omitempty" description:"The name of the entity one level up, spelled exactly as it is written in this list or as it already exists on the site; leave it out for a root of the tree"`
}

type CreateEntitiesRequest struct {
	SiteID   string        `json:"siteId"`
	Entities []EntityInput `json:"entities" description:"The whole tree in one list, a parent before the children that name it; the batch is written or refused as one"`
	Source   string        `json:"source,omitempty" enum:"import,user,ai" description:"Who asked for the entities; leave it out and a batch you propose is recorded as ai"`
}

type CreateEntitiesResponse struct {
	Entities []Entity `json:"entities"`
	Edges    []Edge   `json:"edges"`
}

type UpdateEntityRequest struct {
	ID                string    `json:"id" description:"The id of the entity, exactly as a read tool returned it"`
	Name              *string   `json:"name,omitempty" description:"The new name, left out to keep the current one"`
	Kind              *string   `json:"kind,omitempty" enum:"hub,product,topic,category,custom" description:"The new kind, left out to keep the current one"`
	Intent            *string   `json:"intent,omitempty" description:"The new reader intent, left out to keep the current one"`
	PrimaryKeyword    *string   `json:"primaryKeyword,omitempty" description:"The new primary keyword, left out to keep the current one"`
	SecondaryKeywords *[]string `json:"secondaryKeywords,omitempty" description:"The whole new list of secondary keywords, left out to keep the current one"`
}

type UpdateEntityResponse struct {
	Entity Entity `json:"entity"`
}

type DeleteEntityRequest struct {
	ID string `json:"id" description:"The id of the entity to remove, exactly as a read tool returned it"`
}

type DeleteEntityResponse struct{}

type GetEntityRequest struct {
	ID string `json:"id" description:"The id of the entity, exactly as a read tool returned it"`
}

type GetEntityResponse struct {
	Entity Entity `json:"entity"`
}

type ListEntitiesRequest struct {
	dto.ListRequest
	SiteID           string `json:"siteId"`
	Kind             string `json:"kind,omitempty" enum:"hub,product,topic,category,custom" description:"Keep only entities of this kind"`
	HasCanonicalPage *bool  `json:"hasCanonicalPage,omitempty" description:"Keep only entities that do, or do not, own a page"`
	NamePrefix       string `json:"namePrefix,omitempty" description:"Keep only entities whose name starts with this text"`
}

type SetAnchorsRequest struct {
	EntityID string   `json:"entityId" description:"The id of the entity, exactly as a read tool returned it"`
	Anchors  []Anchor `json:"anchors,omitempty" description:"The whole new list of anchors, which replaces the current one; leave it out to clear every anchor the entity has"`
}

type SetAnchorsResponse struct {
	Entity Entity `json:"entity"`
}

type AddEdgeRequest struct {
	SiteID       string  `json:"siteId"`
	FromEntityID string  `json:"fromEntityId" description:"The id of the entity the edge starts at; for a parent edge this is the child"`
	ToEntityID   string  `json:"toEntityId" description:"The id of the entity the edge points at; for a parent edge this is the parent"`
	Kind         string  `json:"kind" enum:"parent,related" description:"A parent edge builds the tree and must not make a cycle; a related edge is an undirected sibling link"`
	Weight       float64 `json:"weight,omitempty" minimum:"0" maximum:"1" description:"How close the two are, between 0 and 1; a parent edge takes 1 and leaving it out records no closeness at all"`
	Source       string  `json:"source,omitempty" enum:"import,user,ai" description:"Who asked for the edge; leave it out and it is recorded as user"`
	Status       string  `json:"status,omitempty" enum:"approved,proposed,rejected" description:"Leave it out to add the edge approved; propose it instead to leave the decision to the user"`
	Reason       string  `json:"reason,omitempty" description:"Why the two belong together, one short sentence"`
}

type AddEdgeResponse struct {
	Edge Edge `json:"edge"`
}

type ApproveEdgeRequest struct {
	ID string `json:"id" description:"The id of the proposed edge, exactly as a read tool returned it"`
}

type ApproveEdgeResponse struct {
	Edge Edge `json:"edge"`
}

type RejectEdgeRequest struct {
	ID string `json:"id" description:"The id of the proposed edge, exactly as a read tool returned it"`
}

type RejectEdgeResponse struct {
	Edge Edge `json:"edge"`
}

type DeleteEdgeRequest struct {
	ID string `json:"id" description:"The id of the edge to remove, exactly as a read tool returned it"`
}

type DeleteEdgeResponse struct{}

type ListEdgesRequest struct {
	dto.ListRequest
	SiteID   string `json:"siteId"`
	Kind     string `json:"kind,omitempty" enum:"parent,related" description:"Keep only edges of this kind"`
	Status   string `json:"status,omitempty" enum:"approved,proposed,rejected" description:"Keep only edges in this state"`
	EntityID string `json:"entityId,omitempty" description:"Keep only edges that touch this entity, at either end"`
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
