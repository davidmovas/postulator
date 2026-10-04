package sqlite_test

import (
	"reflect"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/domain/category"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

func newCategory(t *testing.T, siteID, name, parentID string) category.Category {
	t.Helper()

	record, err := category.New(category.Category{
		ID: id.New(), SiteID: siteID, Name: name, ParentID: parentID, CreatedAt: sqlitetest.Stamp, UpdatedAt: sqlitetest.Stamp,
	})
	if err != nil {
		t.Fatalf("category %q: %v", name, err)
	}
	return record
}

func insertCategories(t *testing.T, repo *sqlite.CategoryRepo, records ...category.Category) {
	t.Helper()

	for i := range records {
		if err := repo.Insert(t.Context(), records[i]); err != nil {
			t.Fatalf("Insert %q: %v", records[i].Name, err)
		}
	}
}

func TestCategoryRepoInsertsAndLists(t *testing.T) {
	t.Parallel()

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")
	other := sqlitetest.Site(t, store, "blog")
	repo := sqlite.NewCategoryRepo(store)

	healing := newCategory(t, owner.ID, "Healing", "")
	recovery := newCategory(t, owner.ID, "recovery", "")
	zinc := newCategory(t, owner.ID, "Zinc", "")
	bpc := newCategory(t, owner.ID, "BPC-157", healing.ID)
	bpcLiquid := newCategory(t, owner.ID, "Liquid", bpc.ID)
	recoveryLiquid := newCategory(t, owner.ID, "Liquid", recovery.ID)
	elsewhere := newCategory(t, other.ID, "Healing", "")
	insertCategories(t, repo, healing, bpc, bpcLiquid, zinc, recovery, recoveryLiquid, elsewhere)

	listed, err := repo.ListBySite(t.Context(), owner.ID)
	if err != nil {
		t.Fatalf("ListBySite: %v", err)
	}
	if want := []category.Category{healing, recovery, zinc, bpc, recoveryLiquid, bpcLiquid}; !reflect.DeepEqual(listed, want) {
		t.Fatalf("ListBySite = %+v\nwant parents before children, then by name: %+v", listed, want)
	}

	if listed, err = repo.ListBySite(t.Context(), other.ID); err != nil || !reflect.DeepEqual(listed, []category.Category{elsewhere}) {
		t.Fatalf("the other site's categories = %+v, %v", listed, err)
	}
	if listed, err = repo.ListBySite(t.Context(), id.New()); err != nil || listed == nil || len(listed) != 0 {
		t.Fatalf("a site with no categories = %#v, %v; want an empty list", listed, err)
	}
}

func TestCategoryRepoStoresTheKeyTheNameFoldsTo(t *testing.T) {
	t.Parallel()

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")
	repo := sqlite.NewCategoryRepo(store)

	stale := category.Category{
		ID: id.New(), SiteID: owner.ID, Name: "Tools &amp; Kits", Key: "stale", CreatedAt: sqlitetest.Stamp, UpdatedAt: sqlitetest.Stamp,
	}
	insertCategories(t, repo, stale)

	listed, err := repo.ListBySite(t.Context(), owner.ID)
	if err != nil || len(listed) != 1 {
		t.Fatalf("ListBySite = %+v, %v", listed, err)
	}
	if got := listed[0]; got.Key != "tools & kits" || got.Name != stale.Name {
		t.Fatalf("stored name %q and key %q, want the name kept and the key it folds to", got.Name, got.Key)
	}
	if err = repo.Insert(t.Context(), newCategory(t, owner.ID, "TOOLS & KITS", "")); !errors.IsCode(err, errors.Conflict) {
		t.Fatalf("a name that folds to the stored key = %v, want CONFLICT", err)
	}
}

func TestCategoryRepoRefuses(t *testing.T) {
	t.Parallel()

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")
	other := sqlitetest.Site(t, store, "blog")
	repo := sqlite.NewCategoryRepo(store)

	healing := newCategory(t, owner.ID, "Healing", "")
	bpc := newCategory(t, owner.ID, "BPC-157", healing.ID)
	foreign := newCategory(t, other.ID, "Foreign", "")
	insertCategories(t, repo, healing, bpc, foreign)

	cases := []struct {
		name   string
		record category.Category
		code   errors.Code
	}{
		{name: "the same name at the top in another case", record: newCategory(t, owner.ID, "HEALING", ""), code: errors.Conflict},
		{name: "the same name under one parent", record: newCategory(t, owner.ID, "bpc-157", healing.ID), code: errors.Conflict},
		{name: "the same id twice", record: func() category.Category {
			again := newCategory(t, owner.ID, "Recovery", "")
			again.ID = healing.ID
			return again
		}(), code: errors.Conflict},
		{name: "a parent on another site", record: newCategory(t, owner.ID, "Liquid", foreign.ID), code: errors.Invalid},
		{name: "a parent that does not exist", record: newCategory(t, owner.ID, "Liquid", id.New()), code: errors.Invalid},
		{name: "a site that does not exist", record: newCategory(t, id.New(), "Liquid", ""), code: errors.Invalid},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := repo.Insert(t.Context(), tc.record); !errors.IsCode(err, tc.code) {
				t.Fatalf("Insert = %v, want %s", err, tc.code)
			}
		})
	}

	listed, err := repo.ListBySite(t.Context(), owner.ID)
	if err != nil || !reflect.DeepEqual(listed, []category.Category{healing, bpc}) {
		t.Fatalf("refused categories were stored: %+v, %v", listed, err)
	}
}

