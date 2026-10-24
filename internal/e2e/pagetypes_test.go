//go:build e2e

package e2e_test

import (
	"slices"
	"strconv"
	"testing"

	"github.com/davidmovas/postulator/internal/app"
	"github.com/davidmovas/postulator/internal/application/observe"
	"github.com/davidmovas/postulator/internal/application/pagetypes"
	"github.com/davidmovas/postulator/internal/application/sites"
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
	if len(types) != len(pagetype.Presets()) {
		t.Errorf("the first observation left %d types, want one per preset", len(types))
	}
	for _, preset := range pagetype.Presets() {
		if kind, seeded := types[preset]; !seeded || kind.Name != preset.DefaultName() {
			t.Errorf("the %s type is %+v, want it seeded as %q", preset, kind, preset.DefaultName())
		}
	}

	models := modelsOf(t, core, siteID)
	for path := range models {
		if want, identified := identityType(models[path].Page); identified {
			assertTyped(t, path, models[path].Page, types, want)
		}
	}
	for path, want := range expected.types {
		model, held := models[path]
		if !held {
			t.Errorf("the model holds no %s", path)
			continue
		}
		assertTyped(t, path, model.Page, types, want)
	}

	edit := pagetypes.SetScopeRequest{
		PageTypeID: types[pagetype.Hub].ID, Field: string(field.Body), Enabled: true, AIWrites: true,
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
		t.Fatalf("edit the Hub type's body scope: %v", err)
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

func assertTyped(t *testing.T, path string, page pagemap.Page, types map[pagetype.Preset]pagetypes.Type, want expectedType) {
	t.Helper()

	kind := types[want.preset]
	if page.PageTypeID == nil || *page.PageTypeID != kind.ID || page.TypeSource != pagemap.TypeSourceAuto || page.TypeReason != want.reason {
		t.Errorf("%s (%s, technical %q) is typed %s by %q %q, want %s by auto %s", path, page.WPType, page.TechnicalReason,
			typeName(types, page.PageTypeID), page.TypeSource, page.TypeReason, kind.Name, want.reason)
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
