//go:build e2e

package e2e_test

import (
	"cmp"
	"maps"
	"slices"
	"strconv"
	"testing"

	"github.com/davidmovas/postulator/internal/app"
	"github.com/davidmovas/postulator/internal/application/graph"
	"github.com/davidmovas/postulator/internal/application/observe"
	"github.com/davidmovas/postulator/internal/application/pages"
	"github.com/davidmovas/postulator/internal/application/pagetypes"
	"github.com/davidmovas/postulator/internal/application/runs"
	"github.com/davidmovas/postulator/internal/application/sites"
	"github.com/davidmovas/postulator/internal/application/templates"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
)

const (
	subtitleKey   = "page_subtitle"
	subtitleLabel = "Page subtitle"
	subtitleValue = "Read this before the first order arrives"
)

func TestThePageTypesFollowTheSite(t *testing.T) {
	live := newSite(t)
	requirePlugin(t, live.env, true)
	expected := modelOf(t, live.env)

	core := openCore(t)
	owner, err := core.Sites.Create(t.Context(), sites.CreateRequest{
		Name: "Typed Shop", BaseURL: live.env.baseURL, Username: live.env.user, Password: live.env.pass, AllowInsecure: true,
	})
	if err != nil {
		t.Fatalf("create the site: %v", err)
	}
	siteID := owner.Site.ID
	runSync(t, core, siteID)

	types := typesOf(t, core, siteID)
	if len(types) != len(pagetype.Seeded()) {
		t.Errorf("the first observation left %d types, want one per seeded preset", len(types))
	}
	for _, preset := range pagetype.Seeded() {
		if kind, seeded := types[preset]; !seeded || kind.Name != preset.DefaultName() {
			t.Errorf("the %s type is %+v, want it seeded as %q", preset, kind, preset.DefaultName())
		}
	}

	rules := rulesOf(t, core, siteID)
	models := modelsOf(t, core, siteID)
	census := map[string][]string{}
	for path := range models {
		page := models[path].Page
		if page.WPID == nil || page.Status == pagemap.StatusArchived {
			continue
		}
		if page.PageTypeID == nil {
			t.Errorf("%s (%s) is on the site and has no type after the first observation", path, page.WPType)
			continue
		}
		group := typeName(types, page.PageTypeID) + " by " + cmp.Or(rules[page.TypeRuleID], string(page.TypeReason))
		census[group] = append(census[group], path)
		if want, identified := identityType(page); identified {
			assertTyped(t, path, page, types, rules, want)
		}
		if page.TypeReason == pagemap.TypeByUnmatched && expected.types[path] != typedByDefault {
			t.Errorf("%s fell to %s because no rule matched it, and the stand names no such page", path, typeName(types, page.PageTypeID))
		}
	}
	for _, group := range slices.Sorted(maps.Keys(census)) {
		slices.Sort(census[group])
		t.Logf("%s: %d pages %v", group, len(census[group]), census[group])
	}
	for path, want := range expected.types {
		model, held := models[path]
		if !held {
			t.Errorf("the model holds no %s", path)
			continue
		}
		assertTyped(t, path, model.Page, types, rules, want)
	}
	assertNothingToRetype(t, core, siteID, "after the first observation")
	assertTemplatesFollowTheTypes(t, core, siteID, models, expected.templates)

	edit := pagetypes.SetScopeRequest{
		PageTypeID: types[pagetype.Category].ID, Field: string(field.Body), Enabled: true, AIWrites: true,
		Instruction: "Open with what the listed products share, then say what sets each apart.",
		MinLength:   300, MaxLength: 1800,
		Links: field.LinkPolicy{
			Links: true,
			Rules: field.LinkRules{
				UpDepth: 2, DownLinks: true, SiblingMinWeight: 0.6, MaxLinks: 8, MaxPerTarget: 1,
				ParentLinkWithinParagraphs: 2, ChildrenSection: true,
			},
			ForbidExternal: true, ForbidSelf: true, AnchorStrategy: field.AnchorRotate,
		},
	}
	if _, err = core.PageTypes.SetScope(t.Context(), edit); err != nil {
		t.Fatalf("edit the Category type's body scope: %v", err)
	}

	holder, held := models[expected.custom]
	if !held || holder.Page.WPID == nil {
		t.Fatalf("the model holds no %s on the site to carry a custom field", expected.custom)
	}
	set, err := core.PageTypes.Set(t.Context(), siteID)
	if err != nil {
		t.Fatalf("load the site's types: %v", err)
	}
	custom := pagetypes.SetScopeRequest{
		PageTypeID: set.TypeOf(holder.Page).ID, Field: string(field.Instance(field.Meta, subtitleKey)), Enabled: true,
		Links: field.NoLinks(), Label: subtitleLabel, Provider: string(field.ProviderPlugin),
	}
	if _, err = core.PageTypes.SetScope(t.Context(), custom); err != nil {
		t.Fatalf("enable the custom field on the type of %s: %v", expected.custom, err)
	}

	wpID := strconv.FormatInt(*holder.Page.WPID, 10)
	wpCLI(t, live.env, "post", "meta", "update", wpID, subtitleKey, subtitleValue)
	t.Cleanup(func() { wpCLI(t, live.env, "post", "meta", "delete", wpID, subtitleKey) })

	runSync(t, core, siteID)

	after := typesOf(t, core, siteID)
	edited := []pagetypes.SetScopeRequest{edit, custom}
	for i := range edited {
		want := &edited[i]
		var kind pagetypes.Type
		for preset := range after {
			if after[preset].ID == want.PageTypeID {
				kind = after[preset]
			}
		}
		if scope, found := scopeIn(kind, field.Key(want.Field)); !found || !scope.Stored || !sameScope(scope, want) {
			t.Errorf("after another observation the %s scope of %s is %+v, want %+v", want.Field, kind.Name, scope, *want)
		}
	}

	read, err := core.Observe.Page(t.Context(), holder.Page.ID)
	if err != nil {
		t.Fatalf("read the model of %s: %v", expected.custom, err)
	}
	if !slices.ContainsFunc(read.Fields, func(value field.Value) bool {
		return value.Key == field.Instance(field.Meta, subtitleKey) && value.Current == subtitleValue
	}) {
		t.Errorf("%s holds the fields %+v, want %s read as %q", expected.custom, read.Fields, subtitleKey, subtitleValue)
	}

	again := modelsOf(t, core, siteID)
	for path := range models {
		before, now := models[path].Page, again[path].Page
		if before.PageTypeID == nil {
			continue
		}
		if now.PageTypeID == nil || *now.PageTypeID != *before.PageTypeID || now.TypeSource != before.TypeSource ||
			now.TypeReason != before.TypeReason {
			t.Errorf("the second observation moved %s from %s by %s %s to %s by %q %q", path, typeName(after, before.PageTypeID),
				before.TypeSource, before.TypeReason, typeName(after, now.PageTypeID), now.TypeSource, now.TypeReason)
		}
	}
	assertNothingToRetype(t, core, siteID, "after the second observation")
}

