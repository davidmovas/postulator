package graph

type ProposeFromPagesRequest struct {
	SiteID     string   `json:"siteId"`
	PageIDs    []string `json:"pageIds,omitempty" description:"Propose only for these pages, exactly as pages_list returned their ids; leave it out for every page that carries no entity"`
	PathPrefix string   `json:"pathPrefix,omitempty" description:"Propose only for pages whose path starts with this text, such as one section; leave it out for the whole site"`
}

type ProposeFromPagesResponse struct {
	Entities []Entity `json:"entities"`
	Edges    []Edge   `json:"edges"`
	Skipped  int      `json:"skipped"`
	Tokens   int      `json:"tokens"`
}

type PreviewFromPagesRequest struct {
	SiteID     string   `json:"siteId"`
	PageIDs    []string `json:"pageIds,omitempty" description:"Preview only for these pages, exactly as pages_list returned their ids; leave it out for every page that carries no entity"`
	PathPrefix string   `json:"pathPrefix,omitempty" description:"Preview only for pages whose path starts with this text, such as one section; leave it out for the whole site"`
}

type ProposedEntity struct {
	PageID            string   `json:"pageId,omitempty" description:"The page it is about, as previewed; empty for one from a keyword"`
	Path              string   `json:"path,omitempty" description:"The path of that page"`
	Name              string   `json:"name" description:"The name, two to four words"`
	Kind              string   `json:"kind,omitempty" enum:"hub,product,topic,category,custom" description:"What it is; a topic when left out"`
	Intent            string   `json:"intent,omitempty" description:"What a reader wants from it"`
	PrimaryKeyword    string   `json:"primaryKeyword,omitempty" description:"The phrase a reader searches for"`
	SecondaryKeywords []string `json:"secondaryKeywords,omitempty" description:"Further phrases it can rank for"`
	Anchors           []string `json:"anchors,omitempty" description:"Link texts another page may use"`
	Parent            string   `json:"parent,omitempty" description:"The parent page path or entity name"`
	Related           []string `json:"related,omitempty" description:"Sibling page paths or entity names"`
	ExistingEntityID  string   `json:"existingEntityId,omitempty" description:"The entity of that name that already exists"`
}

type PreviewFromPagesResponse struct {
	Entities []ProposedEntity `json:"entities"`
	Pages    int              `json:"pages"`
	Skipped  int              `json:"skipped"`
	Tokens   int              `json:"tokens"`
}

type ProposeFromKeywordsRequest struct {
	SiteID         string   `json:"siteId"`
	Keywords       []string `json:"keywords" description:"The search keywords to turn into entities, one phrase each"`
	ParentEntityID string   `json:"parentEntityId,omitempty" description:"The entity every proposed one sits under unless the model says otherwise; leave it out for none"`
}

type ProposeFromKeywordsResponse struct {
	Entities []ProposedEntity `json:"entities"`
	Skipped  int              `json:"skipped"`
	Tokens   int              `json:"tokens"`
}

type ApplyProposalsRequest struct {
	SiteID   string           `json:"siteId"`
	Entities []ProposedEntity `json:"entities" description:"The proposals to write, as a preview answered them, without the ones the person declined"`
}

type ApplyProposalsResponse struct {
	Entities []Entity `json:"entities"`
	Edges    []Edge   `json:"edges"`
	Mapped   int      `json:"mapped"`
	Skipped  int      `json:"skipped"`
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
