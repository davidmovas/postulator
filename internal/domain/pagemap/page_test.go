package pagemap_test

import (
	stderrors "errors"
	"slices"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/keyword"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	siteA = "0b6c2a4e-1f3d-4c8b-9a2e-5d7f8e9a0b1c"
	pageA = "aaaaaaaa-1111-4aaa-8aaa-aaaaaaaaaaaa"
	pageB = "bbbbbbbb-2222-4bbb-8bbb-bbbbbbbbbbbb"
	pageC = "cccccccc-3333-4ccc-8ccc-cccccccccccc"
	pageD = "dddddddd-4444-4ddd-8ddd-dddddddddddd"
	entA  = "1a1a1a1a-1a1a-4a1a-8a1a-1a1a1a1a1a1a"
	entB  = "2b2b2b2b-2b2b-4b2b-8b2b-2b2b2b2b2b2b"
)

var stamp = time.Date(2026, time.September, 18, 9, 0, 0, 0, time.UTC)

func page(id, path string, entityID *string) pagemap.Page {
	return pagemap.Page{ID: id, SiteID: siteA, Path: path, Slug: pagemap.Slug(path), WPType: pagemap.WPPage, Status: pagemap.StatusPlanned, EntityID: entityID, CreatedAt: stamp, UpdatedAt: stamp}
}

func ptr(s string) *string {
	return &s
}

func fieldOf(t *testing.T, err error) string {
	t.Helper()
	var kernel *errors.Error
	if !stderrors.As(err, &kernel) {
		t.Fatalf("error %v is not a kernel error", err)
	}
	field, ok := kernel.Details["field"].(string)
	if !ok {
		t.Fatalf("error %v carries no field detail", err)
	}
	return field
}

func TestNewPageNormalises(t *testing.T) {
	t.Parallel()

	p := page(pageA, "/Shop/Bags", nil)
	p.Title = "  Bags  "
	p.Keywords = keyword.List{{Text: " totes"}, {Text: "Totes"}, {Text: ""}, {Text: "clutches"}, {Text: " leather bags ", Volume: new(700)}}
	p.Notes = []pagemap.Note{
		{Label: " Intent Owner ", Text: " GEO Commercial "}, {Label: "Notes", Text: "  "}, {Label: " ", Text: "no label"},
		{Label: "intent owner", Text: "said twice"}, {Label: "Reason", Text: "one form of the product"},
	}
	got, err := pagemap.NewPage(p)
	if err != nil {
		t.Fatalf("NewPage: %v", err)
	}
	if got.Path != "/shop/bags/" || got.Slug != "bags" || got.Title != "Bags" {
		t.Errorf("NewPage = path %q slug %q title %q", got.Path, got.Slug, got.Title)
	}
	if !slices.Equal(got.Keywords.Texts(), []string{"leather bags", "totes", "clutches"}) {
		t.Errorf("NewPage keywords = %v, want them trimmed, deduplicated and ordered by volume", got.Keywords.Texts())
	}
	wantNotes := []pagemap.Note{{Label: "Intent Owner", Text: "GEO Commercial"}, {Label: "Reason", Text: "one form of the product"}}
	if !slices.Equal(got.Notes, wantNotes) {
		t.Errorf("NewPage notes = %+v, want %+v", got.Notes, wantNotes)
	}
}

func TestNewPageWithoutKeywordsOrNotesCarriesEmptyLists(t *testing.T) {
	t.Parallel()

	got, err := pagemap.NewPage(page(pageA, "/a/", nil))
	if err != nil {
		t.Fatalf("NewPage: %v", err)
	}
	if got.Keywords == nil || len(got.Keywords) != 0 {
		t.Errorf("Keywords = %#v, want an empty list that is not nil", got.Keywords)
	}
	if got.Notes == nil || len(got.Notes) != 0 {
		t.Errorf("Notes = %#v, want an empty list that is not nil", got.Notes)
	}
}

func TestNewPageRejects(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		mutate func(*pagemap.Page)
		field  string
	}{
		{name: "no id", mutate: func(p *pagemap.Page) { p.ID = "" }, field: "id"},
		{name: "no site", mutate: func(p *pagemap.Page) { p.SiteID = "" }, field: "siteId"},
		{name: "bad path", mutate: func(p *pagemap.Page) { p.Path = "/a b/" }, field: "path"},
		{name: "unknown wp type", mutate: func(p *pagemap.Page) { p.WPType = "widget" }, field: "wpType"},
		{name: "unknown status", mutate: func(p *pagemap.Page) { p.Status = "lost" }, field: "status"},
		{name: "own parent", mutate: func(p *pagemap.Page) { p.ParentPageID = new(p.ID) }, field: "parentPageId"},
		{name: "empty entity", mutate: func(p *pagemap.Page) { p.EntityID = new("") }, field: "entityId"},
		{name: "empty template", mutate: func(p *pagemap.Page) { p.TemplateID = new("") }, field: "templateId"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p := page(pageA, "/a/", nil)
			tc.mutate(&p)
			_, err := pagemap.NewPage(p)
			if !errors.IsCode(err, errors.Invalid) {
				t.Fatalf("code = %q, want INVALID", errors.CodeOf(err))
			}
			if got := fieldOf(t, err); got != tc.field {
				t.Errorf("field = %q, want %q", got, tc.field)
			}
		})
	}
}

