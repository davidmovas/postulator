package sqlite_test

import (
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func snapshotOf(pageID string, side block.Side, raw string, at time.Time) block.Snapshot {
	return block.Snapshot{PageID: pageID, Side: side, Raw: raw, Hash: block.Hash(raw), Kind: block.Parse(raw).Kind, ObservedAt: at}
}

func TestPageBodyRepoObserveKeepsTheBase(t *testing.T) {
	t.Parallel()

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")
	other := sqlitetest.Site(t, store, "other")
	page := sqlitetest.Page(t, store, owner.ID, "/tb-500/")
	second := sqlitetest.Page(t, store, owner.ID, "/bpc-157/")
	elsewhere := sqlitetest.Page(t, store, other.ID, "/elsewhere/")
	repo := sqlite.NewPageBodyRepo(store)

	if _, _, found, err := repo.Get(t.Context(), page.ID); err != nil || found {
		t.Fatalf("Get before Observe = found %t, %v; want nothing", found, err)
	}

	first := "<p>The first body.</p>"
	if err := repo.Observe(t.Context(), snapshotOf(page.ID, block.Current, first, sqlitetest.Stamp)); err != nil {
		t.Fatalf("Observe: %v", err)
	}
	current, base, found, err := repo.Get(t.Context(), page.ID)
	if err != nil || !found {
		t.Fatalf("Get after the first Observe = found %t, %v", found, err)
	}
	if want := snapshotOf(page.ID, block.Current, first, sqlitetest.Stamp); !reflect.DeepEqual(current, want) {
		t.Fatalf("current after the first Observe = %+v\nwant %+v", current, want)
	}
	if want := snapshotOf(page.ID, block.Base, first, sqlitetest.Stamp); !reflect.DeepEqual(base, want) {
		t.Fatalf("base after the first Observe = %+v\nwant %+v", base, want)
	}

	edited := "<!-- wp:paragraph -->\n<p>Edited on the site.</p>\n<!-- /wp:paragraph -->"
	later := sqlitetest.Stamp.Add(time.Hour)
	if err = repo.Observe(t.Context(), snapshotOf(page.ID, block.Current, edited, later)); err != nil {
		t.Fatalf("Observe again: %v", err)
	}
	current, base, _, err = repo.Get(t.Context(), page.ID)
	if err != nil {
		t.Fatalf("Get after a second Observe: %v", err)
	}
	if want := snapshotOf(page.ID, block.Current, edited, later); !reflect.DeepEqual(current, want) || current.Kind != block.Blocks {
		t.Fatalf("current after a second Observe = %+v\nwant %+v", current, want)
	}
	if want := snapshotOf(page.ID, block.Base, first, sqlitetest.Stamp); !reflect.DeepEqual(base, want) {
		t.Fatalf("the base moved to %+v\nwant it kept at %+v", base, want)
	}

	for _, pageID := range []string{second.ID, elsewhere.ID} {
		if err = repo.Observe(t.Context(), snapshotOf(pageID, block.Current, first, sqlitetest.Stamp)); err != nil {
			t.Fatalf("Observe another page: %v", err)
		}
	}
	want := []block.Snapshot{
		snapshotOf(page.ID, block.Base, first, sqlitetest.Stamp),
		snapshotOf(page.ID, block.Current, edited, later),
		snapshotOf(second.ID, block.Base, first, sqlitetest.Stamp),
		snapshotOf(second.ID, block.Current, first, sqlitetest.Stamp),
	}
	slices.SortStableFunc(want, func(a, b block.Snapshot) int { return strings.Compare(a.PageID, b.PageID) })
	listed, err := repo.ListBySite(t.Context(), owner.ID)
	if err != nil || !reflect.DeepEqual(listed, want) {
		t.Fatalf("ListBySite = %+v, %v\nwant %+v", listed, err, want)
	}

	if err = repo.Observe(t.Context(), snapshotOf("no-such-page", block.Current, first, sqlitetest.Stamp)); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("the body of no page code = %q, want INVALID", errors.CodeOf(err))
	}
	if err = sqlite.NewPageRepo(store).Delete(t.Context(), page.ID); err != nil {
		t.Fatalf("delete the page: %v", err)
	}
	if _, _, found, err = repo.Get(t.Context(), page.ID); err != nil || found {
		t.Errorf("Get after the page is deleted = found %t, %v; want nothing", found, err)
	}
}

