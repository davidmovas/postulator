package pagetypes_test

import (
	"slices"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/application/pagetypes"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func (h harness) rules(t *testing.T, siteID string) pagetypes.RulesResponse {
	t.Helper()
	listed, err := h.service.ListRules(t.Context(), pagetypes.ListRulesRequest{SiteID: siteID})
	if err != nil {
		t.Fatalf("ListRules: %v", err)
	}
	return listed
}

func numbers(rules []pagetypes.Rule) []int {
	out := make([]int, 0, len(rules))
	for i := range rules {
		out = append(out, rules[i].Number)
	}
	return out
}

func TestANewSiteGetsTheDefaultRulesAndTheFallback(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	listed := h.rules(t, h.siteID)
	if len(listed.Rules) != 14 || !slices.Equal(numbers(listed.Rules), []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14}) {
		t.Fatalf("the rules = %v, want fourteen numbered in order", numbers(listed.Rules))
	}
	info := h.typeOf(t, h.siteID, pagetype.Info)
	if listed.Fallback == nil || listed.Fallback.TypeID != info.ID || listed.Fallback.Kind != string(pagetype.KindFallback) {
		t.Fatalf("the fallback = %+v, want Info", listed.Fallback)
	}
	if first := listed.Rules[0]; first.TypeID != h.typeOf(t, h.siteID, pagetype.Technical).ID || first.Conditions.Technical == nil {
		t.Errorf("the first rule = %+v, want technical pages to Technical", first)
	}
	guide := h.typeOf(t, h.siteID, pagetype.Guide)
	if under := listed.Rules[10].Conditions.Ancestor; under == nil || under.TypeID != guide.ID {
		t.Errorf("rule 11 = %+v, want it under the site's own Guide", listed.Rules[10])
	}
}

func TestARuleIsAddedEditedMovedAndRemoved(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	guide := h.typeOf(t, h.siteID, pagetype.Guide)

	added, err := h.service.SaveRule(t.Context(), pagetypes.SaveRuleRequest{
		SiteID: h.siteID, Name: " Research notes ", Enabled: true, TypeID: guide.ID,
		Conditions: pagetype.Conditions{URLs: []pagetype.URLMatch{{Mode: pagetype.URLPrefix, Value: "Research/"}}},
	})
	if err != nil {
		t.Fatalf("SaveRule: %v", err)
	}
	if added.Number != 15 || added.Name != "Research notes" || added.Conditions.URLs[0].Value != "/research/" {
		t.Fatalf("the added rule = %+v, want it last, trimmed and its address normalized", added)
	}
	listed := h.rules(t, h.siteID)
	if listed.Rules[len(listed.Rules)-1].ID != added.ID || listed.Fallback == nil {
		t.Fatalf("the rules = %+v, want the new rule before the fallback", listed)
	}

	for name, req := range map[string]pagetypes.SaveRuleRequest{
		"a bad pattern": {SiteID: h.siteID, Name: "Bad", Enabled: true, TypeID: guide.ID, Conditions: pagetype.Conditions{
			URLs: []pagetype.URLMatch{{Mode: pagetype.URLGlob, Value: "/blog/[x]/"}},
		}},
		"no condition":           {SiteID: h.siteID, Name: "Everything", Enabled: true, TypeID: guide.ID},
		"another site's type":    {SiteID: h.siteID, Name: "Other", Enabled: true, TypeID: h.typeOf(t, h.otherID, pagetype.Guide).ID, Conditions: pagetype.Conditions{Slugs: []string{"x"}}},
		"a page of another site": {SiteID: h.siteID, Name: "Under", Enabled: true, TypeID: guide.ID, Conditions: pagetype.Conditions{Ancestor: &pagetype.Ancestor{PageID: sqlitetest.Page(t, h.store, h.otherID, "/elsewhere/").ID}}},
	} {
		if _, err = h.service.SaveRule(t.Context(), req); !errors.IsCode(err, errors.Invalid) {
			t.Errorf("%s = %v, want INVALID", name, err)
		}
	}

	added.Name, added.Enabled = "Research", false
	edited, err := h.service.SaveRule(t.Context(), pagetypes.SaveRuleRequest{
		ID: added.ID, SiteID: h.siteID, Name: added.Name, Enabled: added.Enabled, TypeID: added.TypeID, Conditions: added.Conditions,
	})
	if err != nil || edited.Name != "Research" || edited.Enabled || edited.Number != 15 {
		t.Fatalf("the edited rule = %+v %v", edited, err)
	}

	order := make([]string, 0, len(listed.Rules))
	for _, rule := range slices.Backward(listed.Rules) {
		order = append(order, rule.ID)
	}
	moved, err := h.service.ReorderRules(t.Context(), pagetypes.ReorderRulesRequest{SiteID: h.siteID, RuleIDs: order})
	if err != nil || moved.Rules[0].ID != added.ID || moved.Rules[14].ID != listed.Rules[0].ID {
		t.Fatalf("the reversed rules = %+v %v", numbers(moved.Rules), err)
	}
	if _, err = h.service.ReorderRules(t.Context(), pagetypes.ReorderRulesRequest{SiteID: h.siteID, RuleIDs: order[1:]}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("an order that leaves a rule out = %v, want INVALID", err)
	}

	if _, err = h.service.DeleteRule(t.Context(), pagetypes.DeleteRuleRequest{ID: added.ID}); err != nil {
		t.Fatalf("DeleteRule: %v", err)
	}
	after := h.rules(t, h.siteID)
	if len(after.Rules) != 14 || !slices.Equal(numbers(after.Rules), []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14}) {
		t.Fatalf("after the delete = %v, want the others renumbered", numbers(after.Rules))
	}
	if _, err = h.service.DeleteRule(t.Context(), pagetypes.DeleteRuleRequest{ID: after.Fallback.ID}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("deleting the fallback = %v, want INVALID", err)
	}
	if len(h.recorder.Events()) == 0 {
		t.Error("no event told the window")
	}
}

