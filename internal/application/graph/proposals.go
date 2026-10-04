package graph

import "github.com/davidmovas/postulator/internal/kernel/dto"

type ProposeFromPagesRequest struct {
	SiteID     string   `json:"siteId"`
	PageIDs    []string `json:"pageIds,omitempty" description:"Only these page ids; default every unmapped page"`
	PathPrefix string   `json:"pathPrefix,omitempty" description:"Only paths starting with this; default the whole site"`
}

type ProposeFromPagesResponse struct {
	Entities []Entity `json:"entities"`
	Edges    []Edge   `json:"edges"`
	Skipped  int      `json:"skipped"`
	Tokens   int      `json:"tokens"`
}

type PreviewFromPagesRequest struct {
	SiteID     string   `json:"siteId"`
	PageIDs    []string `json:"pageIds,omitempty" description:"Only these page ids; default every unmapped page"`
	PathPrefix string   `json:"pathPrefix,omitempty" description:"Only paths starting with this; default the whole site"`
}

type ProposedEntity struct {
	PageID           string        `json:"pageId,omitempty" description:"Page id as previewed; empty for a keyword"`
	Path             string        `json:"path,omitempty" description:"That page's path"`
	Name             string        `json:"name" description:"Name, 2-4 words"`
	Kind             string        `json:"kind,omitempty" enum:"hub,product,topic,category,custom" description:"Kind; default topic"`
	Intent           string        `json:"intent,omitempty" description:"Reader's goal"`
	Keywords         []dto.Keyword `json:"keywords,omitempty" description:"Search phrases, main first"`
	Anchors          []string      `json:"anchors,omitempty" description:"Link texts other pages may use"`
	Parent           string        `json:"parent,omitempty" description:"Parent path or entity name"`
	Related          []string      `json:"related,omitempty" description:"Sibling paths or entity names"`
	ExistingEntityID string        `json:"existingEntityId,omitempty" description:"Existing entity of that name"`
}

type PreviewFromPagesResponse struct {
	Entities []ProposedEntity `json:"entities"`
	Pages    int              `json:"pages"`
	Skipped  int              `json:"skipped"`
	Tokens   int              `json:"tokens"`
}

type ProposeFromKeywordsRequest struct {
	SiteID         string   `json:"siteId"`
	Keywords       []string `json:"keywords" description:"One phrase each, volume optional in brackets: bpc 157 (1200)"`
	ParentEntityID string   `json:"parentEntityId,omitempty" description:"Default parent entity id of the proposals"`
}

type ProposeFromKeywordsResponse struct {
	Entities []ProposedEntity `json:"entities"`
	Skipped  int              `json:"skipped"`
	Tokens   int              `json:"tokens"`
}

type ApplyProposalsRequest struct {
	SiteID   string           `json:"siteId"`
	Entities []ProposedEntity `json:"entities" description:"Proposals as previewed, minus declined ones"`
}

type ApplyProposalsResponse struct {
	Entities []Entity `json:"entities"`
	Edges    []Edge   `json:"edges"`
	Mapped   int      `json:"mapped"`
	Skipped  int      `json:"skipped"`
}

type ProposeRelatedRequest struct {
	SiteID   string `json:"siteId"`
	EntityID string `json:"entityId,omitempty" description:"Only pairs with this entity"`
}

type ProposeRelatedResponse struct {
	Edges   []Edge `json:"edges"`
	Skipped int    `json:"skipped"`
	Tokens  int    `json:"tokens"`
}
