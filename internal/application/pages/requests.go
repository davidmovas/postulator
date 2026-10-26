package pages

import (
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/kernel/dto"
)

type CreateRequest struct {
	SiteID          string        `json:"siteId"`
	Path            string        `json:"path" description:"Site path with both slashes, e.g. /supplements/creatine/"`
	WPType          string        `json:"wpType,omitempty" enum:"page,post,product,product_cat" description:"WordPress type; default page"`
	Title           string        `json:"title,omitempty" description:"WordPress title"`
	H1              string        `json:"h1,omitempty" description:"H1; default the title"`
	MetaTitle       string        `json:"metaTitle,omitempty" description:"SEO title; default from the template"`
	MetaDescription string        `json:"metaDescription,omitempty" description:"SEO description; default from the template"`
	Keywords        []dto.Keyword `json:"keywords,omitempty" description:"Search phrases; default its entity's"`
	Status          string        `json:"status,omitempty" enum:"planned,exists,published,archived" description:"Default planned"`
	EntityID        *string       `json:"entityId,omitempty" description:"Id of the entity it is about"`
	TemplateID      *string       `json:"templateId,omitempty" description:"Template id; default the site's"`
}

type CreateResponse struct {
	Page Page `json:"page"`
}

type UpdateRequest struct {
	ID              string         `json:"id" description:"Page id"`
	Path            *string        `json:"path,omitempty" description:"New path"`
	WPType          *string        `json:"wpType,omitempty" enum:"page,post,product,product_cat" description:"New WordPress type"`
	Title           *string        `json:"title,omitempty" description:"New title"`
	H1              *string        `json:"h1,omitempty" description:"New H1"`
	MetaTitle       *string        `json:"metaTitle,omitempty" description:"New SEO title"`
	MetaDescription *string        `json:"metaDescription,omitempty" description:"New SEO description"`
	Keywords        *[]dto.Keyword `json:"keywords,omitempty" description:"Whole new keyword list; empty follows its entity"`
	Status          *string        `json:"status,omitempty" enum:"planned,exists,published,archived" description:"New status"`
	TemplateID      *string        `json:"templateId,omitempty" description:"New template id"`
}

type UpdateResponse struct {
	Page Page `json:"page"`
}

type SetTypeRequest struct {
	PageID     string             `json:"pageId"`
	PageTypeID string             `json:"pageTypeId"`
	By         pagemap.TypeReason `json:"-"`
}

type SetTypeResponse struct {
	Page Page `json:"page"`
}

type DeleteRequest struct {
	ID     string `json:"id" description:"Page id"`
	OnSite bool   `json:"onSite,omitempty" description:"Also move it to the WordPress trash, still restorable"`
}

type DeleteResponse struct{}

type GetRequest struct {
	ID string `json:"id" description:"Page id"`
}

type GetResponse struct {
	Page  Page       `json:"page"`
	Links []PageLink `json:"links"`
}

type PageFilter struct {
	Status             string `json:"status,omitempty" enum:"planned,exists,published,archived" description:"Only this state"`
	EntityID           string `json:"entityId,omitempty" description:"Only pages of this entity"`
	IncludeDescendants bool   `json:"includeDescendants,omitempty" description:"With entityId, also the entities below it"`
	Unmapped           bool   `json:"unmapped,omitempty" description:"Only pages with no entity"`
	PathPrefix         string `json:"pathPrefix,omitempty" description:"Only paths starting with this"`
	PageTypeID         string `json:"pageTypeId,omitempty" description:"Only pages of this type, from pagetypes_list"`
	Body               string `json:"body,omitempty" description:"with or without text of its own"`
	Findings           bool   `json:"findings,omitempty" description:"Only pages with an open finding"`
	Search             string `json:"search,omitempty" description:"Words in the title or path"`
	Under              string `json:"under,omitempty" description:"Only this page and those under it on the site map"`
}

type ListRequest struct {
	dto.ListRequest
	SiteID string `json:"siteId"`
	PageFilter
}

type CountRequest struct {
	SiteID string `json:"siteId"`
	PageFilter
}

type CountResponse struct {
	Pages int `json:"pages"`
}

type Subtree struct {
	PageID string `json:"pageId"`
	TypeID string `json:"typeId,omitempty"`
}

type Selection struct {
	SiteID         string      `json:"siteId"`
	PageIDs        []string    `json:"pageIds,omitempty"`
	Subtrees       []Subtree   `json:"subtrees,omitempty"`
	Filter         *PageFilter `json:"filter,omitempty"`
	Seen           int         `json:"seen,omitempty"`
	RuleID         string      `json:"ruleId,omitempty"`
	Except         []string    `json:"except,omitempty"`
	ExceptSubtrees []Subtree   `json:"exceptSubtrees,omitempty"`
}

type ResolveResponse struct {
	PageIDs []string `json:"pageIds"`
}

type SetTypesRequest struct {
	Selection  Selection          `json:"selection"`
	PageTypeID *string            `json:"pageTypeId"`
	By         pagemap.TypeReason `json:"-"`
}

type SetTypesResponse struct {
	Typed   int `json:"typed"`
	Refused int `json:"refused"`
}

type SetTemplatesRequest struct {
	Selection  Selection `json:"selection"`
	TemplateID *string   `json:"templateId"`
}

type SetTemplatesResponse struct {
	Changed int `json:"changed"`
}

type MapToEntityRequest struct {
	PageID   string `json:"pageId" description:"Page id"`
	EntityID string `json:"entityId" description:"Id of the entity it is about"`
}

type MapToEntityResponse struct {
	Page Page `json:"page"`
}

type UnmapRequest struct {
	PageID string `json:"pageId" description:"Page id"`
}

type UnmapResponse struct {
	Page Page `json:"page"`
}

type SetCanonicalRequest struct {
	EntityID string `json:"entityId" description:"Entity id"`
	PageID   string `json:"pageId" description:"Id of the page that becomes canonical"`
}

type SetCanonicalResponse struct {
	Page Page `json:"page"`
}

type TreeRequest struct {
	SiteID string `json:"siteId"`
}

type TreeResponse struct {
	Roots []TreeNode `json:"roots"`
}

type PreviewLinkRequest struct {
	PageID string `json:"pageId" description:"Page id"`
}

type PreviewLinkResponse struct {
	URL       string   `json:"url"`
	ExpiresAt dto.Time `json:"expiresAt"`
	Kind      string   `json:"kind"`
}

type AssignTemplateRequest struct {
	SiteID     string   `json:"siteId"`
	PageIDs    []string `json:"pageIds"`
	TemplateID string   `json:"templateId"`
}

type AssignTemplateResponse struct {
	Changed int `json:"changed"`
}
