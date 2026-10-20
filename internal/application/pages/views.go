package pages

import (
	"github.com/davidmovas/postulator/internal/application"
	"github.com/davidmovas/postulator/internal/domain/note"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/kernel/dto"
)

type Note struct {
	Label string `json:"label"`
	Text  string `json:"text"`
}

type Page struct {
	ID              string        `json:"id"`
	SiteID          string        `json:"siteId"`
	Path            string        `json:"path"`
	PlannedPath     string        `json:"plannedPath"`
	Slug            string        `json:"slug"`
	ParentPageID    *string       `json:"parentPageId"`
	WPType          string        `json:"wpType"`
	WPID            *int64        `json:"wpId"`
	Title           string        `json:"title"`
	H1              string        `json:"h1"`
	MetaTitle       string        `json:"metaTitle"`
	MetaDescription string        `json:"metaDescription"`
	Canonical       string        `json:"canonical"`
	Keywords        []dto.Keyword `json:"keywords"`
	Notes           []Note        `json:"notes"`
	Status          string        `json:"status"`
	EntityID        *string       `json:"entityId"`
	TemplateID      *string       `json:"templateId"`
	PageTypeID      *string       `json:"pageTypeId"`
	TypeSource      string        `json:"typeSource"`
	TypeReason      string        `json:"typeReason"`
	ContentHash     string        `json:"contentHash"`
	WPModifiedAt    dto.Time      `json:"wpModifiedAt"`
	LastSyncedAt    dto.Time      `json:"lastSyncedAt"`
	Drift           bool          `json:"drift"`
	CreatedAt       dto.Time      `json:"createdAt"`
	UpdatedAt       dto.Time      `json:"updatedAt"`
}

type PageLink struct {
	ID         string   `json:"id"`
	SiteID     string   `json:"siteId"`
	FromPageID string   `json:"fromPageId"`
	ToPageID   *string  `json:"toPageId"`
	ToURL      string   `json:"toUrl"`
	AnchorText string   `json:"anchorText"`
	Origin     string   `json:"origin"`
	Type       string   `json:"type"`
	Region     string   `json:"region"`
	Source     string   `json:"source"`
	Position   *int     `json:"position"`
	Ours       bool     `json:"ours"`
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

func view(p pagemap.Page) Page {
	return Page{
		ID:              p.ID,
		SiteID:          p.SiteID,
		Path:            p.Path,
		PlannedPath:     p.PlannedPath,
		Slug:            p.Slug,
		ParentPageID:    p.ParentPageID,
		WPType:          string(p.WPType),
		WPID:            p.WPID,
		Title:           p.Title,
		H1:              p.H1,
		MetaTitle:       p.MetaTitle,
		MetaDescription: p.MetaDescription,
		Keywords:        application.KeywordViews(p.Keywords),
		Notes:           noteViews(p.Notes),
		Canonical:       p.Canonical,
		Status:          string(p.Status),
		EntityID:        p.EntityID,
		TemplateID:      p.TemplateID,
		PageTypeID:      p.PageTypeID,
		TypeSource:      string(p.TypeSource),
		TypeReason:      string(p.TypeReason),
		ContentHash:     p.ContentHash,
		WPModifiedAt:    dto.TimeOf(p.WPModifiedAt),
		LastSyncedAt:    dto.TimeOf(p.LastSyncedAt),
		Drift:           p.Drift,
		CreatedAt:       dto.NewTime(p.CreatedAt),
		UpdatedAt:       dto.NewTime(p.UpdatedAt),
	}
}

func noteViews(notes []note.Note) []Note {
	out := make([]Note, 0, len(notes))
	for _, n := range notes {
		out = append(out, Note{Label: n.Label, Text: n.Text})
	}
	return out
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
		Type:       string(l.Type),
		Region:     l.Region,
		Source:     string(l.Source),
		Position:   l.Position,
		Ours:       l.Ours,
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
