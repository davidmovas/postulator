package pages_test

import (
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/application/pages"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func TestReplaceLinks(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	from := h.page(t, "/shop/", nil)
	to := h.page(t, "/shop/shoes/", nil)
	h.recorder.Reset()

	replaced, err := h.service.ReplaceLinks(t.Context(), pages.ReplaceLinksRequest{PageID: from.ID, Links: []pages.LinkInput{
		{ToPageID: &to.ID, ToURL: "/shop/shoes/", AnchorText: "shoes"},
		{ToURL: "https://elsewhere.example.com/", AnchorText: "elsewhere", Origin: "observed"},
	}})
	if err != nil {
		t.Fatalf("ReplaceLinks: %v", err)
	}
	if len(replaced.Links) != 2 || replaced.Links[0].Origin != "generated" || replaced.Links[1].Origin != "observed" || replaced.Links[0].FromPageID != from.ID || replaced.Links[0].ObservedAt.String() != "2026-09-18T09:00:00Z" {
		t.Errorf("ReplaceLinks = %+v", replaced.Links)
	}
	h.wantEvents(t, events.PagesChanged)

	got, err := h.service.Get(t.Context(), pages.GetRequest{ID: from.ID})
	if err != nil || len(got.Links) != 2 {
		t.Errorf("Get links = %+v, %v", got.Links, err)
	}

	foreignSite := sqlitetest.Site(t, h.store, "blog")
	foreign := sqlitetest.Page(t, h.store, foreignSite.ID, "/x/")
	if _, err = h.service.ReplaceLinks(t.Context(), pages.ReplaceLinksRequest{PageID: from.ID, Links: []pages.LinkInput{{ToPageID: &foreign.ID, ToURL: "/x/", AnchorText: "x"}}}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("link to another site code = %q, want INVALID", errors.CodeOf(err))
	}
	missing := "missing"
	if _, err = h.service.ReplaceLinks(t.Context(), pages.ReplaceLinksRequest{PageID: from.ID, Links: []pages.LinkInput{{ToPageID: &missing, ToURL: "/y/", AnchorText: "y"}}}); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("link to an unknown page code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
	if _, err = h.service.ReplaceLinks(t.Context(), pages.ReplaceLinksRequest{PageID: from.ID, Links: []pages.LinkInput{{AnchorText: "nowhere"}}}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("link without a target code = %q, want INVALID", errors.CodeOf(err))
	}
	if _, err = h.service.ReplaceLinks(t.Context(), pages.ReplaceLinksRequest{PageID: "missing"}); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("unknown page code = %q, want NOT_FOUND", errors.CodeOf(err))
	}

	kept, err := sqlite.NewPageLinkRepo(h.store).ListForPage(t.Context(), from.ID)
	if err != nil || len(kept) != 2 {
		t.Errorf("a refused replacement must leave the previous links untouched, got %d, %v", len(kept), err)
	}
	if len(h.recorder.Events()) != 0 {
		t.Error("refused replacements must not publish")
	}

	cleared, err := h.service.ReplaceLinks(t.Context(), pages.ReplaceLinksRequest{PageID: from.ID})
	if err != nil || len(cleared.Links) != 0 {
		t.Errorf("clearing = %+v, %v", cleared.Links, err)
	}
	h.wantEvents(t, events.PagesChanged)
}
