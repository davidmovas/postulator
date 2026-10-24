package sqlite_test

import (
	"encoding/json"
	"maps"
	"slices"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/domain/contextpack"
	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/kernel/id"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

func TestFindingRepoListsAndCountsInSQL(t *testing.T) {
	t.Parallel()

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")
	other := sqlitetest.Site(t, store, "other")
	hub := sqlitetest.Page(t, store, owner.ID, "/hub/")
	child := sqlitetest.Page(t, store, owner.ID, "/hub/child/")
	away := sqlitetest.Page(t, store, owner.ID, "/away/")
	elsewhere := sqlitetest.Page(t, store, other.ID, "/elsewhere/")
	generation, _, err := sqlite.NewPageMapRepo(store).State(t.Context(), owner.ID)
	if err != nil {
		t.Fatalf("State: %v", err)
	}
	if err = sqlite.NewPageMapRepo(store).Write(t.Context(), owner.ID, []contextpack.MapRow{
		{PageID: hub.ID, SiteID: owner.ID, Path: hub.Path, Trail: hub.ID + "/", Row: true, TypeID: "hub-type"},
		{PageID: child.ID, SiteID: owner.ID, ParentID: hub.ID, Path: child.Path, Trail: hub.ID + "/" + child.ID + "/", Row: true, TypeID: "entity-type"},
		{PageID: away.ID, SiteID: owner.ID, Path: away.Path, Trail: away.ID + "/", Row: true, TypeID: "entity-type"},
	}, nil, generation); err != nil {
		t.Fatalf("Write the map: %v", err)
	}

	repo := sqlite.NewFindingRepo(store)
	seed := func(siteID string, pageID *string, code finding.Code, subject string, created time.Time) finding.Finding {
		t.Helper()
		record, newErr := finding.New(finding.Finding{
			ID: id.New(), SiteID: siteID, PageID: pageID, Code: code, SubjectKey: subject, Subject: json.RawMessage(`{}`),
			CreatedAt: created, UpdatedAt: created,
		})
		if newErr != nil {
			t.Fatalf("finding.New: %v", newErr)
		}
		if upsertErr := repo.Upsert(t.Context(), record); upsertErr != nil {
			t.Fatalf("Upsert: %v", upsertErr)
		}
		return record
	}
	stamp := sqlitetest.Stamp
	onHub := seed(owner.ID, &hub.ID, finding.PageWithoutEntity, "hub", stamp)
	onChild := seed(owner.ID, &child.ID, finding.PageWithoutEntity, "child", stamp.Add(time.Minute))
	technical := seed(owner.ID, &child.ID, finding.TechnicalDetected, "child", stamp.Add(2*time.Minute))
	onAway := seed(owner.ID, &away.ID, finding.PageWithoutEntity, "away", stamp.Add(3*time.Minute))
	siteWide := seed(owner.ID, nil, finding.URLFormMixed, "site", stamp.Add(4*time.Minute))
	seed(other.ID, &elsewhere.ID, finding.PageWithoutEntity, "elsewhere", stamp)
	if err = repo.Clear(t.Context(), owner.ID, finding.PageWithoutEntity, "away", finding.ResolvedByModel, finding.ResolutionPageMapped, stamp); err != nil {
		t.Fatalf("Clear: %v", err)
	}

	ids := func(q finding.Query, page paging.Request) ([]string, paging.List[finding.Finding]) {
		t.Helper()
		q.SiteID = owner.ID
		listed, listErr := repo.List(t.Context(), q, page)
		if listErr != nil {
			t.Fatalf("List(%+v): %v", q, listErr)
		}
		out := make([]string, 0, len(listed.Items))
		for i := range listed.Items {
			out = append(out, listed.Items[i].ID)
		}
		return out, listed
	}
	cases := []struct {
		name  string
		query finding.Query
		want  []string
	}{
		{"every finding of the site", finding.Query{}, []string{onHub.ID, onChild.ID, technical.ID, onAway.ID, siteWide.ID}},
		{"the open ones", finding.Query{Status: finding.Open}, []string{onHub.ID, onChild.ID, technical.ID, siteWide.ID}},
		{"one code", finding.Query{Code: finding.PageWithoutEntity}, []string{onHub.ID, onChild.ID, onAway.ID}},
		{"one page", finding.Query{PageID: child.ID}, []string{onChild.ID, technical.ID}},
		{"one type", finding.Query{PageTypeID: "entity-type"}, []string{onChild.ID, technical.ID, onAway.ID}},
		{"under a hub", finding.Query{Under: hub.ID + "/"}, []string{onHub.ID, onChild.ID, technical.ID}},
		{"open under a hub by code", finding.Query{Under: hub.ID + "/", Status: finding.Open, Code: finding.TechnicalDetected}, []string{technical.ID}},
	}
	for _, tc := range cases {
		if got, _ := ids(tc.query, paging.Request{}); !slices.Equal(got, tc.want) {
			t.Errorf("%s = %v, want %v", tc.name, got, tc.want)
		}
	}
	first, page := ids(finding.Query{}, paging.Request{Limit: 2})
	if !page.HasMore || len(first) != 2 {
		t.Fatalf("the first page = %v, more %v", first, page.HasMore)
	}
	if rest, _ := ids(finding.Query{}, paging.Request{After: page.Next, Limit: 10}); !slices.Equal(rest, []string{technical.ID, onAway.ID, siteWide.ID}) {
		t.Errorf("the second page = %v", rest)
	}

	counted, err := repo.CountByCode(t.Context(), finding.Query{SiteID: owner.ID, Status: finding.Open, Under: hub.ID + "/"})
	if err != nil || !maps.Equal(counted, map[finding.Code]int{finding.PageWithoutEntity: 2, finding.TechnicalDetected: 1}) {
		t.Errorf("CountByCode under the hub = %v, %v", counted, err)
	}
}