func TestTheFallbackKeepsNoConditionAndAnOrdinaryType(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	fallback := h.rules(t, h.siteID).Fallback
	guide, product := h.typeOf(t, h.siteID, pagetype.Guide), h.typeOf(t, h.siteID, pagetype.Product)

	moved, err := h.service.SaveRule(t.Context(), pagetypes.SaveRuleRequest{
		ID: fallback.ID, SiteID: h.siteID, Name: fallback.Name, Enabled: true, TypeID: guide.ID,
	})
	if err != nil || moved.TypeID != guide.ID {
		t.Fatalf("the fallback to Guide = %+v %v", moved, err)
	}
	for name, req := range map[string]pagetypes.SaveRuleRequest{
		"a condition": {ID: fallback.ID, SiteID: h.siteID, Name: fallback.Name, Enabled: true, TypeID: guide.ID, Conditions: pagetype.Conditions{Slugs: []string{"x"}}},
		"turned off":  {ID: fallback.ID, SiteID: h.siteID, Name: fallback.Name, TypeID: guide.ID},
		"a product":   {ID: fallback.ID, SiteID: h.siteID, Name: fallback.Name, Enabled: true, TypeID: product.ID},
	} {
		if _, err = h.service.SaveRule(t.Context(), req); !errors.IsCode(err, errors.Invalid) {
			t.Errorf("the fallback with %s = %v, want INVALID", name, err)
		}
	}
}