func TestCategoryRepoDeleteTakesTheBranchItsTermsAndItsPages(t *testing.T) {
	t.Parallel()

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")
	repo := sqlite.NewCategoryRepo(store)
	terms := sqlite.NewCategoryTermRepo(store)
	pages := sqlite.NewPageRepo(store)

	healing := newCategory(t, owner.ID, "Healing", "")
	bpc := newCategory(t, owner.ID, "BPC-157", healing.ID)
	liquid := newCategory(t, owner.ID, "Liquid", bpc.ID)
	recovery := newCategory(t, owner.ID, "Recovery", "")
	insertCategories(t, repo, healing, bpc, liquid, recovery)

	kept := storedCategoryTerm(recovery, category.TaxonomyCategory, 9, 0)
	for _, term := range []category.Term{
		storedCategoryTerm(bpc, category.TaxonomyCategory, 6, 5), storedCategoryTerm(liquid, category.TaxonomyProductCategory, 31, 0), kept,
	} {
		if err := terms.Upsert(t.Context(), term); err != nil {
			t.Fatalf("Upsert the %s term: %v", term.Name, err)
		}
	}

	dangling := id.New()
	filed := map[string]string{"/liquid/": liquid.ID, "/bpc/": bpc.ID, "/recovery/": recovery.ID, "/plain/": "", "/stray/": dangling}
	byPath := make(map[string]string, len(filed))
	for path, categoryID := range filed {
		record := fullPage(owner.ID, path, sqlitetest.Stamp)
		record.CategoryID = categoryID
		if err := pages.Insert(t.Context(), record); err != nil {
			t.Fatalf("Insert the page %s: %v", path, err)
		}
		byPath[path] = record.ID
	}
	requirePages := func(want map[string]string) {
		t.Helper()
		for path, categoryID := range want {
			got, err := pages.Get(t.Context(), byPath[path])
			if err != nil {
				t.Fatalf("Get the page %s: %v", path, err)
			}
			if got.CategoryID != categoryID {
				t.Errorf("the page %s is filed under %q, want %q", path, got.CategoryID, categoryID)
			}
		}
	}

	if err := repo.Delete(t.Context(), bpc.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	listed, err := repo.ListBySite(t.Context(), owner.ID)
	if err != nil || !reflect.DeepEqual(listed, []category.Category{healing, recovery}) {
		t.Fatalf("ListBySite after the delete = %+v, %v; want the branch gone", listed, err)
	}
	stored, err := terms.ListBySite(t.Context(), owner.ID)
	if err != nil || !reflect.DeepEqual(stored, []category.Term{kept}) {
		t.Fatalf("terms after the delete = %+v, %v; want only the one outside the branch", stored, err)
	}
	requirePages(map[string]string{"/liquid/": "", "/bpc/": "", "/recovery/": recovery.ID, "/plain/": "", "/stray/": dangling})

	if err = repo.Delete(t.Context(), bpc.ID); !errors.IsCode(err, errors.NotFound) {
		t.Fatalf("Delete twice = %v, want NOT_FOUND", err)
	}
	if err = repo.Delete(t.Context(), dangling); !errors.IsCode(err, errors.NotFound) {
		t.Fatalf("Delete a category that never was = %v, want NOT_FOUND", err)
	}
	requirePages(map[string]string{"/stray/": dangling})
}
