package pages

import "github.com/davidmovas/postulator/internal/kernel/dto"

type CreateRequest struct {
	SiteID          string        `json:"siteId"`
	Path            string        `json:"path" description:"Site path with both slashes, e.g. /supplements/creatine/"`
	WPType          string        `json:"wpType,omitempty" enum:"page,post,product,product_cat" description:"WordPress type; default page"`
	Title           string        `json:"title,omitempty" description:"WordPress title"`
	H1              string        `json:"h1,omitempty" description:"H1; default the title"`
	MetaTitle       string        `json:"metaTitle,omitempty" description:"SEO title; default from the template"`
	MetaDescription string        `json:"metaDescription,omitempty" description:"SEO description; default from the template"`
	Canonical       string        `json:"canonical,omitempty" description:"Canonical URL; default its own"`
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
	Canonical       *string        `json:"canonical,omitempty" description:"New canonical URL"`
	Keywords        *[]dto.Keyword `json:"keywords,omitempty" description:"Whole new keyword list; empty follows its entity"`
	Status          *string        `json:"status,omitempty" enum:"planned,exists,published,archived" description:"New status"`
	TemplateID      *string        `json:"templateId,omitempty" description:"New template id"`
}

type UpdateResponse struct {
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

type ListRequest struct {
	dto.ListRequest
	SiteID             string `json:"siteId"`
	Status             string `json:"status,omitempty" enum:"planned,exists,published,archived" description:"Only this state"`
	EntityID           string `json:"entityId,omitempty" description:"Only pages of this entity"`
	IncludeDescendants bool   `json:"includeDescendants,omitempty" description:"With entityId, also the entities below it"`
	Unmapped           bool   `json:"unmapped,omitempty" description:"Only pages with no entity"`
	PathPrefix         string `json:"pathPrefix,omitempty" description:"Only paths starting with this"`
	CategoryID         string `json:"categoryId,omitempty" description:"Only this category and those below it"`
}

type ListCategoriesRequest struct {
	SiteID string `json:"siteId"`
}

type ListCategoriesResponse struct {
	Categories []CategoryNode `json:"categories"`
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

type LinkInput struct {
	ToPageID   *string `json:"toPageId,omitempty" description:"Target page id; omit for an external link"`
	ToURL      string  `json:"toUrl" description:"Link address"`
	AnchorText string  `json:"anchorText" description:"Link text"`
	Origin     string  `json:"origin,omitempty" enum:"generated,observed" description:"Default generated; observed is read off the live page"`
}

type ReplaceLinksRequest struct {
	PageID string      `json:"pageId" description:"Page id"`
	Links  []LinkInput `json:"links,omitempty" description:"Whole new list; omit to clear all"`
}

type ReplaceLinksResponse struct {
	Links []PageLink `json:"links"`
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
