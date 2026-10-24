package sqlite_test

import (
	"encoding/json"
	"maps"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

func TestTheNodeReadsAnswerForThePagesTheyName(t *testing.T) {
	t.Parallel()

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")
	a := sqlitetest.Page(t, store, owner.ID, "/a/")
	b := sqlitetest.Page(t, store, owner.ID, "/b/")
	c := sqlitetest.Page(t, store, owner.ID, "/c/")

	listed, err := sqlite.NewPageRepo(store).ListByIDs(t.Context(), []string{c.ID, a.ID, id.New()})
	if err != nil || len(listed) != 2 || listed[0].ID != a.ID || listed[1].ID != c.ID {
		t.Fatalf("ListByIDs = %+v, %v; want /a/ and /c/ in path order", listed, err)
	}

	link := func(from, to pagemap.Page, kind pagemap.LinkType, source pagemap.LinkSource) pagemap.PageLink {
		return pagemap.PageLink{
			ID: id.New(), SiteID: owner.ID, FromPageID: from.ID, ToPageID: &to.ID, ToURL: to.Path, Type: kind, Source: source,
			ObservedAt: sqlitetest.Stamp,
		}
	}
	links := sqlite.NewPageLinkRepo(store)
	if err = links.ReplaceForPage(t.Context(), a.ID, pagemap.SourceRaw, []pagemap.PageLink{
		link(a, b, pagemap.LinkContextual, pagemap.SourceRaw), link(a, b, pagemap.LinkContextual, pagemap.SourceRaw),
		link(a, a, pagemap.LinkContextual, pagemap.SourceRaw),
	}); err != nil {
		t.Fatalf("ReplaceForPage: %v", err)
	}
	if err = links.ReplaceForPage(t.Context(), c.ID, pagemap.SourceRender, []pagemap.PageLink{
		link(c, b, pagemap.LinkNavigation, pagemap.SourceRender), link(c, b, pagemap.LinkContextual, pagemap.SourceRender),
	}); err != nil {
		t.Fatalf("ReplaceForPage: %v", err)
	}
	inbound, outbound, err := links.CountContextual(t.Context(), []string{a.ID, b.ID, c.ID})
	if err != nil {
		t.Fatalf("CountContextual: %v", err)
	}
	if want := map[string]int{b.ID: 3}; !maps.Equal(inbound, want) {
		t.Errorf("inbound = %v, want three contextual links into /b/ and none from a page to itself", inbound)
	}
	if want := map[string]int{a.ID: 3, c.ID: 1}; !maps.Equal(outbound, want) {
		t.Errorf("outbound = %v, want every contextual link a page holds", outbound)
	}

	repo := sqlite.NewFindingRepo(store)
	for _, held := range []struct {
		page    pagemap.Page
		code    finding.Code
		subject string
	}{
		{a, finding.PageWithoutEntity, "a"}, {a, finding.TechnicalDetected, "a"}, {c, finding.PageWithoutEntity, "c"},
	} {
		pageID := held.page.ID
		record, newErr := finding.New(finding.Finding{
			ID: id.New(), SiteID: owner.ID, PageID: &pageID, Code: held.code, SubjectKey: held.subject, Subject: json.RawMessage(`{}`),
			CreatedAt: sqlitetest.Stamp, UpdatedAt: sqlitetest.Stamp,
		})
		if newErr != nil {
			t.Fatalf("finding.New: %v", newErr)
		}
		if err = repo.Upsert(t.Context(), record); err != nil {
			t.Fatalf("Upsert: %v", err)
		}
	}
	if err = repo.Clear(t.Context(), owner.ID, finding.PageWithoutEntity, "c", finding.ResolvedByModel, finding.ResolutionPageMapped, sqlitetest.Stamp); err != nil {
		t.Fatalf("Clear: %v", err)
	}
	counted, err := repo.CountOpen(t.Context(), []string{a.ID, b.ID, c.ID})
	if err != nil || !maps.Equal(counted, map[string]int{a.ID: 2}) {
		t.Errorf("CountOpen = %v, %v; want /a/'s two open findings", counted, err)
	}
	if empty, emptyErr := repo.CountOpen(t.Context(), nil); emptyErr != nil || len(empty) != 0 {
		t.Errorf("CountOpen of nothing = %v, %v", empty, emptyErr)
	}
}
