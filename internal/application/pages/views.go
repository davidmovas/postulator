package pages

import (
	"time"

	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/kernel/dto"
)

type Page struct {
	ID              string   `json:"id"`
	SiteID          string   `json:"siteId"`
	Path            string   `json:"path"`
	Slug            string   `json:"slug"`
	ParentPageID    *string  `json:"parentPageId"`
	WPType          string   `json:"wpType"`
	WPID            *int64   `json:"wpId"`
	Title           string   `json:"title"`
	H1              string   `json:"h1"`
	MetaTitle       string   `json:"metaTitle"`
	MetaDescription string   `json:"metaDescription"`
	Canonical       string   `json:"canonical"`
	Status          string   `json:"status"`
	EntityID        *string  `json:"entityId"`
	TemplateID      *string  `json:"templateId"`
	ContentHash     string   `json:"contentHash"`
	WPModifiedAt    dto.Time `json:"wpModifiedAt"`
	LastSyncedAt    dto.Time `json:"lastSyncedAt"`
	Drift           bool     `json:"drift"`
	CreatedAt       dto.Time `json:"createdAt"`
	UpdatedAt       dto.Time `json:"updatedAt"`
}

type PageLink struct {
	ID         string   `json:"id"`
	SiteID     string   `json:"siteId"`
	FromPageID string   `json:"fromPageId"`
	ToPageID   *string  `json:"toPageId"`
	ToURL      string   `json:"toUrl"`
	AnchorText string   `json:"anchorText"`
	Origin     string   `json:"origin"`
	ObservedAt dto.Time `json:"observedAt"`
}

type TreeNode struct {
	Page     Page       `json:"page"`
	Children []TreeNode `json:"children"`
}

type Conflict struct {
	PageID   string `json:"pageId"`
	Path     string `json:"path"`
	Reason   string `json:"reason"`
	EntityID string `json:"entityId,omitempty"`
}

func optionalTime(t *time.Time) dto.Time {
	if t == nil {
		return dto.Time{}
	}
	return dto.NewTime(*t)
}

func view(p pagemap.Page) Page {
	return Page{
		ID:              p.ID,
		SiteID:          p.SiteID,
		Path:            p.Path,
		Slug:            p.Slug,
		ParentPageID:    p.ParentPageID,
		WPType:          string(p.WPType),
		WPID:            p.WPID,
		Title:           p.Title,
		H1:              p.H1,
		MetaTitle:       p.MetaTitle,
		MetaDescription: p.MetaDescription,
		Canonical:       p.Canonical,
		Status:          string(p.Status),
		EntityID:        p.EntityID,
		TemplateID:      p.TemplateID,
		ContentHash:     p.ContentHash,
		WPModifiedAt:    optionalTime(p.WPModifiedAt),
		LastSyncedAt:    optionalTime(p.LastSyncedAt),
		Drift:           p.Drift,
		CreatedAt:       dto.NewTime(p.CreatedAt),
		UpdatedAt:       dto.NewTime(p.UpdatedAt),
	}
}

func linkView(l pagemap.PageLink) PageLink {
	return PageLink{
		ID:         l.ID,
		SiteID:     l.SiteID,
		FromPageID: l.FromPageID,
		ToPageID:   l.ToPageID,
		ToURL:      l.ToURL,
		AnchorText: l.AnchorText,
		Origin:     string(l.Origin),
		ObservedAt: dto.NewTime(l.ObservedAt),
	}
}

func linkViews(links []pagemap.PageLink) []PageLink {
	out := make([]PageLink, 0, len(links))
	for i := range links {
		out = append(out, linkView(links[i]))
	}
	return out
}

func nodeViews(nodes []pagemap.Node) []TreeNode {
	out := make([]TreeNode, 0, len(nodes))
	for i := range nodes {
		out = append(out, TreeNode{Page: view(nodes[i].Page), Children: nodeViews(nodes[i].Children)})
	}
	return out
}

func conflicts(evidence []pagemap.Evidence) []Conflict {
	out := make([]Conflict, 0, len(evidence))
	for i := range evidence {
		out = append(out, Conflict{PageID: evidence[i].PageID, Path: evidence[i].Path, Reason: string(evidence[i].Reason), EntityID: evidence[i].EntityID})
	}
	return out
}