func rulesOf(t *testing.T, core *app.Core, siteID string) map[string]string {
	t.Helper()

	listed, err := core.PageTypes.ListRules(t.Context(), pagetypes.ListRulesRequest{SiteID: siteID})
	if err != nil {
		t.Fatalf("list the classification rules: %v", err)
	}
	out := make(map[string]string, len(listed.Rules)+1)
	for i := range listed.Rules {
		out[listed.Rules[i].ID] = listed.Rules[i].Name
	}
	if listed.Fallback != nil {
		out[listed.Fallback.ID] = listed.Fallback.Name
	}
	return out
}

func assertNothingToRetype(t *testing.T, core *app.Core, siteID, when string) {
	t.Helper()

	preview, err := core.Classification.Preview(t.Context(), pagetypes.PreviewRequest{SiteID: siteID})
	if err != nil {
		t.Fatalf("preview the classification %s: %v", when, err)
	}
	if preview.Moving != 0 || preview.Refreshed != 0 || len(preview.Groups) != 0 {
		t.Errorf("the preview %s = %+v, want nothing the sync left for an apply", when, preview)
	}
	applied, err := core.Classification.Apply(t.Context(), pagetypes.ApplyRequest{SiteID: siteID, Hash: preview.Hash})
	if err != nil || applied.Moved != 0 {
		t.Errorf("applying the empty preview %s = %+v %v, want nothing moved", when, applied, err)
	}
}

