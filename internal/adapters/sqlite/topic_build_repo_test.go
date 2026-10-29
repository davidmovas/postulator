package sqlite_test

import (
	"slices"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
)

func TestTopicBuildRepoRemembersWhatEachSiteBuilt(t *testing.T) {
	t.Parallel()

	store := sqlitetest.Open(t)
	shop := sqlitetest.Site(t, store, "shop")
	blog := sqlitetest.Site(t, store, "blog")
	repo := sqlite.NewTopicBuildRepo(store)

	for _, recorded := range []struct{ siteID, subject string }{
		{shop.ID, "page:p2"}, {shop.ID, "topic:/peptides"}, {shop.ID, "page:p2"}, {blog.ID, "page:p9"},
	} {
		if err := repo.Record(t.Context(), recorded.siteID, recorded.subject, sqlitetest.Stamp); err != nil {
			t.Fatalf("Record %s: %v", recorded.subject, err)
		}
	}

	cases := []struct {
		name   string
		siteID string
		want   []string
	}{
		{name: "a site reads its own subjects once each", siteID: shop.ID, want: []string{"page:p2", "topic:/peptides"}},
		{name: "another site reads only its own", siteID: blog.ID, want: []string{"page:p9"}},
		{name: "a site that built nothing reads nothing", siteID: "unknown", want: []string{}},
	}
	for _, tc := range cases {
		got, err := repo.ListBySite(t.Context(), tc.siteID)
		if err != nil || !slices.Equal(got, tc.want) {
			t.Errorf("%s: ListBySite = %v, %v; want %v", tc.name, got, err, tc.want)
		}
	}
	if err := repo.Record(t.Context(), shop.ID, " ", sqlitetest.Stamp); err == nil {
		t.Error("an empty subject was recorded")
	}
}
