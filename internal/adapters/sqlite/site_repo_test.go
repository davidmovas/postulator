package sqlite_test

import (
	stderrors "errors"
	"reflect"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/domain/site"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

func fullSite(name string, at time.Time) site.Site {
	record := site.Site{
		ID: id.New(), Name: name, BaseURL: "https://" + name + ".example.com", Username: "editor", Status: site.StatusPaused, AllowInsecure: true,
		Plugin:    site.PluginState{Installed: true, Version: "1.2.0", Capabilities: []string{"bulk", "seo_meta"}, SEOPlugin: "yoast"},
		Defaults:  site.Defaults{ModelProfiles: map[llm.Role]llm.ModelRef{llm.RoleWriter: {Provider: "openai", Model: "gpt"}}},
		CreatedAt: at, UpdatedAt: at,
	}
	record.SecretRef = site.SecretRef(record.ID)
	return record
}

func detailOf(t *testing.T, err error, key string) any {
	t.Helper()
	var kernel *errors.Error
	if !stderrors.As(err, &kernel) {
		t.Fatalf("error %v is not a kernel error", err)
	}
	return kernel.Details[key]
}

func siteNames(list []site.Site) []string {
	out := make([]string, 0, len(list))
	for i := range list {
		out = append(out, list[i].Name)
	}
	return out
}

func TestSiteRepoRoundTrip(t *testing.T) {
	t.Parallel()

	store := sqlitetest.Open(t)
	repo := sqlite.NewSiteRepo(store)
	want := fullSite("shop", sqlitetest.Stamp)

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

	if err = repo.Insert(t.Context(), want); !errors.IsCode(err, errors.Conflict) {
		t.Errorf("second Insert code = %q, want CONFLICT", errors.CodeOf(err))
	}

	want.Name = "Shop Two"
	want.Status = site.StatusActive
	templateID := "t1"
	want.Defaults.TemplateID = &templateID
	want.UpdatedAt = sqlitetest.Stamp.Add(time.Hour)
	if err = repo.Update(t.Context(), want); !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("Update with an unknown template must fail the foreign key as INVALID, got %v", err)
	}
	want.Defaults.TemplateID = nil
	if err = repo.Update(t.Context(), want); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got, err = repo.Get(t.Context(), want.ID)
	if err != nil {
		t.Fatalf("Get after update: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Get after update = %+v\nwant %+v", got, want)
	}

	if err = repo.Delete(t.Context(), want.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err = repo.Get(t.Context(), want.ID); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("Get after delete code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
	if detail := detailOf(t, err, "siteId"); detail != want.ID {
		t.Errorf("siteId detail = %v", detail)
	}
	if err = repo.Delete(t.Context(), want.ID); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("Delete twice code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
	if err = repo.Update(t.Context(), want); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("Update after delete code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
}

func TestSiteRepoListPagesForwardAndBackward(t *testing.T) {
	t.Parallel()

	store := sqlitetest.Open(t)
	repo := sqlite.NewSiteRepo(store)
	names := []string{"alpha", "Bravo", "charlie", "Delta", "echo"}
	for i, name := range names {
		record := fullSite(name, sqlitetest.Stamp.Add(time.Duration(i)*time.Hour))
		if i%2 == 0 {
			record.Status = site.StatusActive
		}
		if err := repo.Insert(t.Context(), record); err != nil {
			t.Fatalf("Insert %s: %v", name, err)
		}
	}

	first, err := repo.List(t.Context(), site.Query{Sort: site.SortCreatedAt}, paging.Request{Limit: 2})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if got := siteNames(first.Items); !reflect.DeepEqual(got, []string{"alpha", "Bravo"}) || !first.HasMore || first.Next == "" || first.Prev != "" {
		t.Fatalf("first page = %v, hasMore %v, next %q, prev %q", got, first.HasMore, first.Next, first.Prev)
	}

	second, err := repo.List(t.Context(), site.Query{Sort: site.SortCreatedAt}, paging.Request{After: first.Next, Limit: 2})
	if err != nil {
		t.Fatalf("List after: %v", err)
	}
	if got := siteNames(second.Items); !reflect.DeepEqual(got, []string{"charlie", "Delta"}) || !second.HasMore || second.Prev == "" {
		t.Fatalf("second page = %v, hasMore %v, prev %q", got, second.HasMore, second.Prev)
	}

	back, err := repo.List(t.Context(), site.Query{Sort: site.SortCreatedAt}, paging.Request{Before: second.Prev, Limit: 2})
	if err != nil {
		t.Fatalf("List before: %v", err)
	}
	if got := siteNames(back.Items); !reflect.DeepEqual(got, []string{"alpha", "Bravo"}) || back.HasMore {
		t.Fatalf("backward page = %v, hasMore %v", got, back.HasMore)
	}

	last, err := repo.List(t.Context(), site.Query{Sort: site.SortCreatedAt}, paging.Request{After: second.Next, Limit: 2})
	if err != nil {
		t.Fatalf("List last: %v", err)
	}
	if got := siteNames(last.Items); !reflect.DeepEqual(got, []string{"echo"}) || last.HasMore || last.Next != "" {
		t.Fatalf("last page = %v, hasMore %v, next %q", got, last.HasMore, last.Next)
	}

	byName, err := repo.List(t.Context(), site.Query{Sort: site.SortName}, paging.Request{Limit: 10})
	if err != nil {
		t.Fatalf("List by name: %v", err)
	}
	if got := siteNames(byName.Items); !reflect.DeepEqual(got, []string{"alpha", "Bravo", "charlie", "Delta", "echo"}) {
		t.Errorf("name order must be case-insensitive, got %v", got)
	}

	descending, err := repo.List(t.Context(), site.Query{Sort: site.SortName, Desc: true}, paging.Request{Limit: 2})
	if err != nil {
		t.Fatalf("List desc: %v", err)
	}
	if got := siteNames(descending.Items); !reflect.DeepEqual(got, []string{"echo", "Delta"}) {
		t.Errorf("descending = %v", got)
	}

	active := site.StatusActive
	filtered, err := repo.List(t.Context(), site.Query{Status: &active, Sort: site.SortCreatedAt}, paging.Request{Limit: 10})
	if err != nil {
		t.Fatalf("List filtered: %v", err)
	}
	if got := siteNames(filtered.Items); !reflect.DeepEqual(got, []string{"alpha", "charlie", "echo"}) {
		t.Errorf("filtered = %v", got)
	}

	if _, err = repo.List(t.Context(), site.Query{Sort: site.SortName}, paging.Request{After: first.Next, Limit: 2}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("a cursor from another sort must be INVALID, got %v", err)
	}
}

func TestSiteFixture(t *testing.T) {
	t.Parallel()

	store := sqlitetest.Open(t)
	fixture := sqlitetest.Site(t, store, "shop")
	got, err := sqlite.NewSiteRepo(store).Get(t.Context(), fixture.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !reflect.DeepEqual(got, fixture) {
		t.Errorf("fixture round trip = %+v\nwant %+v", got, fixture)
	}
}
