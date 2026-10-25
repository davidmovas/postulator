package pages_test

import (
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/application/pages"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/render"
)

func (h harness) observeH1(t *testing.T, pageID, h1 string, source render.H1Source) {
	t.Helper()
	now := h.clock.Now()
	if err := sqlite.NewPageFieldRepo(h.store).Observe(t.Context(), pageID, []field.Value{field.NewValue(pageID, field.H1, h1, now)}); err != nil {
		t.Fatalf("observe the h1: %v", err)
	}
	if source == render.H1Unknown {
		return
	}
	record := render.Record{PageID: pageID, Extraction: render.Extraction{Hash: "h", Root: render.RootFound, H1: h1, H1Source: source}, ExtractedAt: now}
	if err := sqlite.NewPageRenderRepo(h.store).Put(t.Context(), record); err != nil {
		t.Fatalf("store the render: %v", err)
	}
}

func TestAPageShowsTheObservedH1AndItsSourceBesideThePlannedOne(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	created, err := h.service.Create(t.Context(), pages.CreateRequest{SiteID: h.siteID, Path: "/tb-500/", Title: "TB-500", H1: "TB-500 Research Peptide"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	themed := created.Page
	own := h.page(t, "/bpc-157/", nil)
	unrendered := h.page(t, "/healing/", nil)
	bare := h.page(t, "/peptides/", nil)
	h.observeH1(t, themed.ID, "TB-500", render.H1Theme)
	h.observeH1(t, own.ID, "BPC-157 explained", render.H1Content)
	h.observeH1(t, unrendered.ID, "Healing peptides", render.H1Unknown)

	want := map[string][2]string{
		themed.ID:     {"TB-500", "theme"},
		own.ID:        {"BPC-157 explained", "content"},
		unrendered.ID: {"Healing peptides", "content"},
		bare.ID:       {"", "none"},
	}
	listed, err := h.service.List(t.Context(), pages.ListRequest{SiteID: h.siteID})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	for _, page := range listed.Items {
		if got := [2]string{page.ObservedH1, page.H1Source}; got != want[page.ID] {
			t.Errorf("%s lists the h1 %q from %q, want %q from %q", page.Path, got[0], got[1], want[page.ID][0], want[page.ID][1])
		}
		planned := ""
		if page.ID == themed.ID {
			planned = "TB-500 Research Peptide"
		}
		if page.H1 != planned {
			t.Errorf("%s: the planned h1 became %q, want %q", page.Path, page.H1, planned)
		}
	}

	got, err := h.service.Get(t.Context(), pages.GetRequest{ID: themed.ID})
	if err != nil || got.Page.ObservedH1 != "TB-500" || got.Page.H1Source != "theme" {
		t.Errorf("Get = %q from %q, %v; want the theme's TB-500", got.Page.ObservedH1, got.Page.H1Source, err)
	}

	tree, err := h.service.Tree(t.Context(), pages.TreeRequest{SiteID: h.siteID})
	if err != nil {
		t.Fatalf("Tree: %v", err)
	}
	for _, node := range tree.Roots {
		if got := [2]string{node.Page.ObservedH1, node.Page.H1Source}; got != want[node.Page.ID] {
			t.Errorf("the tree shows %s with %q from %q", node.Page.Path, got[0], got[1])
		}
	}
}
