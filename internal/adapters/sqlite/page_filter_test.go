package sqlite_test

import (
	"slices"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/contextpack"
	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
	"github.com/davidmovas/postulator/internal/kernel/id"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

func TestThePagesAreFilteredByTypeTextFindingsAndSubtreeAndTypedInBulk(t *testing.T) {
	t.Parallel()

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")
	other := sqlitetest.Site(t, store, "other")
	repo := sqlite.NewPageRepo(store)
	home := sqlitetest.Page(t, store, owner.ID, "/")
	hub := sqlitetest.Page(t, store, owner.ID, "/peptides/")
	child := sqlitetest.Page(t, store, owner.ID, "/peptides/healing/")
	guide := sqlitetest.Page(t, store, owner.ID, "/lab-guides/")
	elsewhere := sqlitetest.Page(t, store, other.ID, "/peptides/")

	rows := []contextpack.MapRow{
		mapRow(home, "", home.ID+"/"), mapRow(hub, home.ID, home.ID+"/"+hub.ID+"/"),
		mapRow(child, hub.ID, home.ID+"/"+hub.ID+"/"+child.ID+"/"), mapRow(guide, home.ID, home.ID+"/"+guide.ID+"/"),
	}
	rows[1].TypeID, rows[2].TypeID, rows[3].TypeID = "category-type", "category-type", "guide-type"
	rows[3].TitleKey = "lab guides for peptides"
	if err := sqlite.NewPageMapRepo(store).Write(t.Context(), owner.ID, rows, nil, 1); err != nil {
		t.Fatalf("Write the map: %v", err)
	}
	if err := sqlite.NewPageBodyRepo(store).Observe(t.Context(), block.Snapshot{
		PageID: guide.ID, Side: block.Current, Raw: "<p>Text.</p>", Hash: "h", Kind: block.Classic, ObservedAt: sqlitetest.Stamp,
	}); err != nil {
		t.Fatalf("Observe the body: %v", err)
	}
	open, err := finding.New(finding.Finding{
		ID: id.New(), SiteID: owner.ID, PageID: &hub.ID, Code: finding.TypeByDefault, SubjectKey: finding.SubjectPage(hub.ID),
		Subject: []byte(`{}`), CreatedAt: sqlitetest.Stamp, UpdatedAt: sqlitetest.Stamp,
	})
	if err != nil {
		t.Fatalf("finding.New: %v", err)
	}
	if err = sqlite.NewFindingRepo(store).Upsert(t.Context(), open); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	cases := map[string]struct {
		q    pagemap.Query
		want []string
	}{
		"a type":              {pagemap.Query{SiteID: owner.ID, PageTypeID: "category-type"}, []string{hub.Path, child.Path}},
		"a subtree":           {pagemap.Query{SiteID: owner.ID, Trail: home.ID + "/" + hub.ID + "/"}, []string{hub.Path, child.Path}},
		"words in the title":  {pagemap.Query{SiteID: owner.ID, Search: "guides for"}, []string{guide.Path}},
		"words in the path":   {pagemap.Query{SiteID: owner.ID, Search: "healing"}, []string{child.Path}},
		"a page with text":    {pagemap.Query{SiteID: owner.ID, Body: pagemap.BodyWith}, []string{guide.Path}},
		"pages without text":  {pagemap.Query{SiteID: owner.ID, Body: pagemap.BodyWithout}, []string{home.Path, hub.Path, child.Path}},
		"an open finding":     {pagemap.Query{SiteID: owner.ID, Findings: true}, []string{hub.Path}},
		"a type and its text": {pagemap.Query{SiteID: owner.ID, PageTypeID: "guide-type", Body: pagemap.BodyWithout}, nil},
		"by title":            {pagemap.Query{SiteID: owner.ID, Sort: pagemap.SortTitle}, []string{home.Path, guide.Path, hub.Path, child.Path}},
	}
	for name, tc := range cases {
		listed, listErr := repo.List(t.Context(), tc.q, paging.Request{Limit: 50})
		if listErr != nil {
			t.Fatalf("%s: List: %v", name, listErr)
		}
		got := make([]string, 0, len(listed.Items))
		for i := range listed.Items {
			got = append(got, listed.Items[i].Path)
		}
		if tc.q.Sort == "" {
			slices.Sort(got)
			slices.Sort(tc.want)
		}
		if !slices.Equal(got, tc.want) && (len(got) != 0 || len(tc.want) != 0) {
			t.Errorf("%s = %v, want %v", name, got, tc.want)
		}
		count, countErr := repo.Count(t.Context(), tc.q)
		if countErr != nil || count != len(tc.want) {
			t.Errorf("%s: Count = %d %v, want %d", name, count, countErr, len(tc.want))
		}
	}
	ids, err := repo.IDs(t.Context(), pagemap.Query{SiteID: owner.ID, PageTypeID: "category-type"})
	if err != nil || len(ids) != 2 {
		t.Fatalf("IDs = %v %v", ids, err)
	}
	under, err := sqlite.NewPageMapRepo(store).IDsUnder(t.Context(), owner.ID, home.ID+"/"+hub.ID+"/", "category-type")
	if err != nil || !slices.Equal(under, []string{hub.ID, child.ID}) {
		t.Fatalf("IDsUnder = %v %v, want the hub and its child", under, err)
	}

	types := sqlite.NewPageTypeRepo(store)
	kind := pagetype.Type{ID: id.New(), SiteID: owner.ID, Name: "Guide", Preset: pagetype.Guide, CreatedAt: sqlitetest.Stamp, UpdatedAt: sqlitetest.Stamp}
	if err = types.Insert(t.Context(), kind); err != nil {
		t.Fatalf("Insert the type: %v", err)
	}
	at := sqlitetest.Stamp.Add(time.Hour)
	typed, err := repo.SetTypes(t.Context(), owner.ID, []string{hub.ID, child.ID, elsewhere.ID}, kind.ID, pagemap.TypeByUser, at)
	if err != nil || typed != 2 {
		t.Fatalf("SetTypes = %d %v, want the two pages of the site", typed, err)
	}
	if again, againErr := repo.SetTypes(t.Context(), owner.ID, []string{hub.ID}, kind.ID, pagemap.TypeByUser, at); againErr != nil || again != 0 {
		t.Errorf("typing a page with the type it holds = %d %v, want nothing written", again, againErr)
	}
	stored, err := repo.Get(t.Context(), hub.ID)
	if err != nil || stored.TypeSource != pagemap.TypeSourceManual || *stored.PageTypeID != kind.ID {
		t.Fatalf("the typed page = %+v %v", stored, err)
	}
	if cleared, clearErr := repo.ClearTypes(t.Context(), owner.ID, []string{hub.ID}, at); clearErr != nil || cleared != 1 {
		t.Fatalf("ClearTypes = %d %v", cleared, clearErr)
	}
	if stored, err = repo.Get(t.Context(), hub.ID); err != nil || stored.PageTypeID != nil || stored.TypeSource != pagemap.TypeSourceNone {
		t.Fatalf("the page handed back = %+v %v, want it untyped", stored, err)
	}

	templateID := sqlitetest.Template(t, store, "Hub writer").ID
	set, err := repo.SetTemplates(t.Context(), owner.ID, []string{hub.ID, child.ID, elsewhere.ID}, &templateID, at)
	if err != nil || set != 2 {
		t.Fatalf("SetTemplates = %d %v", set, err)
	}
	if cleared, clearErr := repo.SetTemplates(t.Context(), owner.ID, []string{hub.ID}, nil, at); clearErr != nil || cleared != 1 {
		t.Fatalf("clearing a template = %d %v", cleared, clearErr)
	}
	if stored, err = repo.Get(t.Context(), child.ID); err != nil || stored.TemplateID == nil || *stored.TemplateID != templateID {
		t.Fatalf("the child's template = %+v %v", stored.TemplateID, err)
	}
	if foreign, getErr := repo.Get(t.Context(), elsewhere.ID); getErr != nil || foreign.TemplateID != nil || foreign.PageTypeID != nil {
		t.Errorf("another site's page = %+v %v, want it untouched", foreign, getErr)
	}
}