func TestPageBodyRepoRecordSetsBothSides(t *testing.T) {
	t.Parallel()

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")
	page := sqlitetest.Page(t, store, owner.ID, "/tb-500/")
	planned := sqlitetest.Page(t, store, owner.ID, "/new/")
	repo := sqlite.NewPageBodyRepo(store)

	if err := repo.Observe(t.Context(), snapshotOf(page.ID, block.Current, "<p>Theirs.</p>", sqlitetest.Stamp)); err != nil {
		t.Fatalf("Observe: %v", err)
	}
	if err := repo.Observe(t.Context(), snapshotOf(page.ID, block.Current, "<p>Theirs, edited.</p>", sqlitetest.Stamp.Add(time.Minute))); err != nil {
		t.Fatalf("Observe again: %v", err)
	}

	ours := "<p>Theirs, edited.</p>\n\n<p>Ours.</p>"
	written := sqlitetest.Stamp.Add(time.Hour)
	for _, pageID := range []string{page.ID, planned.ID} {
		if err := repo.Record(t.Context(), pageID, ours, written); err != nil {
			t.Fatalf("Record: %v", err)
		}
		current, base, found, err := repo.Get(t.Context(), pageID)
		if err != nil || !found {
			t.Fatalf("Get after Record = found %t, %v", found, err)
		}
		if want := snapshotOf(pageID, block.Current, ours, written); !reflect.DeepEqual(current, want) {
			t.Errorf("current after Record = %+v\nwant %+v", current, want)
		}
		if want := snapshotOf(pageID, block.Base, ours, written); !reflect.DeepEqual(base, want) {
			t.Errorf("base after Record = %+v\nwant %+v", base, want)
		}
	}
	if err := repo.Record(t.Context(), "no-such-page", ours, written); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("recording the body of no page code = %q, want INVALID", errors.CodeOf(err))
	}
}

func TestPageBodyRepoReplacesTheWrittenBlocks(t *testing.T) {
	t.Parallel()

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")
	page := sqlitetest.Page(t, store, owner.ID, "/tb-500/")
	repo := sqlite.NewPageBodyRepo(store)

	if got, err := repo.ListWritten(t.Context(), page.ID); err != nil || len(got) != 0 {
		t.Fatalf("ListWritten on a fresh page = %+v, %v", got, err)
	}
	if err := repo.ReplaceWritten(t.Context(), page.ID, []string{"bbb", "aaa", "bbb"}, "run-1", sqlitetest.Stamp); err != nil {
		t.Fatalf("ReplaceWritten: %v", err)
	}
	want := []block.Written{
		{PageID: page.ID, BlockHash: "aaa", RunID: "run-1", WrittenAt: sqlitetest.Stamp},
		{PageID: page.ID, BlockHash: "bbb", RunID: "run-1", WrittenAt: sqlitetest.Stamp},
	}
	if got, err := repo.ListWritten(t.Context(), page.ID); err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("ListWritten = %+v, %v\nwant %+v", got, err, want)
	}

	later := sqlitetest.Stamp.Add(time.Hour)
	if err := repo.ReplaceWritten(t.Context(), page.ID, []string{"ccc"}, "run-2", later); err != nil {
		t.Fatalf("ReplaceWritten again: %v", err)
	}
	want = []block.Written{{PageID: page.ID, BlockHash: "ccc", RunID: "run-2", WrittenAt: later}}
	if got, err := repo.ListWritten(t.Context(), page.ID); err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("ListWritten after a second replace = %+v, %v\nwant %+v", got, err, want)
	}

	if err := repo.ReplaceWritten(t.Context(), "no-such-page", []string{"ddd"}, "run-3", later); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("the blocks of no page code = %q, want INVALID", errors.CodeOf(err))
	}
	if err := repo.ReplaceWritten(t.Context(), page.ID, nil, "run-3", later); err != nil {
		t.Fatalf("ReplaceWritten with none: %v", err)
	}
	if got, err := repo.ListWritten(t.Context(), page.ID); err != nil || len(got) != 0 {
		t.Fatalf("ListWritten after clearing = %+v, %v", got, err)
	}

	if err := repo.ReplaceWritten(t.Context(), page.ID, []string{"eee"}, "run-4", later); err != nil {
		t.Fatalf("ReplaceWritten before the delete: %v", err)
	}
	if err := sqlite.NewPageRepo(store).Delete(t.Context(), page.ID); err != nil {
		t.Fatalf("delete the page: %v", err)
	}
	if got, err := repo.ListWritten(t.Context(), page.ID); err != nil || len(got) != 0 {
		t.Errorf("ListWritten after the page is deleted = %+v, %v", got, err)
	}
}