func TestNewPageLink(t *testing.T) {
	t.Parallel()

	valid := pagemap.PageLink{ID: "l1", SiteID: siteA, FromPageID: pageA, ToPageID: new(pageB), ToURL: " https://a/b/ ", AnchorText: " bags ", Origin: pagemap.OriginGenerated, ObservedAt: stamp}
	link, err := pagemap.NewPageLink(valid)
	if err != nil {
		t.Fatalf("NewPageLink: %v", err)
	}
	if link.ToURL != "https://a/b/" || link.AnchorText != "bags" {
		t.Errorf("NewPageLink did not trim: %+v", link)
	}

	cases := []struct {
		name   string
		mutate func(*pagemap.PageLink)
		field  string
	}{
		{name: "no id", mutate: func(l *pagemap.PageLink) { l.ID = "" }, field: "id"},
		{name: "no site", mutate: func(l *pagemap.PageLink) { l.SiteID = "" }, field: "siteId"},
		{name: "no source page", mutate: func(l *pagemap.PageLink) { l.FromPageID = "" }, field: "fromPageId"},
		{name: "empty target page", mutate: func(l *pagemap.PageLink) { l.ToPageID = new("") }, field: "toPageId"},
		{name: "no target at all", mutate: func(l *pagemap.PageLink) { l.ToPageID = nil; l.ToURL = " " }, field: "toUrl"},
		{name: "target page without a url", mutate: func(l *pagemap.PageLink) { l.ToURL = " " }, field: "toUrl"},
		{name: "unknown origin", mutate: func(l *pagemap.PageLink) { l.Origin = "guessed" }, field: "origin"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			l := valid
			tc.mutate(&l)
			_, err := pagemap.NewPageLink(l)
			if !errors.IsCode(err, errors.Invalid) {
				t.Fatalf("code = %q, want INVALID", errors.CodeOf(err))
			}
			if got := fieldOf(t, err); got != tc.field {
				t.Errorf("field = %q, want %q", got, tc.field)
			}
		})
	}
}

func TestUnmapped(t *testing.T) {
	t.Parallel()

	pages := []pagemap.Page{page(pageC, "/c/", nil), page(pageA, "/a/", new(entA)), page(pageB, "/b/", nil)}
	got := pagemap.Unmapped(pages)
	if len(got) != 2 || got[0].Path != "/b/" || got[1].Path != "/c/" {
		t.Errorf("Unmapped = %+v", got)
	}
}

func TestEnums(t *testing.T) {
	t.Parallel()

	for _, wp := range []pagemap.WPType{pagemap.WPPage, pagemap.WPPost, pagemap.WPProduct, pagemap.WPProductCategory} {
		if !wp.Valid() {
			t.Errorf("%q must be valid", wp)
		}
	}
	for _, status := range []pagemap.Status{pagemap.StatusPlanned, pagemap.StatusExists, pagemap.StatusPublished, pagemap.StatusArchived} {
		if !status.Valid() {
			t.Errorf("%q must be valid", status)
		}
	}
	if pagemap.WPType("x").Valid() || pagemap.Status("x").Valid() || pagemap.LinkOrigin("x").Valid() {
		t.Error("unknown enum values must be invalid")
	}
	if !pagemap.OriginGenerated.Valid() || !pagemap.OriginObserved.Valid() {
		t.Error("origins must be valid")
	}
	if !pagemap.SortCreatedAt.Valid() || !pagemap.SortPath.Valid() || pagemap.Sort("x").Valid() {
		t.Error("sort validity is wrong")
	}
}

func TestATypeKnowsTheTaxonomyItsCategoriesComeFrom(t *testing.T) {
	t.Parallel()

	cases := []struct {
		wpType   pagemap.WPType
		taxonomy graph.Taxonomy
		has      bool
	}{
		{wpType: pagemap.WPPage, taxonomy: graph.TaxonomyCategory, has: true},
		{wpType: pagemap.WPPost, taxonomy: graph.TaxonomyCategory, has: true},
		{wpType: pagemap.WPProduct, taxonomy: graph.TaxonomyProductCategory, has: true},
		{wpType: pagemap.WPProductCategory},
		{wpType: "attachment"},
	}

	for _, tc := range cases {
		taxonomy, has := tc.wpType.Taxonomy()
		if taxonomy != tc.taxonomy || has != tc.has {
			t.Errorf("%q.Taxonomy() = %q, %t; want %q, %t", tc.wpType, taxonomy, has, tc.taxonomy, tc.has)
		}
	}
}

func TestATypeKnowsItsNumberSequenceAndItsFamily(t *testing.T) {
	t.Parallel()

	cases := []struct {
		left   pagemap.WPType
		right  pagemap.WPType
		term   bool
		family bool
	}{
		{left: pagemap.WPPage, right: pagemap.WPPage, family: true},
		{left: pagemap.WPPage, right: pagemap.WPPost, family: true},
		{left: pagemap.WPPost, right: pagemap.WPProduct},
		{left: pagemap.WPProduct, right: pagemap.WPProduct, family: true},
		{left: pagemap.WPProduct, right: pagemap.WPProductCategory},
		{left: pagemap.WPProductCategory, right: pagemap.WPProductCategory, term: true, family: true},
		{left: pagemap.WPProductCategory, right: pagemap.WPPage, term: true},
	}

	for _, tc := range cases {
		t.Run(string(tc.left)+" and "+string(tc.right), func(t *testing.T) {
			t.Parallel()

			if got := tc.left.Term(); got != tc.term {
				t.Errorf("%s.Term() = %t, want %t", tc.left, got, tc.term)
			}
			if got := tc.left.SameFamily(tc.right); got != tc.family {
				t.Errorf("%s.SameFamily(%s) = %t, want %t", tc.left, tc.right, got, tc.family)
			}
			if got := tc.right.SameFamily(tc.left); got != tc.family {
				t.Errorf("the family is not symmetric for %s and %s", tc.left, tc.right)
			}
		})
	}
}
