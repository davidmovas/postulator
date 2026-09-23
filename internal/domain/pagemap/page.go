package pagemap

import (
	"slices"
	"strings"
	"time"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type WPType string

const (
	WPPage            WPType = "page"
	WPPost            WPType = "post"
	WPProduct         WPType = "product"
	WPProductCategory WPType = "product_cat"
)

func (t WPType) Valid() bool {
	switch t {
	case WPPage, WPPost, WPProduct, WPProductCategory:
		return true
	default:
		return false
	}
}

type Status string

const (
	StatusPlanned   Status = "planned"
	StatusExists    Status = "exists"
	StatusPublished Status = "published"
	StatusArchived  Status = "archived"
)

func (s Status) Valid() bool {
	switch s {
	case StatusPlanned, StatusExists, StatusPublished, StatusArchived:
		return true
	default:
		return false
	}
}

type LinkOrigin string

const (
	OriginGenerated LinkOrigin = "generated"
	OriginObserved  LinkOrigin = "observed"
)

func (o LinkOrigin) Valid() bool {
	switch o {
	case OriginGenerated, OriginObserved:
		return true
	default:
		return false
	}
}

type Page struct {
	ID              string
	SiteID          string
	Path            string
	Slug            string
	ParentPageID    *string
	WPType          WPType
	WPID            *int64
	Title           string
	H1              string
	MetaTitle       string
	MetaDescription string
	Canonical       string
	Status          Status
	EntityID        *string
	TemplateID      *string
	ContentHash     string
	Observed        Observed
	WPModifiedAt    *time.Time
	LastSyncedAt    *time.Time
	Drift           bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type PageLink struct {
	ID         string
	SiteID     string
	FromPageID string
	ToPageID   *string
	ToURL      string
	AnchorText string
	Origin     LinkOrigin
	ObservedAt time.Time
}

func invalid(message, field string) *errors.Error {
	return errors.New(errors.Invalid, message).WithDetail("field", field)
}

func emptyRef(ref *string) bool {
	return ref != nil && *ref == ""
}

func NewPage(p Page) (Page, error) {
	switch {
	case p.ID == "":
		return Page{}, invalid("page id must not be empty", "id")
	case p.SiteID == "":
		return Page{}, invalid("page site id must not be empty", "siteId")
	case !p.WPType.Valid():
		return Page{}, invalid("page wordpress type is not recognized", "wpType")
	case !p.Status.Valid():
		return Page{}, invalid("page status is not recognized", "status")
	case emptyRef(p.ParentPageID):
		return Page{}, invalid("parent page id must not be empty when set", "parentPageId")
	case p.ParentPageID != nil && *p.ParentPageID == p.ID:
		return Page{}, invalid("page cannot be its own parent", "parentPageId")
	case emptyRef(p.EntityID):
		return Page{}, invalid("entity id must not be empty when set", "entityId")
	case emptyRef(p.TemplateID):
		return Page{}, invalid("template id must not be empty when set", "templateId")
	}

	path, err := NormalizePath(p.Path)
	if err != nil {
		return Page{}, err
	}
	p.Path = path
	p.Slug = Slug(path)
	p.Title = strings.TrimSpace(p.Title)
	p.H1 = strings.TrimSpace(p.H1)
	p.MetaTitle = strings.TrimSpace(p.MetaTitle)
	p.MetaDescription = strings.TrimSpace(p.MetaDescription)
	p.Canonical = strings.TrimSpace(p.Canonical)
	return p, nil
}

func NewPageLink(l PageLink) (PageLink, error) {
	l.ToURL = strings.TrimSpace(l.ToURL)
	l.AnchorText = strings.TrimSpace(l.AnchorText)
	switch {
	case l.ID == "":
		return PageLink{}, invalid("link id must not be empty", "id")
	case l.SiteID == "":
		return PageLink{}, invalid("link site id must not be empty", "siteId")
	case l.FromPageID == "":
		return PageLink{}, invalid("link source page must not be empty", "fromPageId")
	case emptyRef(l.ToPageID):
		return PageLink{}, invalid("target page id must not be empty when set", "toPageId")
	case l.ToURL == "":
		return PageLink{}, invalid("link needs a target url, whether or not it names a target page", "toUrl")
	case !l.Origin.Valid():
		return PageLink{}, invalid("link origin is not recognized", "origin")
	}
	return l, nil
}

func byPath(a, b Page) int {
	if c := strings.Compare(a.Path, b.Path); c != 0 {
		return c
	}
	return strings.Compare(a.ID, b.ID)
}

func Unmapped(pages []Page) []Page {
	out := make([]Page, 0, len(pages))
	for i := range pages {
		if pages[i].EntityID == nil {
			out = append(out, pages[i])
		}
	}
	slices.SortFunc(out, byPath)
	return out
}
