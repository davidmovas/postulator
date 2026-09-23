package graph

type ProposeFromPagesRequest struct {
	SiteID string `json:"siteId"`
}

type ProposeFromPagesResponse struct {
	Entities []Entity `json:"entities"`
	Edges    []Edge   `json:"edges"`
	Skipped  int      `json:"skipped"`
	Tokens   int      `json:"tokens"`
}

type ProposeRelatedRequest struct {
	SiteID   string `json:"siteId"`
	EntityID string `json:"entityId,omitempty" description:"propose only pairs that involve this entity when it is given"`
}

type ProposeRelatedResponse struct {
	Edges   []Edge `json:"edges"`
	Skipped int    `json:"skipped"`
	Tokens  int    `json:"tokens"`
}
