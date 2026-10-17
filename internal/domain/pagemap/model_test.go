package pagemap_test

import (
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func TestNewPageKeepsAnObservedSlug(t *testing.T) {
	t.Parallel()

	observed, err := pagemap.NewPage(pagemap.Page{
		ID: "p", SiteID: "s", Path: "/peptides/bpc-157/", Slug: "bpc-157-peptide",
		WPType: pagemap.WPPage, Status: pagemap.StatusPublished,
	})
	if err != nil {
		t.Fatalf("NewPage: %v", err)
	}
	if observed.Slug != "bpc-157-peptide" {
		t.Errorf("an observed slug was replaced by %q", observed.Slug)
	}

	planned, err := pagemap.NewPage(pagemap.Page{
		ID: "p", SiteID: "s", Path: "/peptides/bpc-157/", WPType: pagemap.WPPage, Status: pagemap.StatusPlanned,
	})
	if err != nil {
		t.Fatalf("NewPage: %v", err)
	}
	if planned.Slug != "bpc-157" {
		t.Errorf("a planned page without a slug takes the path's, got %q", planned.Slug)
	}
}

func TestWPTypeAcceptsAnyRegisteredType(t *testing.T) {
	t.Parallel()

	for _, valid := range []pagemap.WPType{"page", "post", "product", "product_cat", "faq", "wp_block2"} {
		if !valid.Valid() {
			t.Errorf("%q is refused", valid)
		}
	}
	for _, invalid := range []pagemap.WPType{"", "Faq", "my type", "faq/x"} {
		if invalid.Valid() {
			t.Errorf("%q is accepted", invalid)
		}
	}
	faq := pagemap.WPType("faq")
	if faq.Term() || faq.StoreAddressed() {
		t.Error("a custom post type is neither a term nor store addressed")
	}
	if !faq.SameFamily(pagemap.WPPage) || faq.SameFamily(pagemap.WPProduct) || faq.SameFamily(pagemap.WPProductCategory) {
		t.Error("a custom post type shares the posts' family only")
	}
}

func TestNewPageReadsTheReasons(t *testing.T) {
	t.Parallel()

	page, err := pagemap.NewPage(pagemap.Page{
		ID: "p", SiteID: "s", Path: "/x/", WPType: pagemap.WPPage, Status: pagemap.StatusPublished,
		ParentReason: pagemap.ParentJSONLD, TechnicalReason: pagemap.TechnicalShop, ReadOnlyReason: pagemap.ReadOnlyBuilder("elementor"),
	})
	if err != nil {
		t.Fatalf("NewPage: %v", err)
	}
	if page.ParentReason != pagemap.ParentJSONLD || page.TechnicalReason != pagemap.TechnicalShop || page.ReadOnlyReason != "builder:elementor" {
		t.Errorf("the reasons did not survive: %+v", page)
	}
	if _, err := pagemap.NewPage(pagemap.Page{
		ID: "p", SiteID: "s", Path: "/x/", WPType: pagemap.WPPage, Status: pagemap.StatusPublished, ParentReason: "guess",
	}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("an unknown parent reason is accepted: %v", err)
	}
	if _, err := pagemap.NewPage(pagemap.Page{
		ID: "p", SiteID: "s", Path: "/x/", WPType: pagemap.WPPage, Status: pagemap.StatusPublished, TechnicalReason: "because",
	}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("an unknown technical reason is accepted: %v", err)
	}
	if pagemap.TechnicalWorkbook.Observed() || pagemap.TechnicalManual.Observed() || !pagemap.TechnicalNoindex.Observed() {
		t.Error("the observation owns every reason but workbook and manual")
	}
}

func TestNewPageLinkRefusesAnUnknownTypeOrSource(t *testing.T) {
	t.Parallel()

	position := 2
	link := pagemap.PageLink{
		ID: "l", SiteID: "s", FromPageID: "a", ToURL: "/b/", AnchorText: "b", Origin: pagemap.OriginObserved,
		Type: pagemap.LinkContextual, Region: "content", Source: pagemap.SourceRaw, Position: &position, Ours: true,
		ObservedAt: time.Now(),
	}
	if _, err := pagemap.NewPageLink(link); err != nil {
		t.Fatalf("a typed link is refused: %v", err)
	}
	untyped := link
	untyped.Type = ""
	if _, err := pagemap.NewPageLink(untyped); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("a link without a type is accepted: %v", err)
	}
	unsourced := link
	unsourced.Source = ""
	if _, err := pagemap.NewPageLink(unsourced); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("a link without a source is accepted: %v", err)
	}
}