func assertTemplatesFollowTheTypes(t *testing.T, core *app.Core, siteID string, models map[string]observe.PageModel, want map[string]string) {
	t.Helper()

	listed, err := core.PageTypes.List(t.Context(), pagetypes.ListRequest{SiteID: siteID})
	if err != nil {
		t.Fatalf("list the page types: %v", err)
	}
	targets := make([]string, 0, len(want))
	for _, path := range slices.Sorted(maps.Keys(want)) {
		page := models[path].Page
		targets = append(targets, page.ID)
		if page.EntityID != nil {
			continue
		}
		created, createErr := core.Graph.CreateEntity(t.Context(), graph.CreateEntityRequest{SiteID: siteID, Name: page.Title})
		if createErr != nil {
			t.Fatalf("create the entity of %s, which the site's breadcrumbs left without one: %v", path, createErr)
		}
		if _, mapErr := core.Pages.MapToEntity(t.Context(), pages.MapToEntityRequest{PageID: page.ID, EntityID: created.Entity.ID}); mapErr != nil {
			t.Fatalf("map %s to its entity: %v", path, mapErr)
		}
	}
	for i := range listed.Types {
		kind := &listed.Types[i]
		if kind.Suggested == nil || kind.TemplateID != nil {
			continue
		}
		if _, err = core.PageTypes.Save(t.Context(), pagetypes.SaveRequest{ID: kind.ID, TemplateID: kind.Suggested}); err != nil {
			t.Fatalf("give %s its suggested template: %v", kind.Name, err)
		}
	}
	for path, name := range want {
		resolved, resolveErr := core.Templates.ResolveForPage(t.Context(), templates.ResolveForPageRequest{PageID: models[path].Page.ID})
		if resolveErr != nil || resolved.Name != name || resolved.Source != templates.SourceType {
			t.Errorf("a run on auto resolves %s to %q by %q (%v), want its type's %s", path, resolved.Name, resolved.Source, resolveErr, name)
		}
	}
	if _, err = core.Runs.Estimate(t.Context(), runs.StartRequest{SiteID: siteID, PageIDs: targets}); err != nil {
		t.Errorf("estimate a run on auto over %v: %v", slices.Sorted(maps.Keys(want)), err)
	}
	for path := range want {
		page, getErr := core.Pages.Get(t.Context(), pages.GetRequest{ID: models[path].Page.ID})
		if getErr != nil || page.Page.TemplateID != nil {
			t.Errorf("%s after the estimate = %+v %v, want it still following its type", path, page.Page.TemplateID, getErr)
		}
	}
}

func typesOf(t *testing.T, core *app.Core, siteID string) map[pagetype.Preset]pagetypes.Type {
	t.Helper()

	listed, err := core.PageTypes.List(t.Context(), pagetypes.ListRequest{SiteID: siteID})
	if err != nil {
		t.Fatalf("list the page types: %v", err)
	}
	out := make(map[pagetype.Preset]pagetypes.Type, len(listed.Types))
	for i := range listed.Types {
		preset := pagetype.Preset(listed.Types[i].Preset)
		if _, twice := out[preset]; twice {
			t.Errorf("the site carries two %s types", preset)
		}
		out[preset] = listed.Types[i]
	}
	return out
}

func modelsOf(t *testing.T, core *app.Core, siteID string) map[string]observe.PageModel {
	t.Helper()

	stored := pagesByPath(t, core.Pages, siteID)
	out := make(map[string]observe.PageModel, len(stored))
	for path := range stored {
		model, err := core.Observe.Page(t.Context(), stored[path].ID)
		if err != nil {
			t.Fatalf("read the model of %s: %v", path, err)
		}
		out[path] = model
	}
	return out
}

func identityType(page pagemap.Page) (expectedType, bool) {
	switch {
	case page.WPID == nil || page.Status == pagemap.StatusArchived:
		return expectedType{}, false
	case page.WPType == pagemap.WPProduct:
		return typedProduct, true
	case page.WPType.Term():
		return typedStoreCategory, true
	case page.TechnicalReason == pagemap.TechnicalShop:
		return typedCatalog, true
	case page.TechnicalReason != pagemap.TechnicalNone:
		return typedTechnical, true
	default:
		return expectedType{}, false
	}
}

func assertTyped(t *testing.T, path string, page pagemap.Page, types map[pagetype.Preset]pagetypes.Type, rules map[string]string,
	want expectedType) {
	t.Helper()

	kind, reason := types[want.preset], pagemap.TypeByRule
	if want.rule == "" {
		reason = pagemap.TypeByUnmatched
	}
	if page.PageTypeID == nil || *page.PageTypeID != kind.ID || page.TypeSource != pagemap.TypeSourceAuto || page.TypeReason != reason ||
		want.rule != "" && rules[page.TypeRuleID] != want.rule {
		t.Errorf("%s (%s, technical %q) is typed %s by %q %q under %q, want %s by %s under %q", path, page.WPType, page.TechnicalReason,
			typeName(types, page.PageTypeID), page.TypeSource, page.TypeReason, rules[page.TypeRuleID], kind.Name, reason, want.rule)
	}
}

func typeName(types map[pagetype.Preset]pagetypes.Type, typeID *string) string {
	if typeID == nil {
		return "<none>"
	}
	for preset := range types {
		if types[preset].ID == *typeID {
			return types[preset].Name
		}
	}
	return *typeID
}

func scopeIn(kind pagetypes.Type, key field.Key) (pagetypes.Scope, bool) {
	for i := range kind.Scopes {
		if kind.Scopes[i].Field == string(key) {
			return kind.Scopes[i], true
		}
	}
	return pagetypes.Scope{}, false
}

func sameScope(scope pagetypes.Scope, want *pagetypes.SetScopeRequest) bool {
	return scope.Field == want.Field && scope.Enabled == want.Enabled && scope.AIWrites == want.AIWrites &&
		scope.Instruction == want.Instruction && scope.MinLength == want.MinLength && scope.MaxLength == want.MaxLength &&
		scope.Rewrites == want.Rewrites && scope.Links == want.Links && scope.Label == want.Label && scope.Provider == want.Provider
}
