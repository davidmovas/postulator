package sqlite_test

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

func globalTemplate(name string, at time.Time) template.Template {
	seed := template.Seed()[3]
	return template.Template{ID: id.New(), Scope: template.ScopeGlobal, Name: name, PageKind: seed.PageKind, Version: 1, Spec: seed.Spec, CreatedAt: at, UpdatedAt: at}
}

func templateNames(list []template.Template) []string {
	out := make([]string, 0, len(list))
	for i := range list {
		out = append(out, list[i].Name)
	}
	return out
}

func TestTemplateRepoRoundTrip(t *testing.T) {
	t.Parallel()

	store := sqlitetest.Open(t)
	repo := sqlite.NewTemplateRepo(store)
	want := globalTemplate("Hub", sqlitetest.Stamp)

	if err := repo.Insert(t.Context(), want); err != nil {
		t.Fatalf("Insert: %v", err)
	}
	got, err := repo.Get(t.Context(), want.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Get = %+v\nwant %+v", got, want)
	}
	if err = repo.Insert(t.Context(), globalTemplate("hub", sqlitetest.Stamp)); !errors.IsCode(err, errors.Conflict) {
		t.Errorf("duplicate global name code = %q, want CONFLICT", errors.CodeOf(err))
	}

	owner := sqlitetest.Site(t, store, "shop")
	scoped := globalTemplate("Hub", sqlitetest.Stamp)
	scoped.Scope = template.ScopeSite
	scoped.SiteID = &owner.ID
	if err = repo.Insert(t.Context(), scoped); err != nil {
		t.Fatalf("a site may reuse a global name: %v", err)
	}
	if err = repo.Insert(t.Context(), scoped); !errors.IsCode(err, errors.Conflict) {
		t.Errorf("duplicate site name code = %q, want CONFLICT", errors.CodeOf(err))
	}

	want.Name = "Hub Page"
	want.Version = 2
	want.Spec.Tone = "warm"
	want.UpdatedAt = sqlitetest.Stamp.Add(time.Minute)
	if err = repo.Update(t.Context(), want); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got, err = repo.Get(t.Context(), want.ID)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("Get after update = %+v, %v\nwant %+v", got, err, want)
	}

	if err = repo.Delete(t.Context(), want.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if err = repo.Delete(t.Context(), want.ID); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("Delete twice code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
	if err = repo.Update(t.Context(), want); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("Update missing code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
	if _, err = repo.Get(t.Context(), want.ID); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("Get missing code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
}

func TestAStoredEarlierBuiltInReadsBackAsShipped(t *testing.T) {
	t.Parallel()

	store := sqlitetest.Open(t)
	repo := sqlite.NewTemplateRepo(store)
	current := make(map[string]template.Template)
	for _, seed := range template.Seed() {
		current[seed.Name] = seed
	}

	for _, earlier := range template.Superseded() {
		record := earlier
		record.ID = id.New()
		record.CreatedAt = sqlitetest.Stamp
		record.UpdatedAt = sqlitetest.Stamp
		if err := repo.Insert(t.Context(), record); err != nil {
			t.Fatalf("Insert %s: %v", record.Name, err)
		}
		read, err := repo.Get(t.Context(), record.ID)
		if err != nil {
			t.Fatalf("Get %s: %v", record.Name, err)
		}
		if !template.Supersedes(current[record.Name], read) {
			t.Errorf("the stored %s no longer reads as the copy an earlier release shipped", record.Name)
		}
	}
}

func TestTemplateRepoListAndFilters(t *testing.T) {
	t.Parallel()

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")
	repo := sqlite.NewTemplateRepo(store)

	names := []string{"Category", "Guide", "Hub"}
	for i, name := range names {
		record := globalTemplate(name, sqlitetest.Stamp.Add(time.Duration(i)*time.Minute))
		if name == "Guide" {
			record.PageKind = "guide"
		}
		if err := repo.Insert(t.Context(), record); err != nil {
			t.Fatalf("Insert %s: %v", name, err)
		}
	}
	scoped := globalTemplate("Landing", sqlitetest.Stamp.Add(time.Hour))
	scoped.Scope = template.ScopeSite
	scoped.SiteID = &owner.ID
	if err := repo.Insert(t.Context(), scoped); err != nil {
		t.Fatalf("Insert scoped: %v", err)
	}

	global := template.ScopeGlobal
	first, err := repo.List(t.Context(), template.Query{Scope: &global, Sort: template.SortName}, paging.Request{Limit: 2})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if got := templateNames(first.Items); !reflect.DeepEqual(got, []string{"Category", "Guide"}) || !first.HasMore {
		t.Fatalf("first page = %v", got)
	}
	rest, err := repo.List(t.Context(), template.Query{Scope: &global, Sort: template.SortName}, paging.Request{After: first.Next, Limit: 2})
	if err != nil {
		t.Fatalf("List after: %v", err)
	}
	if got := templateNames(rest.Items); !reflect.DeepEqual(got, []string{"Hub"}) || rest.HasMore {
		t.Fatalf("second page = %v", got)
	}
	back, err := repo.List(t.Context(), template.Query{Scope: &global, Sort: template.SortName}, paging.Request{Before: rest.Prev, Limit: 2})
	if err != nil {
		t.Fatalf("List before: %v", err)
	}
	if got := templateNames(back.Items); !reflect.DeepEqual(got, []string{"Category", "Guide"}) {
		t.Fatalf("backward page = %v", got)
	}

	bySite, err := repo.List(t.Context(), template.Query{SiteID: &owner.ID, Sort: template.SortCreatedAt}, paging.Request{Limit: 10})
	if err != nil || len(bySite.Items) != 1 || bySite.Items[0].Name != "Landing" {
		t.Errorf("by site = %+v, %v", bySite.Items, err)
	}
	byKind, err := repo.List(t.Context(), template.Query{PageKind: "guide", Sort: template.SortCreatedAt}, paging.Request{Limit: 10})
	if err != nil || len(byKind.Items) != 1 || byKind.Items[0].Name != "Guide" {
		t.Errorf("by kind = %+v, %v", byKind.Items, err)
	}
	byName, err := repo.List(t.Context(), template.Query{Scope: &global, Name: "hub", Sort: template.SortCreatedAt}, paging.Request{Limit: 10})
	if err != nil || len(byName.Items) != 1 || byName.Items[0].Name != "Hub" {
		t.Errorf("by name must be case-insensitive, got %+v, %v", byName.Items, err)
	}
	newest, err := repo.List(t.Context(), template.Query{Sort: template.SortCreatedAt, Desc: true}, paging.Request{Limit: 1})
	if err != nil || len(newest.Items) != 1 || newest.Items[0].Name != "Landing" {
		t.Errorf("newest = %+v, %v", newest.Items, err)
	}
}

func TestTemplateRepoOverrides(t *testing.T) {
	t.Parallel()

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")
	page := sqlitetest.Page(t, store, owner.ID, "/shop/")
	base := sqlitetest.Template(t, store, "Hub")
	repo := sqlite.NewTemplateRepo(store)

	if _, err := repo.GetOverride(t.Context(), base.ID, template.OverrideSite, owner.ID); !errors.IsCode(err, errors.NotFound) {
		t.Fatalf("GetOverride before upsert code = %q, want NOT_FOUND", errors.CodeOf(err))
	}

	siteOverride := template.Override{ID: id.New(), TemplateID: base.ID, Scope: template.OverrideSite, TargetID: owner.ID, Patch: json.RawMessage(`{"tone":"warm"}`), CreatedAt: sqlitetest.Stamp, UpdatedAt: sqlitetest.Stamp}
	stored, err := repo.UpsertOverride(t.Context(), siteOverride)
	if err != nil {
		t.Fatalf("UpsertOverride: %v", err)
	}
	if !reflect.DeepEqual(stored, siteOverride) {
		t.Errorf("stored = %+v\nwant %+v", stored, siteOverride)
	}

	replacement := siteOverride
	replacement.ID = id.New()
	replacement.Patch = json.RawMessage(`{"tone":"cold"}`)
	replacement.UpdatedAt = sqlitetest.Stamp.Add(time.Minute)
	stored, err = repo.UpsertOverride(t.Context(), replacement)
	if err != nil {
		t.Fatalf("UpsertOverride again: %v", err)
	}
	if stored.ID != siteOverride.ID || string(stored.Patch) != `{"tone":"cold"}` || !stored.UpdatedAt.Equal(replacement.UpdatedAt) || !stored.CreatedAt.Equal(siteOverride.CreatedAt) {
		t.Errorf("an existing target keeps its id and creation time: %+v", stored)
	}

	pageOverride := template.Override{ID: id.New(), TemplateID: base.ID, Scope: template.OverridePage, TargetID: page.ID, Patch: json.RawMessage(`{"length":{"min":100}}`), CreatedAt: sqlitetest.Stamp, UpdatedAt: sqlitetest.Stamp}
	if _, err = repo.UpsertOverride(t.Context(), pageOverride); err != nil {
		t.Fatalf("UpsertOverride page: %v", err)
	}
	stranger := pageOverride
	stranger.ID = id.New()
	stranger.TargetID = "no-such-page"
	if _, err = repo.UpsertOverride(t.Context(), stranger); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("unknown page target code = %q, want INVALID", errors.CodeOf(err))
	}

	all, err := repo.ListOverrides(t.Context(), base.ID)
	if err != nil || len(all) != 2 {
		t.Fatalf("ListOverrides = %+v, %v", all, err)
	}
	got, err := repo.GetOverride(t.Context(), base.ID, template.OverridePage, page.ID)
	if err != nil || string(got.Patch) != `{"length":{"min":100}}` {
		t.Errorf("GetOverride page = %+v, %v", got, err)
	}

	if err = repo.DeleteOverride(t.Context(), got.ID); err != nil {
		t.Fatalf("DeleteOverride: %v", err)
	}
	if err = repo.DeleteOverride(t.Context(), got.ID); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("DeleteOverride twice code = %q, want NOT_FOUND", errors.CodeOf(err))
	}

	if err = repo.Delete(t.Context(), base.ID); err != nil {
		t.Fatalf("Delete template: %v", err)
	}
	if _, err = repo.GetOverride(t.Context(), base.ID, template.OverrideSite, owner.ID); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("overrides must cascade with their template, got %v", err)
	}
}

func TestTemplateFixture(t *testing.T) {
	t.Parallel()

	store := sqlitetest.Open(t)
	fixture := sqlitetest.Template(t, store, "Hub")
	got, err := sqlite.NewTemplateRepo(store).Get(t.Context(), fixture.ID)
	if err != nil || !reflect.DeepEqual(got, fixture) {
		t.Errorf("fixture round trip = %+v, %v\nwant %+v", got, err, fixture)
	}
}
