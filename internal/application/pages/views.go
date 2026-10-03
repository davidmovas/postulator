package pages

import (
	"time"

	"github.com/davidmovas/postulator/internal/application"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/kernel/dto"
)

type Observed struct {
	Link   string `json:"link"`
	Slug   string `json:"slug"`
	Status string `json:"status"`
	Title  string `json:"title"`
	H1     string `json:"h1"`
}

type Mismatch struct {
	Field   string `json:"field"`
	Planned string `json:"planned"`
	Actual  string `json:"actual"`
}

type Note struct {
	Label string `json:"label"`
	Text  string `json:"text"`
}

type Page struct {
	ID                   string         `json:"id"`
	SiteID               string         `json:"siteId"`
	Path                 string         `json:"path"`
	PlannedPath          string         `json:"plannedPath"`
	Slug                 string         `json:"slug"`
	ParentPageID         *string        `json:"parentPageId"`
	WPType               string         `json:"wpType"`
	WPID                 *int64         `json:"wpId"`
	Title                string         `json:"title"`
	H1                   string         `json:"h1"`
	MetaTitle            string         `json:"metaTitle"`
	MetaDescription      string         `json:"metaDescription"`
	Canonical            string         `json:"canonical"`
	Keywords             []dto.Keyword  `json:"keywords"`
	Notes                []Note         `json:"notes"`
	Status               string         `json:"status"`
	EntityID             *string        `json:"entityId"`
	Categories           []dto.Category `json:"categories"`
	CategoriesNeedPlugin bool           `json:"categoriesNeedPlugin"`
	TemplateID           *string        `json:"templateId"`
	ContentHash          string         `json:"contentHash"`
	Observed             Observed       `json:"observed"`
	Mismatches           []Mismatch     `json:"mismatches"`
	WPModifiedAt         dto.Time       `json:"wpModifiedAt"`
	LastSyncedAt         dto.Time       `json:"lastSyncedAt"`
	Drift                bool           `json:"drift"`
	CreatedAt            dto.Time       `json:"createdAt"`
	UpdatedAt            dto.Time       `json:"updatedAt"`
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

func view(p pagemap.Page, filed filing) Page {
	categories, needPlugin := filed.of(p)
	return Page{
		ID:                   p.ID,
		SiteID:               p.SiteID,
		Path:                 p.Path,
		PlannedPath:          p.PlannedPath,
		Slug:                 p.Slug,
		ParentPageID:         p.ParentPageID,
		WPType:               string(p.WPType),
		WPID:                 p.WPID,
		Title:                p.Title,
		H1:                   p.H1,
		MetaTitle:            p.MetaTitle,
		MetaDescription:      p.MetaDescription,
		Keywords:             application.KeywordViews(p.Keywords),
		Notes:                noteViews(p.Notes),
		Canonical:            p.Canonical,
		Status:               string(p.Status),
		EntityID:             p.EntityID,
		Categories:           categories,
		CategoriesNeedPlugin: needPlugin,
		TemplateID:           p.TemplateID,
		ContentHash:          p.ContentHash,
		Observed: Observed{
			Link: p.Observed.Link, Slug: p.Observed.Slug, Status: p.Observed.Status,
			Title: p.Observed.Title, H1: p.Observed.H1,
		},
		Mismatches:   mismatchViews(p),
		WPModifiedAt: optionalTime(p.WPModifiedAt),
		LastSyncedAt: optionalTime(p.LastSyncedAt),
		Drift:        p.Drift,
		CreatedAt:    dto.NewTime(p.CreatedAt),
		UpdatedAt:    dto.NewTime(p.UpdatedAt),
	}
}

func noteViews(notes []pagemap.Note) []Note {
	out := make([]Note, 0, len(notes))
	for _, note := range notes {
		out = append(out, Note{Label: note.Label, Text: note.Text})
	}
	return out
}

func mismatchViews(p pagemap.Page) []Mismatch {
	found := p.Mismatches()
	out := make([]Mismatch, 0, len(found))
	for i := range found {
		out = append(out, Mismatch{Field: found[i].Field, Planned: found[i].Planned, Actual: found[i].Actual})
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

func nodeViews(nodes []pagemap.Node, filed filing) []TreeNode {
	out := make([]TreeNode, 0, len(nodes))
	for i := range nodes {
		out = append(out, TreeNode{Page: view(nodes[i].Page, filed), Children: nodeViews(nodes[i].Children, filed)})
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
