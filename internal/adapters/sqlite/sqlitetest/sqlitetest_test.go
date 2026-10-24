package sqlitetest_test

import (
	"context"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
)

func TestOpen(t *testing.T) {
	t.Parallel()

	store := sqlitetest.Open(t)
	if err := store.Do(t.Context(), func(context.Context) error { return nil }); err != nil {
		t.Fatalf("Do: %v", err)
	}
}

func TestACountedStoreTalliesWhatOneCallRuns(t *testing.T) {
	t.Parallel()

	store, counter := sqlitetest.OpenCounted(t)
	owner := sqlitetest.Site(t, store, "counted")
	for _, path := range []string{"/a/", "/b/", "/c/"} {
		sqlitetest.Page(t, store, owner.ID, path)
	}
	pages := sqlite.NewPageRepo(store)
	if _, err := pages.ListBySite(t.Context(), owner.ID); err != nil {
		t.Fatalf("warm the reader: %v", err)
	}

	counter.Reset()
	listed, err := pages.ListBySite(t.Context(), owner.ID)
	if err != nil {
		t.Fatalf("ListBySite: %v", err)
	}
	got := counter.Tally()
	if got.Statements != 1 || got.Rows != len(listed) || len(listed) != 3 {
		t.Errorf("one listing of three pages tallied %+v", got)
	}
	if distinct := counter.Distinct(); len(distinct) != 1 || !strings.Contains(distinct[0], "FROM pages") {
		t.Errorf("the distinct statements = %v, want the one listing", distinct)
	}

	measured, err := counter.Measure(func() error {
		_, getErr := pages.Get(t.Context(), listed[0].ID)
		return getErr
	})
	if err != nil {
		t.Fatalf("Measure: %v", err)
	}
	if measured.Statements != 1 || measured.Rows != 1 {
		t.Errorf("one page read tallied %+v", measured)
	}

	counter.Reset()
	if tally := counter.Tally(); tally.Statements != 0 || tally.Rows != 0 || len(counter.Distinct()) != 0 {
		t.Errorf("a reset counter holds %+v and %v", tally, counter.Distinct())
	}
}

func TestKeyFitsTheCipher(t *testing.T) {
	t.Parallel()

	if key := sqlitetest.Key(); len(key) != 32 {
		t.Fatalf("Key length = %d, want 32", len(key))
	}
}
