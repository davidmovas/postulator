package sqlite_test

import (
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

func TestPageTypeRepoRoundTrip(t *testing.T) {
	t.Parallel()

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")
	other := sqlitetest.Site(t, store, "other")
	repo := sqlite.NewPageTypeRepo(store)
	templateID := sqlitetest.Template(t, store, "Hub writer").ID

	hub := pagetype.Type{ID: id.New(), SiteID: owner.ID, Name: "Hub", Preset: pagetype.Hub, CreatedAt: sqlitetest.Stamp, UpdatedAt: sqlitetest.Stamp}
	pillar := pagetype.Type{
		ID: id.New(), SiteID: owner.ID, Name: "Pillar", Preset: pagetype.Hub, TemplateID: &templateID,
		CreatedAt: sqlitetest.Stamp.Add(time.Minute), UpdatedAt: sqlitetest.Stamp.Add(time.Minute),
	}
	for _, kind := range []pagetype.Type{pillar, hub} {
		if err := repo.Insert(t.Context(), kind); err != nil {
			t.Fatalf("Insert %s: %v", kind.Name, err)
		}
	}
	got, err := repo.Get(t.Context(), pillar.ID)
	if err != nil || !reflect.DeepEqual(got, pillar) {
		t.Fatalf("Get = %+v %v, want %+v", got, err, pillar)
	}
	listed, err := repo.ListBySite(t.Context(), owner.ID)
	if err != nil {
		t.Fatalf("ListBySite: %v", err)
	}
	if names := typeNames(listed); !slices.Equal(names, []string{"Hub", "Pillar"}) {
		t.Fatalf("ListBySite = %v, want the oldest first", names)
	}

	same := pagetype.Type{ID: id.New(), SiteID: owner.ID, Name: "PILLAR", Preset: pagetype.Entity, CreatedAt: sqlitetest.Stamp, UpdatedAt: sqlitetest.Stamp}
	if err = repo.Insert(t.Context(), same); !errors.IsCode(err, errors.Conflict) {
		t.Errorf("a second type of one name = %q, want CONFLICT", errors.CodeOf(err))
	}
	same.SiteID = other.ID
	if err = repo.Insert(t.Context(), same); err != nil {
		t.Errorf("the same name on another site: %v", err)
	}
	stranger := same
	stranger.ID, stranger.SiteID = id.New(), "no-such-site"
	if err = repo.Insert(t.Context(), stranger); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("an unknown site = %q, want INVALID", errors.CodeOf(err))
	}

	pillar.Name, pillar.TemplateID, pillar.UpdatedAt = "Category", nil, sqlitetest.Stamp.Add(time.Hour)
	if err = repo.Update(t.Context(), pillar); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got, err = repo.Get(t.Context(), pillar.ID); err != nil || !reflect.DeepEqual(got, pillar) {
		t.Fatalf("Get after update = %+v %v, want %+v", got, err, pillar)
	}
	hub.Name = "category"
	if err = repo.Update(t.Context(), hub); !errors.IsCode(err, errors.Conflict) {
		t.Errorf("a rename onto another type's name = %q, want CONFLICT", errors.CodeOf(err))
	}

	if err = repo.Delete(t.Context(), pillar.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err = repo.Get(t.Context(), pillar.ID); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("Get after delete = %q, want NOT_FOUND", errors.CodeOf(err))
	}
	if err = repo.Delete(t.Context(), pillar.ID); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("Delete twice = %q, want NOT_FOUND", errors.CodeOf(err))
	}
}

func typeNames(types []pagetype.Type) []string {
	out := make([]string, 0, len(types))
	for i := range types {
		out = append(out, types[i].Name)
	}
	return out
}

func TestFieldScopeRepoKeepsOneScopeAFieldAndGoesWithItsType(t *testing.T) {
	t.Parallel()

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")
	types := sqlite.NewPageTypeRepo(store)
	scopes := sqlite.NewFieldScopeRepo(store)
	hub := pagetype.Type{ID: id.New(), SiteID: owner.ID, Name: "Hub", Preset: pagetype.Hub, CreatedAt: sqlitetest.Stamp, UpdatedAt: sqlitetest.Stamp}
	if err := types.Insert(t.Context(), hub); err != nil {
		t.Fatalf("Insert: %v", err)
	}

	body := pagetype.DefaultScope(pagetype.Hub, field.Body, hub.ID)
	body.Instruction, body.MinLength, body.MaxLength, body.UpdatedAt = "Lead with the products the hub lists.", 300, 1200, sqlitetest.Stamp
	custom := field.Scope{
		PageTypeID: hub.ID, Key: field.Instance(field.Meta, "spec_sheet"), Enabled: true, Label: "Spec sheet",
		Provider: field.ProviderStore, Links: field.NoLinks(), UpdatedAt: sqlitetest.Stamp,
	}
	for _, scope := range []field.Scope{body, custom} {
		if err := scopes.Put(t.Context(), scope); err != nil {
			t.Fatalf("Put %s: %v", scope.Key, err)
		}
	}
	listed, err := scopes.ListBySite(t.Context(), owner.ID)
	if err != nil {
		t.Fatalf("ListBySite: %v", err)
	}
	if !reflect.DeepEqual(listed, []field.Scope{body, custom}) {
		t.Fatalf("ListBySite = %+v\nwant %+v", listed, []field.Scope{body, custom})
	}

	body.Enabled, body.Rewrites, body.Links.Rules.MaxLinks = false, true, 7
	body.UpdatedAt = sqlitetest.Stamp.Add(time.Minute)
	if err = scopes.Put(t.Context(), body); err != nil {
		t.Fatalf("Put again: %v", err)
	}
	if listed, err = scopes.ListBySite(t.Context(), owner.ID); err != nil || len(listed) != 2 || !reflect.DeepEqual(listed[0], body) {
		t.Fatalf("after a second Put = %+v %v, want the one scope replaced", listed, err)
	}

	if err = types.Delete(t.Context(), hub.ID); err != nil {
		t.Fatalf("Delete the type: %v", err)
	}
	if listed, err = scopes.ListBySite(t.Context(), owner.ID); err != nil || len(listed) != 0 {
		t.Fatalf("scopes of a deleted type = %+v %v, want none", listed, err)
	}
}
