package pages

import "github.com/davidmovas/postulator/internal/kernel/dto"

type CreateRequest struct {
	SiteID          string  `json:"siteId"`
	Path            string  `json:"path"`
	WPType          string  `json:"wpType,omitempty"`
	Title           string  `json:"title"`
	H1              string  `json:"h1"`
	MetaTitle       string  `json:"metaTitle"`
	MetaDescription string  `json:"metaDescription"`
	Canonical       string  `json:"canonical"`
	Status          string  `json:"status,omitempty"`
	EntityID        *string `json:"entityId,omitempty"`
	TemplateID      *string `json:"templateId,omitempty"`
}

type CreateResponse struct {
	Page Page `json:"page"`
}

type UpdateRequest struct {
	ID              string  `json:"id"`
	Path            *string `json:"path,omitempty"`
	WPType          *string `json:"wpType,omitempty"`
	Title           *string `json:"title,omitempty"`
	H1              *string `json:"h1,omitempty"`
	MetaTitle       *string `json:"metaTitle,omitempty"`
	MetaDescription *string `json:"metaDescription,omitempty"`
	Canonical       *string `json:"canonical,omitempty"`
	Status          *string `json:"status,omitempty"`
	TemplateID      *string `json:"templateId,omitempty"`
}

type UpdateResponse struct {
	Page Page `json:"page"`
}

type DeleteRequest struct {
	ID string `json:"id"`
}

type DeleteResponse struct{}

type GetRequest struct {
	ID string `json:"id"`
}

type GetResponse struct {
	Page  Page       `json:"page"`
	Links []PageLink `json:"links"`
}

type ListRequest struct {
	dto.ListRequest
	SiteID     string `json:"siteId"`
	Status     string `json:"status,omitempty"`
	EntityID   string `json:"entityId,omitempty"`
	Unmapped   bool   `json:"unmapped,omitempty"`
	PathPrefix string `json:"pathPrefix,omitempty"`
}

type MapToEntityRequest struct {
	PageID   string `json:"pageId"`
	EntityID string `json:"entityId"`
}

type MapToEntityResponse struct {
	Page Page `json:"page"`
}

type UnmapRequest struct {
	PageID string `json:"pageId"`
}

type UnmapResponse struct {
	Page Page `json:"page"`
}

type SetCanonicalRequest struct {
	EntityID string `json:"entityId"`
	PageID   string `json:"pageId"`
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
	ToPageID   *string `json:"toPageId,omitempty"`
	ToURL      string  `json:"toUrl"`
	AnchorText string  `json:"anchorText"`
	Origin     string  `json:"origin,omitempty"`
}

type ReplaceLinksRequest struct {
	PageID string      `json:"pageId"`
	Links  []LinkInput `json:"links"`
}

type ReplaceLinksResponse struct {
	Links []PageLink `json:"links"`
}

type PreviewLinkRequest struct {
	PageID string `json:"pageId"`
}

type PreviewLinkResponse struct {
	URL       string   `json:"url"`
	ExpiresAt dto.Time `json:"expiresAt"`
	Kind      string   `json:"kind"`
}
