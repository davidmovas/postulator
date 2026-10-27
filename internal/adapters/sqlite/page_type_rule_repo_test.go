package sqlite_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

func TestPageTypeRuleRepoRoundTrip(t *testing.T) {
	t.Parallel()

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")
	types := sqlite.NewPageTypeRepo(store)
	guide := pagetype.Type{ID: id.New(), SiteID: owner.ID, Name: "Guide", Preset: pagetype.Guide, CreatedAt: sqlitetest.Stamp, UpdatedAt: sqlitetest.Stamp}
	info := pagetype.Type{ID: id.New(), SiteID: owner.ID, Name: "Info", Preset: pagetype.Info, CreatedAt: sqlitetest.Stamp, UpdatedAt: sqlitetest.Stamp}
	for _, kind := range []pagetype.Type{guide, info} {
		if err := types.Insert(t.Context(), kind); err != nil {
			t.Fatalf("Insert %s: %v", kind.Name, err)
		}
	}

	repo := sqlite.NewPageTypeRuleRepo(store)
	words, yes := 800, true
	rule := pagetype.Rule{
		ID: id.New(), SiteID: owner.ID, Kind: pagetype.KindRule, Position: 1, Name: "Long articles", Enabled: true, TypeID: guide.ID,
		Conditions: pagetype.Conditions{
			URLs:      []pagetype.URLMatch{{Mode: pagetype.URLGlob, Value: "/blog/**"}},
			Ancestor:  &pagetype.Ancestor{TypeID: guide.ID},
			Words:     &pagetype.Range{Min: &words},
			Listing:   &pagetype.Listing{},
			Technical: &yes,
		},
		CreatedAt: sqlitetest.Stamp, UpdatedAt: sqlitetest.Stamp,
	}
	fallback := pagetype.Rule{
		ID: id.New(), SiteID: owner.ID, Kind: pagetype.KindFallback, Position: 2, Name: pagetype.FallbackName, Enabled: true,
		TypeID: info.ID, CreatedAt: sqlitetest.Stamp, UpdatedAt: sqlitetest.Stamp,
	}
	for _, held := range []pagetype.Rule{fallback, rule} {
		if err := repo.Insert(t.Context(), held); err != nil {
			t.Fatalf("Insert %s: %v", held.Name, err)
		}
	}
	got, err := repo.Get(t.Context(), rule.ID)
	if err != nil || !reflect.DeepEqual(got, rule) {
		t.Fatalf("Get = %+v %v, want %+v", got, err, rule)
	}
	listed, err := repo.ListBySite(t.Context(), owner.ID)
	if err != nil || len(listed) != 2 || listed[0].ID != rule.ID || listed[1].ID != fallback.ID {
		t.Fatalf("ListBySite = %+v %v, want the rules by their place", listed, err)
	}

	second := fallback
	second.ID = id.New()
	if err = repo.Insert(t.Context(), second); !errors.IsCode(err, errors.Conflict) {
		t.Errorf("a second fallback = %q, want CONFLICT", errors.CodeOf(err))
	}

	rule.Enabled, rule.Position, rule.Name, rule.TypeID = false, 3, "Long reads", info.ID
	rule.UpdatedAt = sqlitetest.Stamp.Add(time.Hour)
	if err = repo.Update(t.Context(), rule); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got, err = repo.Get(t.Context(), rule.ID); err != nil || !reflect.DeepEqual(got, rule) {
		t.Fatalf("after Update = %+v %v, want %+v", got, err, rule)
	}
	if err = types.Delete(t.Context(), info.ID); err == nil {
		t.Error("a type two rules still give was deleted")
	}

	if err = repo.Delete(t.Context(), rule.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err = repo.Get(t.Context(), rule.ID); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("a deleted rule = %q, want NOT_FOUND", errors.CodeOf(err))
	}
	if err = repo.Delete(t.Context(), rule.ID); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("deleting it twice = %q, want NOT_FOUND", errors.CodeOf(err))
	}

	if err = sqlite.NewSiteRepo(store).Delete(t.Context(), owner.ID); err != nil {
		t.Fatalf("delete the site with its types and rules: %v", err)
	}
	if left, listErr := repo.ListBySite(t.Context(), owner.ID); listErr != nil || len(left) != 0 {
		t.Errorf("the site's rules after it went = %+v %v", left, listErr)
	}
}

func TestAPageTypedByARuleIsWrittenOnlyOverTheTypeItWasReadWith(t *testing.T) {
	t.Parallel()

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")
	types := sqlite.NewPageTypeRepo(store)
	guide := pagetype.Type{ID: id.New(), SiteID: owner.ID, Name: "Guide", Preset: pagetype.Guide, CreatedAt: sqlitetest.Stamp, UpdatedAt: sqlitetest.Stamp}
	info := pagetype.Type{ID: id.New(), SiteID: owner.ID, Name: "Info", Preset: pagetype.Info, CreatedAt: sqlitetest.Stamp, UpdatedAt: sqlitetest.Stamp}
	for _, kind := range []pagetype.Type{guide, info} {
		if err := types.Insert(t.Context(), kind); err != nil {
			t.Fatalf("Insert %s: %v", kind.Name, err)
		}
	}
	pages := sqlite.NewPageRepo(store)
	page := sqlitetest.Page(t, store, owner.ID, "/lab-guides/")
	at := sqlitetest.Stamp.Add(time.Hour)

	byRule := pagetype.Assignment{TypeID: guide.ID, Source: pagemap.TypeSourceAuto, Reason: pagemap.TypeByRule, RuleID: "rule-10"}
	moved, err := pages.Classify(t.Context(), page.ID, pagetype.Assignment{}, byRule, at)
	if err != nil || !moved {
		t.Fatalf("Classify an untyped page = %t %v", moved, err)
	}
	stored, err := pages.Get(t.Context(), page.ID)
	if err != nil || pagetype.AssignmentOf(stored) != byRule {
		t.Fatalf("the page = %+v %v, want %+v", pagetype.AssignmentOf(stored), err, byRule)
	}

	unmatched := pagetype.Assignment{TypeID: info.ID, Source: pagemap.TypeSourceAuto, Reason: pagemap.TypeByUnmatched}
	if moved, err = pages.Classify(t.Context(), page.ID, pagetype.Assignment{}, unmatched, at); err != nil || moved {
		t.Fatalf("a write over a type it was not read with = %t %v, want it skipped", moved, err)
	}
	if err = pages.SetType(t.Context(), page.ID, info.ID, pagemap.TypeByUser, at); err != nil {
		t.Fatalf("SetType: %v", err)
	}
	if moved, err = pages.Classify(t.Context(), page.ID, byRule, unmatched, at); err != nil || moved {
		t.Fatalf("a rule's write over a person's type = %t %v, want it skipped", moved, err)
	}
	stored, err = pages.Get(t.Context(), page.ID)
	if err != nil || stored.TypeSource != pagemap.TypeSourceManual || stored.TypeRuleID != "" || *stored.PageTypeID != info.ID {
		t.Fatalf("the page a person typed = %+v %v, want the person's type and no rule", stored, err)
	}
}
