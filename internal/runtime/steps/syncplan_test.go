package steps_test

import (
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

func installUnder(t *testing.T, h *syncHarness, suffix string) {
	t.Helper()

	repo := sqlite.NewSiteRepo(h.store)
	owner, err := repo.Get(t.Context(), h.siteID)
	if err != nil {
		t.Fatalf("read the site: %v", err)
	}
	owner.BaseURL += suffix
	if err = repo.Update(t.Context(), owner); err != nil {
		t.Fatalf("move the site under %s: %v", suffix, err)
	}
}

func TestSyncSiteKeepsThePlanOfAPageItNeverPublished(t *testing.T) {
	t.Parallel()

	h := newSyncHarness(t, 0)
	flat := h.server.Seed(wptest.Item{
		Type: wptest.TypePage, Title: "E-Bike Range", Slug: "e-bike-range", Status: "draft",
		Content: `<h1>E-Bike Range</h1><p>How far.</p>`,
	})
	wpID := flat[0].ID

	planned := pagemap.Page{
		ID: id.New(), SiteID: h.siteID, Path: "/components/batteries/e-bike-range/", Slug: "e-bike-range",
		WPType: pagemap.WPPage, WPID: &wpID, Title: "How Far Can an E-Bike Go?", H1: "How far can an e-bike go?",
		MetaTitle: "E-bike range | Shop", MetaDescription: "How far an e-bike goes on one charge.",
		Status: pagemap.StatusPlanned, CreatedAt: sqlitetest.Stamp, UpdatedAt: sqlitetest.Stamp,
	}
	if err := h.pages.Insert(t.Context(), planned); err != nil {
		t.Fatalf("insert the planned page: %v", err)
	}

	h.all(t)

	stored, err := h.pages.Get(t.Context(), planned.ID)
	if err != nil {
		t.Fatalf("read the planned page: %v", err)
	}
	fields := []struct {
		name string
		got  string
		want string
	}{
		{name: "path", got: stored.Path, want: planned.Path},
		{name: "slug", got: stored.Slug, want: planned.Slug},
		{name: "title", got: stored.Title, want: planned.Title},
		{name: "h1", got: stored.H1, want: planned.H1},
		{name: "metaTitle", got: stored.MetaTitle, want: planned.MetaTitle},
		{name: "metaDescription", got: stored.MetaDescription, want: planned.MetaDescription},
		{name: "status", got: string(stored.Status), want: string(pagemap.StatusPlanned)},
	}
	for _, field := range fields {
		if field.got != field.want {
			t.Errorf("%s = %q, want the plan %q", field.name, field.got, field.want)
		}
	}
	if stored.Observed.Title != "E-Bike Range" || stored.Observed.Slug != "e-bike-range" {
		t.Fatalf("the mirror = %+v, want what the site holds", stored.Observed)
	}
	if stored.WPID == nil || *stored.WPID != wpID {
		t.Fatalf("the page lost the id the site gave it: %+v", stored.WPID)
	}
}

func TestSyncSiteAdoptsAKnownPageThatCarriesNoPlan(t *testing.T) {
	t.Parallel()

	h := newSyncHarness(t, 0)
	seeded := h.server.Seed(wptest.Item{
		Type: wptest.TypePage, Title: "Coffee", Slug: "coffee", Status: "draft",
		Content: `<h1>Coffee</h1><p>All about coffee.</p>`,
	})
	h.all(t)

	adopted := h.byPath(t, "/coffee/")
	if adopted.Title != "Coffee" {
		t.Fatalf("the discovered page = %+v", adopted)
	}

	renamed := "Coffee, roasted"
	if _, err := syncClient(t, h.server).UpdateItem(t.Context(), wp.TypePage, seeded[0].ID, wp.UpdateItem{
		Title: &renamed,
	}); err != nil {
		t.Fatalf("rename the page on the site: %v", err)
	}

	h.restart(t)
	h.all(t)

	stored, err := h.pages.Get(t.Context(), adopted.ID)
	if err != nil {
		t.Fatalf("read the adopted page: %v", err)
	}
	if stored.Title != renamed {
		t.Fatalf("title = %q, want the site's own %q: there was no plan to protect", stored.Title, renamed)
	}
	if found := stored.Mismatches(); len(found) != 0 {
		t.Fatalf("mismatches = %+v, want none", found)
	}
}

func TestSyncSiteDoesNotReplaceAStoredValueWithNothing(t *testing.T) {
	t.Parallel()

	h := newSyncHarness(t, 0, wptest.WithoutPlugin())
	h.server.Seed(wptest.Item{
		Type: wptest.TypePage, Title: "Coffee", Slug: "coffee", Content: `<h1>Coffee</h1><p>All about coffee.</p>`,
	})
	h.all(t)

	adopted := h.byPath(t, "/coffee/")
	adopted.MetaTitle = "Coffee | Shop"
	adopted.MetaDescription = "Everything we roast."
	if err := h.pages.Update(t.Context(), adopted); err != nil {
		t.Fatalf("write the meta a human typed: %v", err)
	}

	h.restart(t)
	h.all(t)

	stored, err := h.pages.Get(t.Context(), adopted.ID)
	if err != nil {
		t.Fatalf("read the adopted page: %v", err)
	}
	if stored.MetaTitle != adopted.MetaTitle || stored.MetaDescription != adopted.MetaDescription {
		t.Fatalf("the meta = %q / %q, want what was stored: core REST says nothing about it",
			stored.MetaTitle, stored.MetaDescription)
	}
}

func TestSyncSiteKeepsTheLinksItGenerated(t *testing.T) {
	t.Parallel()

	h := newSyncHarness(t, 0)
	seedSite(t, h)
	h.all(t)

	espresso := h.byPath(t, "/coffee/espresso/")
	coffee := h.byPath(t, "/coffee/")
	gone := "/coffee/gone/"
	if err := h.links.ReplaceForPage(t.Context(), espresso.ID, []pagemap.PageLink{
		{
			ID: id.New(), SiteID: h.siteID, FromPageID: espresso.ID, ToPageID: &coffee.ID, ToURL: coffee.Path,
			AnchorText: "coffee", Origin: pagemap.OriginGenerated, ObservedAt: sqlitetest.Stamp,
		},
		{
			ID: id.New(), SiteID: h.siteID, FromPageID: espresso.ID, ToURL: gone,
			AnchorText: "gone", Origin: pagemap.OriginGenerated, ObservedAt: sqlitetest.Stamp,
		},
	}); err != nil {
		t.Fatalf("record the generated links: %v", err)
	}

	h.restart(t)
	h.all(t)

	stored, err := h.links.ListForPage(t.Context(), espresso.ID)
	if err != nil {
		t.Fatalf("list the links: %v", err)
	}
	if len(stored) != 1 {
		t.Fatalf("the page carries %d links, want only the one the site still holds: %+v", len(stored), stored)
	}
	if stored[0].ToURL != coffee.Path {
		t.Fatalf("the kept link points at %q", stored[0].ToURL)
	}
	if stored[0].Origin != pagemap.OriginGenerated {
		t.Fatalf("origin = %q, want the run's own %q", stored[0].Origin, pagemap.OriginGenerated)
	}
}

func TestSyncSiteResolvesARelativeHrefAgainstTheInstallBase(t *testing.T) {
	t.Parallel()

	h := newSyncHarness(t, 0, wptest.WithoutPlugin())
	h.server.Seed(wptest.Item{
		Type: wptest.TypePage, Title: "Coffee", Slug: "coffee",
		Content: `<h1>Coffee</h1><p>Try <a href="espresso/">espresso</a> and <a href="#faq">the faq</a>.</p>`,
	})
	installUnder(t, h, "/blog")

	h.all(t)

	coffee := h.byPath(t, "/coffee/")
	stored, err := h.links.ListForPage(t.Context(), coffee.ID)
	if err != nil {
		t.Fatalf("list the links: %v", err)
	}
	if len(stored) != 1 {
		t.Fatalf("the page carries %d links, want the one relative href: %+v", len(stored), stored)
	}
	if stored[0].ToURL != "/blog/espresso/" {
		t.Fatalf("the relative href resolved to %q, want it joined to the install base", stored[0].ToURL)
	}
}