func TestEachRuleCountsThePagesItPlacedAndTheListCountsThoseTypedBeforeTheRules(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	listed := h.rules(t, h.siteID)
	posts := listed.Rules[7]
	guide, info, landing := h.typeOf(t, h.siteID, pagetype.Guide), h.typeOf(t, h.siteID, pagetype.Info), h.typeOf(t, h.siteID, pagetype.Landing)
	repo := sqlite.NewPageRepo(h.store)
	typed := func(path, typeID string, source pagemap.TypeSource, reason pagemap.TypeReason, ruleID string, status pagemap.Status) {
		t.Helper()
		page := sqlitetest.Page(t, h.store, h.siteID, path)
		page.Status = status
		if err := repo.Update(t.Context(), page); err != nil {
			t.Fatalf("archive %s: %v", path, err)
		}
		var err error
		if source == pagemap.TypeSourceManual {
			err = repo.SetType(t.Context(), page.ID, typeID, reason, page.UpdatedAt)
		} else {
			_, err = repo.Classify(t.Context(), page.ID, pagetype.Assignment{},
				pagetype.Assignment{TypeID: typeID, Source: source, Reason: reason, RuleID: ruleID}, page.UpdatedAt)
		}
		if err != nil {
			t.Fatalf("type %s: %v", path, err)
		}
	}
	typed("/blog/first/", guide.ID, pagemap.TypeSourceAuto, pagemap.TypeByRule, posts.ID, pagemap.StatusExists)
	typed("/blog/second/", guide.ID, pagemap.TypeSourceAuto, pagemap.TypeByRule, posts.ID, pagemap.StatusExists)
	typed("/blog/gone/", guide.ID, pagemap.TypeSourceAuto, pagemap.TypeByRule, posts.ID, pagemap.StatusArchived)
	typed("/about/", info.ID, pagemap.TypeSourceAuto, pagemap.TypeByUnmatched, "", pagemap.StatusExists)
	typed("/old-hub/", landing.ID, pagemap.TypeSourceAuto, pagemap.TypeByListing, "", pagemap.StatusExists)
	typed("/old-page/", info.ID, pagemap.TypeSourceAuto, pagemap.TypeByDefault, "", pagemap.StatusExists)
	typed("/mine/", info.ID, pagemap.TypeSourceManual, pagemap.TypeByUser, "", pagemap.StatusExists)

	listed = h.rules(t, h.siteID)
	for i := range listed.Rules {
		want := 0
		if listed.Rules[i].ID == posts.ID {
			want = 2
		}
		if listed.Rules[i].Pages != want {
			t.Errorf("rule %d counts %d pages, want %d", listed.Rules[i].Number, listed.Rules[i].Pages, want)
		}
	}
	if listed.Fallback == nil || listed.Fallback.Pages != 1 {
		t.Errorf("the fallback = %+v, want the one page no rule matched", listed.Fallback)
	}
	if listed.BeforeRules != 2 {
		t.Errorf("pages typed before the rules = %d, want the listing and the default one", listed.BeforeRules)
	}
}

func TestDeletingATypeMovesItsRulesToTheTypeItsPagesMoveTo(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	guide, info := h.typeOf(t, h.siteID, pagetype.Guide), h.typeOf(t, h.siteID, pagetype.Info)

	if _, err := h.service.Delete(t.Context(), pagetypes.DeleteRequest{ID: guide.ID, ReassignTo: info.ID}); err != nil {
		t.Fatalf("Delete Guide: %v", err)
	}
	listed := h.rules(t, h.siteID)
	for i := range listed.Rules {
		if listed.Rules[i].TypeID == guide.ID {
			t.Errorf("rule %d still gives the deleted Guide", listed.Rules[i].Number)
		}
	}
	if under := listed.Rules[10].Conditions.Ancestor; under == nil || under.TypeID != info.ID {
		t.Errorf("the rule under a guide = %+v, want it under the type the guides moved to", listed.Rules[10])
	}

	notes, err := h.service.Create(t.Context(), pagetypes.CreateRequest{SiteID: h.siteID, Name: "Notes", Preset: string(pagetype.Guide)})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	product := h.typeOf(t, h.siteID, pagetype.Product)
	_, err = h.service.Delete(t.Context(), pagetypes.DeleteRequest{ID: info.ID, ReassignTo: product.ID})
	if name, reason := reasonOf(t, err); name != "reassignTo" || reason != "fallback_role" {
		t.Fatalf("moving the fallback onto Product = %v (%s %s), want it refused", err, name, reason)
	}
	if _, err = h.service.Delete(t.Context(), pagetypes.DeleteRequest{ID: info.ID, ReassignTo: notes.ID}); err != nil {
		t.Fatalf("Delete Info into Notes: %v", err)
	}
	if fallback := h.rules(t, h.siteID).Fallback; fallback == nil || fallback.TypeID != notes.ID {
		t.Errorf("the fallback = %+v, want Notes", fallback)
	}
}
