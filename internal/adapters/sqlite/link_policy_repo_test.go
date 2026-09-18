package sqlite_test

import (
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

func policy(name string, siteID *string, at time.Time) template.LinkPolicy {
	scope := template.ScopeGlobal
	if siteID != nil {
		scope = template.ScopeSite
	}
	return template.LinkPolicy{
		ID: id.New(), Scope: scope, SiteID: siteID, Name: name,
		Rules:          template.LinkRules{UpDepth: 2, DownLinks: true, SiblingMinWeight: 0.5, MaxLinks: 10, MaxPerTarget: 1, ParentLinkWithinParagraphs: 2, ChildrenSection: true},
		ForbidExternal: true, ForbidSelf: true, AnchorStrategy: template.AnchorPreferUser, CreatedAt: at, UpdatedAt: at,
	}
}

func policyNames(list []template.LinkPolicy) []string {
	out := make([]string, 0, len(list))
	for i := range list {
		out = append(out, list[i].Name)
	}
	return out
}

func TestLinkPolicyRepoRoundTrip(t *testing.T) {
	t.Parallel()

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")
	repo := sqlite.NewLinkPolicyRepo(store)
	want := policy("Default", nil, sqlitetest.Stamp)

	if err := repo.Insert(t.Context(), want); err != nil {
		t.Fatalf("Insert: %v", err)
	}
	got, err := repo.Get(t.Context(), want.ID)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("Get = %+v, %v\nwant %+v", got, err, want)
	}
	if err = repo.Insert(t.Context(), policy("default", nil, sqlitetest.Stamp)); !errors.IsCode(err, errors.Conflict) {
		t.Errorf("duplicate global name code = %q, want CONFLICT", errors.CodeOf(err))
	}
	if err = repo.Insert(t.Context(), policy("Default", &owner.ID, sqlitetest.Stamp)); err != nil {
		t.Fatalf("a site may reuse a global name: %v", err)
	}

	want.Name = "Relaxed"
	want.ForbidExternal = false
	want.AnchorStrategy = template.AnchorRotate
	want.Rules.MaxLinks = 30
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

func TestLinkPolicyRepoList(t *testing.T) {
	t.Parallel()

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")
	repo := sqlite.NewLinkPolicyRepo(store)

	for i, name := range []string{"Default", "loose", "Strict"} {
		if err := repo.Insert(t.Context(), policy(name, nil, sqlitetest.Stamp.Add(time.Duration(i)*time.Minute))); err != nil {
			t.Fatalf("Insert %s: %v", name, err)
		}
	}
	if err := repo.Insert(t.Context(), policy("Shop rules", &owner.ID, sqlitetest.Stamp.Add(time.Hour))); err != nil {
		t.Fatalf("Insert scoped: %v", err)
	}

	global := template.ScopeGlobal
	first, err := repo.List(t.Context(), template.PolicyQuery{Scope: &global, Sort: template.SortName}, paging.Request{Limit: 2})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if got := policyNames(first.Items); !reflect.DeepEqual(got, []string{"Default", "loose"}) || !first.HasMore {
		t.Fatalf("first page = %v", got)
	}
	rest, err := repo.List(t.Context(), template.PolicyQuery{Scope: &global, Sort: template.SortName}, paging.Request{After: first.Next, Limit: 2})
	if err != nil {
		t.Fatalf("List after: %v", err)
	}
	if got := policyNames(rest.Items); !reflect.DeepEqual(got, []string{"Strict"}) {
		t.Fatalf("second page = %v", got)
	}
	back, err := repo.List(t.Context(), template.PolicyQuery{Scope: &global, Sort: template.SortName}, paging.Request{Before: rest.Prev, Limit: 2})
	if err != nil {
		t.Fatalf("List before: %v", err)
	}
	if got := policyNames(back.Items); !reflect.DeepEqual(got, []string{"Default", "loose"}) {
		t.Fatalf("backward page = %v", got)
	}
	bySite, err := repo.List(t.Context(), template.PolicyQuery{SiteID: &owner.ID, Sort: template.SortCreatedAt}, paging.Request{Limit: 10})
	if err != nil || len(bySite.Items) != 1 || bySite.Items[0].Name != "Shop rules" {
		t.Errorf("by site = %+v, %v", bySite.Items, err)
	}
	byName, err := repo.List(t.Context(), template.PolicyQuery{Scope: &global, Name: "DEFAULT", Sort: template.SortCreatedAt}, paging.Request{Limit: 10})
	if err != nil || len(byName.Items) != 1 || byName.Items[0].Name != "Default" {
		t.Errorf("by name must be case-insensitive, got %+v, %v", byName.Items, err)
	}
}
