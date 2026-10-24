package sqlite_test

import (
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

func TestTheWalkReadsAnswerForThePagesAndTheBodiesTheyName(t *testing.T) {
	t.Parallel()

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")
	other := sqlitetest.Site(t, store, "other")
	a := sqlitetest.Page(t, store, owner.ID, "/a/")
	b := sqlitetest.Page(t, store, owner.ID, "/b/")
	elsewhere := sqlitetest.Page(t, store, other.ID, "/elsewhere/")

	identities := sqlite.NewPageIdentityRepo(store)
	for _, page := range []pagemap.Page{a, b, elsewhere} {
		if err := identities.Put(t.Context(), pagemap.Identity{PageID: page.ID, URL: "https://shop.example" + page.Path, Status: "publish",
			WPType: pagemap.WPPage, ObservedAt: sqlitetest.Stamp}); err != nil {
			t.Fatalf("Put: %v", err)
		}
	}
	listed, err := identities.ListForPages(t.Context(), []string{b.ID, id.New()})
	if err != nil || len(listed) != 1 || listed[0].PageID != b.ID {
		t.Fatalf("ListForPages = %+v, %v; want /b/'s identity alone", listed, err)
	}

	bodies := sqlite.NewPageBodyRepo(store)
	later := sqlitetest.Stamp.Add(time.Hour)
	for _, held := range []struct {
		page pagemap.Page
		at   time.Time
	}{{a, sqlitetest.Stamp}, {a, later}, {elsewhere, later}} {
		raw := "<p>" + held.at.String() + "</p>"
		if err = bodies.Observe(t.Context(), block.Snapshot{
			PageID: held.page.ID, Side: block.Current, Raw: raw, Hash: block.BodyHash(raw), Kind: block.Classic, ObservedAt: held.at,
		}); err != nil {
			t.Fatalf("Observe: %v", err)
		}
	}
	observed, err := bodies.ObservedBySite(t.Context(), owner.ID)
	if err != nil || len(observed) != 1 || !observed[a.ID].Equal(later) {
		t.Errorf("ObservedBySite = %v, %v; want /a/'s current body seen at %v", observed, err, later)
	}
}
