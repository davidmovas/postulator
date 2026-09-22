package pages

import "github.com/davidmovas/postulator/internal/kernel/dto"

type CreateRequest struct {
	SiteID          string  `json:"siteId"`
	Path            string  `json:"path" description:"Where the page lives on the site, a leading and trailing slash, for example /supplements/creatine/"`
	WPType          string  `json:"wpType,omitempty" enum:"page,post,product,product_cat" description:"What WordPress calls the record; leave it out for a page"`
	Title           string  `json:"title" description:"The title the page carries in WordPress"`
	H1              string  `json:"h1,omitempty" description:"The heading the page opens with, left out to follow the title"`
	MetaTitle       string  `json:"metaTitle,omitempty" description:"The SEO title, left out to follow the template"`
	MetaDescription string  `json:"metaDescription,omitempty" description:"The SEO description, left out to follow the template"`
	Canonical       string  `json:"canonical,omitempty" description:"The address this page should be indexed under, left out for its own"`
	Status          string  `json:"status,omitempty" enum:"planned,exists,published,archived" description:"Where the page is in its life; leave it out and it is planned"`
	EntityID        *string `json:"entityId,omitempty" description:"The entity this page is about, exactly as a read tool returned its id"`
	TemplateID      *string `json:"templateId,omitempty" description:"The template that writes this page, left out to take the site default"`
}

type CreateResponse struct {
	Page Page `json:"page"`
}

type UpdateRequest struct {
	ID              string  `json:"id" description:"The id of the page, exactly as a read tool returned it"`
	Path            *string `json:"path,omitempty" description:"The new path, left out to keep the current one"`
	WPType          *string `json:"wpType,omitempty" enum:"page,post,product,product_cat" description:"The new WordPress type, left out to keep the current one"`
	Title           *string `json:"title,omitempty" description:"The new title, left out to keep the current one"`
	H1              *string `json:"h1,omitempty" description:"The new heading, left out to keep the current one"`
	MetaTitle       *string `json:"metaTitle,omitempty" description:"The new SEO title, left out to keep the current one"`
	MetaDescription *string `json:"metaDescription,omitempty" description:"The new SEO description, left out to keep the current one"`
	Canonical       *string `json:"canonical,omitempty" description:"The new canonical address, left out to keep the current one"`
	Status          *string `json:"status,omitempty" enum:"planned,exists,published,archived" description:"The new status, left out to keep the current one"`
	TemplateID      *string `json:"templateId,omitempty" description:"The new template, left out to keep the current one"`
}

type UpdateResponse struct {
	Page Page `json:"page"`
}

type DeleteRequest struct {
	ID     string `json:"id" description:"The id of the page to remove from the map, exactly as a read tool returned it"`
	OnSite bool   `json:"onSite,omitempty" description:"Also move the page to the WordPress trash, where a human can still restore it"`
}

type DeleteResponse struct{}

type GetRequest struct {
	ID string `json:"id" description:"The id of the page, exactly as a read tool returned it"`
}

type GetResponse struct {
	Page  Page       `json:"page"`
	Links []PageLink `json:"links"`
}

type ListRequest struct {
	dto.ListRequest
	SiteID     string `json:"siteId"`
	Status     string `json:"status,omitempty" enum:"planned,exists,published,archived" description:"Keep only pages in this state"`
	EntityID   string `json:"entityId,omitempty" description:"Keep only pages mapped to this entity"`
	Unmapped   bool   `json:"unmapped,omitempty" description:"Keep only pages that carry no entity"`
	PathPrefix string `json:"pathPrefix,omitempty" description:"Keep only pages whose path starts with this text"`
}

type MapToEntityRequest struct {
	PageID   string `json:"pageId" description:"The id of the page, exactly as a read tool returned it"`
	EntityID string `json:"entityId" description:"The id of the entity the page is about, exactly as a read tool returned it"`
}

type MapToEntityResponse struct {
	Page Page `json:"page"`
}

type UnmapRequest struct {
	PageID string `json:"pageId" description:"The id of the page to detach from its entity, exactly as a read tool returned it"`
}

type UnmapResponse struct {
	Page Page `json:"page"`
}

type SetCanonicalRequest struct {
	EntityID string `json:"entityId" description:"The id of the entity, exactly as a read tool returned it"`
	PageID   string `json:"pageId" description:"The id of the page that becomes the entity's canonical one"`
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
	ToPageID   *string `json:"toPageId,omitempty" description:"The id of the page this link points at, left out for a link that leaves the site"`
	ToURL      string  `json:"toUrl" description:"The address the link points at"`
	AnchorText string  `json:"anchorText" description:"The text the link is written as"`
	Origin     string  `json:"origin,omitempty" enum:"generated,observed" description:"Whether Postulator wrote the link or read it off the live page; leave it out for generated"`
}

type ReplaceLinksRequest struct {
	PageID string      `json:"pageId" description:"The id of the page whose links are replaced, exactly as a read tool returned it"`
	Links  []LinkInput `json:"links" description:"The whole new list of links, which replaces the current one"`
}

type ReplaceLinksResponse struct {
	Links []PageLink `json:"links"`
}

type PreviewLinkRequest struct {
	PageID string `json:"pageId" description:"The id of the page to look at, exactly as a read tool returned it"`
}

type PreviewLinkResponse struct {
	URL       string   `json:"url"`
	ExpiresAt dto.Time `json:"expiresAt"`
	Kind      string   `json:"kind"`
}
