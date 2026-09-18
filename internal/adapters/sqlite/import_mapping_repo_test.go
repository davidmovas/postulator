package sqlite_test

import (
	"maps"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/domain/importmap"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

func savedMapping(siteID, name string) importmap.Mapping {
	return importmap.Mapping{
		ID:     id.New(),
		SiteID: siteID,
		Name:   name,
		Columns: map[importmap.Field]string{
			importmap.FieldPath:  "URL",
			importmap.FieldTitle: "Title",
		},
		Options:   importmap.Options{PathPrefixStrip: "https://shop.example.com", KeywordSeparator: ";", AnchorSeparator: "|", ListSeparator: ","},
		CreatedAt: sqlitetest.Stamp,
		UpdatedAt: sqlitetest.Stamp,
	}
}

func TestImportMappingRepoUpsertsReadsAndDeletes(t *testing.T) {
	t.Parallel()

	store := sqlitetest.Open(t)
	site := sqlitetest.Site(t, store, "shop")
	repo := sqlite.NewImportMappingRepo(store)

	record := savedMapping(site.ID, "client sheet")
	if err := repo.Upsert(t.Context(), record); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	back, err := repo.Get(t.Context(), record.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !maps.Equal(back.Columns, record.Columns) {
		t.Fatalf("columns = %v, want %v", back.Columns, record.Columns)
	}
	if back.Options != record.Options || back.Name != record.Name || !back.CreatedAt.Equal(record.CreatedAt) {
		t.Fatalf("mapping = %+v, want %+v", back, record)
	}

	record.Name = "client sheet v2"
	record.Columns[importmap.FieldKeywords] = "Keywords"
	record.UpdatedAt = sqlitetest.Stamp.Add(1)
	if err = repo.Upsert(t.Context(), record); err != nil {
		t.Fatalf("second Upsert: %v", err)
	}

	listed, err := repo.ListBySite(t.Context(), site.ID)
	if err != nil {
		t.Fatalf("ListBySite: %v", err)
	}
	if len(listed) != 1 || listed[0].Name != "client sheet v2" || len(listed[0].Columns) != 3 {
		t.Fatalf("listed = %+v", listed)
	}

	if err = repo.Delete(t.Context(), record.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err = repo.Get(t.Context(), record.ID); !errors.IsCode(err, errors.NotFound) {
		t.Fatalf("Get after Delete = %v, want not found", err)
	}
	if err = repo.Delete(t.Context(), record.ID); !errors.IsCode(err, errors.NotFound) {
		t.Fatalf("second Delete = %v, want not found", err)
	}
}

func TestImportMappingRepoKeepsTheNameUniquePerSite(t *testing.T) {
	t.Parallel()

	store := sqlitetest.Open(t)
	site := sqlitetest.Site(t, store, "shop")
	repo := sqlite.NewImportMappingRepo(store)

	first := savedMapping(site.ID, "client sheet")
	if err := repo.Upsert(t.Context(), first); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	second := savedMapping(site.ID, "Client Sheet")
	if err := repo.Upsert(t.Context(), second); !errors.IsCode(err, errors.Conflict) {
		t.Fatalf("Upsert = %v, want a conflict", err)
	}

	other := sqlitetest.Site(t, store, "blog")
	if err := repo.Upsert(t.Context(), savedMapping(other.ID, "client sheet")); err != nil {
		t.Fatalf("Upsert for another site: %v", err)
	}

	listed, err := repo.ListBySite(t.Context(), site.ID)
	if err != nil {
		t.Fatalf("ListBySite: %v", err)
	}
	if len(listed) != 1 {
		t.Fatalf("listed = %d mappings, want one", len(listed))
	}
}
