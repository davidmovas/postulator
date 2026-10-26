package pagetypes_test

import (
	stderrors "errors"
	"slices"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/application/applicationtest"
	"github.com/davidmovas/postulator/internal/application/pagetypes"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type harness struct {
	service  *pagetypes.Service
	store    *sqlite.Store
	recorder *applicationtest.Recorder
	siteID   string
	otherID  string
}

func newHarness(t *testing.T) harness {
	t.Helper()

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")
	other := sqlitetest.Site(t, store, "other")
	recorder := &applicationtest.Recorder{}
	service := pagetypes.New(pagetypes.Deps{
		Types: sqlite.NewPageTypeRepo(store), Scopes: sqlite.NewFieldScopeRepo(store), Rules: sqlite.NewPageTypeRuleRepo(store),
		Pages: sqlite.NewPageRepo(store), Templates: sqlite.NewTemplateRepo(store), Sites: sqlite.NewSiteRepo(store), UnitOfWork: store,
		Publisher: recorder, Clock: clock.NewFake(time.Date(2026, time.October, 18, 9, 0, 0, 0, time.UTC)),
	})
	h := harness{service: service, store: store, recorder: recorder, siteID: owner.ID, otherID: other.ID}
	for _, site := range []string{owner.ID, other.ID} {
		if err := service.Seed(t.Context(), site); err != nil {
			t.Fatalf("Seed %s: %v", site, err)
		}
	}
	return h
}

func (h harness) list(t *testing.T, siteID string) pagetypes.ListResponse {
	t.Helper()
	listed, err := h.service.List(t.Context(), pagetypes.ListRequest{SiteID: siteID})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	return listed
}

func (h harness) typeOf(t *testing.T, siteID string, preset pagetype.Preset) pagetypes.Type {
	t.Helper()
	types := h.list(t, siteID).Types
	for i := range types {
		if types[i].Preset == string(preset) {
			return types[i]
		}
	}
	t.Fatalf("no %s type on %s", preset, siteID)
	return pagetypes.Type{}
}

func (h harness) typeNamed(t *testing.T, siteID, name string) pagetypes.Type {
	t.Helper()
	types := h.list(t, siteID).Types
	for i := range types {
		if types[i].Name == name {
			return types[i]
		}
	}
	t.Fatalf("no type %s on %s", name, siteID)
	return pagetypes.Type{}
}

func scopeOf(kind pagetypes.Type, key field.Key) pagetypes.Scope {
	for i := range kind.Scopes {
		if kind.Scopes[i].Field == string(key) {
			return kind.Scopes[i]
		}
	}
	return pagetypes.Scope{}
}

func reasonOf(t *testing.T, err error) (name, reason string) {
	t.Helper()
	var kerr *errors.Error
	if !stderrors.As(err, &kerr) {
		t.Fatalf("err = %v, want a kernel error", err)
	}
	if held, ok := kerr.Details["field"].(string); ok {
		name = held
	}
	if held, ok := kerr.Details["reason"].(string); ok {
		reason = held
	}
	return name, reason
}

func TestSeedingGivesASiteThePresetsOnce(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	if err := h.service.Seed(t.Context(), h.siteID); err != nil {
		t.Fatalf("Seed again: %v", err)
	}
	listed := h.list(t, h.siteID)
	names := make([]string, 0, len(listed.Types))
	for _, kind := range listed.Types {
		names = append(names, kind.Name)
	}
	if !slices.Equal(names, []string{"Landing", "Category", "Guide", "Info", "Product", "Store category", "Catalog", "Technical"}) {
		t.Fatalf("types = %v, want the client's eight presets once and in order", names)
	}
	hub := h.typeOf(t, h.siteID, pagetype.Category)
	if body := scopeOf(hub, field.Body); !body.Enabled || !body.Stored || !body.Links.Links || body.Links.Rules.MaxLinks != 20 {
		t.Errorf("the category's body = %+v, want it on with the hub's rules", body)
	}
	if hub.Role != "hub" || h.typeOf(t, h.siteID, pagetype.Guide).Role != "entity" || h.typeOf(t, h.siteID, pagetype.StoreCategory).Role != "archive" {
		t.Error("the types do not show their roles")
	}
	info := h.typeOf(t, h.siteID, pagetype.Info)
	if body, title := scopeOf(info, field.Body), scopeOf(info, field.SEOTitle); body.Enabled || body.Stored || !title.Enabled || !title.Stored {
		t.Errorf("an info page's body = %+v and SEO title = %+v, want the body off and the SEO meta on", body, title)
	}
	if excerpt := scopeOf(hub, field.Excerpt); excerpt.Enabled || excerpt.Stored || excerpt.Field != string(field.Excerpt) {
		t.Errorf("the hub's excerpt = %+v, want it listed and off", excerpt)
	}
	if canonical := scopeOf(hub, field.SEOCanonical); canonical.Field == "" || canonical.Enabled || canonical.AIWrites {
		t.Errorf("the canonical = %+v, want it listed for a person to switch on, off and never AI-written", canonical)
	}
	if !hub.Writable || h.typeOf(t, h.siteID, pagetype.StoreCategory).Writable {
		t.Error("the roles' writability is not shown")
	}
	denied := 0
	for _, f := range listed.Fields {
		if wantAI := f.Key == string(field.Excerpt) || f.Key == string(field.SEOTitle); wantAI && !f.AIWritable {
			t.Errorf("%s reads as no model's to write", f.Key)
		}
		if f.Key == string(field.SEOCanonical) && f.AIWritable {
			t.Error("the canonical reads as AI-writable, want it a person's switch")
		}
		if f.Denied {
			denied++
			if f.Scopable {
				t.Errorf("the denied field %s is scopable", f.Key)
			}
		}
	}
	if denied == 0 {
		t.Error("the catalog lists no denied field")
	}
}

func TestEachTypeCountsItsPagesTheUntypedByTheirIdentity(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	pages := sqlite.NewPageRepo(h.store)
	hub := h.typeOf(t, h.siteID, pagetype.Category)
	typed := sqlitetest.Page(t, h.store, h.siteID, "/peptides/")
	if err := pages.SetType(t.Context(), typed.ID, hub.ID, pagemap.TypeByUser, time.Now()); err != nil {
		t.Fatalf("SetType: %v", err)
	}
	sqlitetest.Page(t, h.store, h.siteID, "/guide/")

	listed := h.list(t, h.siteID)
	counts := map[string]int{}
	for _, kind := range listed.Types {
		counts[kind.Preset] = kind.Pages
	}
	if counts["category"] != 1 || counts["guide"] != 1 {
		t.Errorf("pages per type = %v, want the typed category and the untyped page read as a guide", counts)
	}
}

func builtIns(t *testing.T, store *sqlite.Store) map[string]string {
	t.Helper()
	out := map[string]string{}
	for _, name := range []string{"Hub", "Guide", "Product", "Category"} {
		out[name] = sqlitetest.Template(t, store, name).ID
	}
	return out
}

func templateOf(kind pagetypes.Type) string {
	if kind.TemplateID == nil {
		return ""
	}
	return *kind.TemplateID
}

func TestASeededTypeCarriesItsPresetsTemplate(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	shipped := builtIns(t, h.store)
	fresh := sqlitetest.Site(t, h.store, "fresh").ID
	if err := h.service.Seed(t.Context(), fresh); err != nil {
		t.Fatalf("Seed: %v", err)
	}

	want := map[pagetype.Preset]string{
		pagetype.Landing: shipped["Hub"], pagetype.Category: shipped["Hub"], pagetype.Guide: shipped["Guide"],
		pagetype.Info: shipped["Guide"], pagetype.Product: shipped["Product"],
		pagetype.StoreCategory: "", pagetype.Catalog: "", pagetype.Technical: "",
	}
	for preset, templateID := range want {
		if got := templateOf(h.typeOf(t, fresh, preset)); got != templateID {
			t.Errorf("the seeded %s type's template = %q, want %q", preset, got, templateID)
		}
	}
	if got := templateOf(h.typeOf(t, h.siteID, pagetype.Category)); got != "" {
		t.Errorf("a site seeded before the built-ins existed got %q", got)
	}

	chosen := sqlitetest.Site(t, h.store, "chosen")
	chosen.Defaults.TemplateID = new(shipped["Category"])
	if err := sqlite.NewSiteRepo(h.store).Update(t.Context(), chosen); err != nil {
		t.Fatalf("give the site a default template: %v", err)
	}
	if err := h.service.Seed(t.Context(), chosen.ID); err != nil {
		t.Fatalf("Seed the site with a default: %v", err)
	}
	if got := templateOf(h.typeOf(t, chosen.ID, pagetype.Category)); got != "" {
		t.Errorf("a site whose default is chosen seeded its Category type with %q, which would win over that default", got)
	}
}

func TestATypeCreatedFromAPresetInheritsItsTemplate(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	shipped := builtIns(t, h.store)

	cases := []struct {
		name, preset string
		requested    *string
		want         string
	}{
		{name: "Pillar", preset: "hub", want: shipped["Hub"]},
		{name: "Article", preset: "entity", want: shipped["Guide"]},
		{name: "Item", preset: "product", want: shipped["Product"]},
		{name: "Archive", preset: "store_category", want: ""},
		{name: "Top", preset: "landing", want: shipped["Hub"]},
		{name: "Policies", preset: "info", want: shipped["Guide"]},
		{name: "Plain", preset: "hub", requested: new(""), want: ""},
		{name: "Chosen", preset: "hub", requested: new(shipped["Category"]), want: shipped["Category"]},
	}
	for _, tc := range cases {
		created, err := h.service.Create(t.Context(), pagetypes.CreateRequest{
			SiteID: h.siteID, Name: tc.name, Preset: tc.preset, TemplateID: tc.requested,
		})
		if err != nil {
			t.Fatalf("Create %s: %v", tc.name, err)
		}
		if got := templateOf(created); got != tc.want {
			t.Errorf("%s (%s) template = %q, want %q", tc.name, tc.preset, got, tc.want)
		}
	}
}

func TestACanonicalScopeIsNeverAIWritten(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	hub := h.typeOf(t, h.siteID, pagetype.Category)

	_, err := h.service.SetScope(t.Context(), pagetypes.SetScopeRequest{
		PageTypeID: hub.ID, Field: string(field.SEOCanonical), Enabled: true, AIWrites: true, Links: field.NoLinks(),
	})
	if name, reason := reasonOf(t, err); name != "aiWrites" || reason != "field_not_ai_written" {
		t.Fatalf("an AI-written canonical = %v (%s, %s), want it refused", err, name, reason)
	}
	set, err := h.service.SetScope(t.Context(), pagetypes.SetScopeRequest{
		PageTypeID: hub.ID, Field: string(field.SEOCanonical), Enabled: true, Links: field.NoLinks(),
	})
	if err != nil || !set.Enabled || set.AIWrites {
		t.Fatalf("a person's canonical scope = %+v, %v; want it on and set from the permalink, not by the AI", set, err)
	}
}

func TestATypeIsCreatedFromAPresetAndRenamed(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	created, err := h.service.Create(t.Context(), pagetypes.CreateRequest{SiteID: h.siteID, Name: " Pillar ", Preset: "hub"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.Name != "Pillar" || created.Preset != "hub" || !scopeOf(created, field.Body).Enabled || !scopeOf(created, field.Body).Stored {
		t.Fatalf("created = %+v, want Pillar with the hub's scopes", created)
	}
	if _, err = h.service.Create(t.Context(), pagetypes.CreateRequest{SiteID: h.siteID, Name: "pillar", Preset: "entity"}); !errors.IsCode(err, errors.Conflict) {
		t.Errorf("a second Pillar = %q, want CONFLICT", errors.CodeOf(err))
	}
	if _, err = h.service.Create(t.Context(), pagetypes.CreateRequest{SiteID: h.siteID, Name: "Cat", Preset: "pillar"}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("an unknown preset = %q, want INVALID", errors.CodeOf(err))
	}

	hub := h.typeNamed(t, h.siteID, "Landing")
	name := "Top level"
	renamed, err := h.service.Save(t.Context(), pagetypes.SaveRequest{ID: hub.ID, Name: &name})
	if err != nil || renamed.Name != "Top level" || renamed.Preset != "landing" {
		t.Fatalf("Save = %+v %v, want the landing type renamed", renamed, err)
	}
	foreign := sqlitetest.Template(t, h.store, "Global writer").ID
	if renamed, err = h.service.Save(t.Context(), pagetypes.SaveRequest{ID: hub.ID, TemplateID: &foreign}); err != nil || renamed.TemplateID == nil {
		t.Fatalf("a global template = %+v %v", renamed, err)
	}
	none := ""
	if renamed, err = h.service.Save(t.Context(), pagetypes.SaveRequest{ID: hub.ID, TemplateID: &none}); err != nil || renamed.TemplateID != nil {
		t.Fatalf("clearing the template = %+v %v", renamed, err)
	}
	missing := "no-such-template"
	if _, err = h.service.Save(t.Context(), pagetypes.SaveRequest{ID: hub.ID, TemplateID: &missing}); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("an unknown template = %q, want NOT_FOUND", errors.CodeOf(err))
	}
	if len(h.recorder.Events()) == 0 {
		t.Error("no event told the window")
	}
}

func TestDeletingATypeReassignsItsPages(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	pillar, err := h.service.Create(t.Context(), pagetypes.CreateRequest{SiteID: h.siteID, Name: "Pillar", Preset: "hub"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	pages := sqlite.NewPageRepo(h.store)
	page := sqlitetest.Page(t, h.store, h.siteID, "/peptides/")
	if err = pages.SetType(t.Context(), page.ID, pillar.ID, pagemap.TypeByAI, time.Now()); err != nil {
		t.Fatalf("SetType: %v", err)
	}
	hub := h.typeNamed(t, h.siteID, "Category")
	otherHub := h.typeOf(t, h.otherID, pagetype.Category)

	for name, req := range map[string]pagetypes.DeleteRequest{
		"no type to take the pages": {ID: pillar.ID},
		"the type itself":           {ID: pillar.ID, ReassignTo: pillar.ID},
		"another site's type":       {ID: pillar.ID, ReassignTo: otherHub.ID},
	} {
		if _, err = h.service.Delete(t.Context(), req); err == nil {
			t.Errorf("%s: Delete passed", name)
		}
	}

	deleted, err := h.service.Delete(t.Context(), pagetypes.DeleteRequest{ID: pillar.ID, ReassignTo: hub.ID})
	if err != nil || deleted.Moved != 1 {
		t.Fatalf("Delete = %+v %v, want one page moved", deleted, err)
	}
	moved, err := pages.Get(t.Context(), page.ID)
	if err != nil || moved.PageTypeID == nil || *moved.PageTypeID != hub.ID || moved.TypeSource != pagemap.TypeSourceManual || moved.TypeReason != pagemap.TypeByUser {
		t.Fatalf("the page after the delete = %+v %v, want the hub set by a person", moved, err)
	}

	landing := h.typeOf(t, h.siteID, pagetype.Landing)
	if _, err = h.service.Delete(t.Context(), pagetypes.DeleteRequest{ID: landing.ID, ReassignTo: hub.ID}); err != nil {
		t.Fatalf("deleting Landing while Category holds the hub role: %v", err)
	}
	_, err = h.service.Delete(t.Context(), pagetypes.DeleteRequest{ID: hub.ID, ReassignTo: h.typeOf(t, h.siteID, pagetype.Guide).ID})
	if name, reason := reasonOf(t, err); !errors.IsCode(err, errors.Invalid) || name != "id" || reason != "last_of_role" {
		t.Errorf("deleting the last hub-like type = %v (%s %s), want INVALID last_of_role", err, name, reason)
	}
}

func TestAScopeIsRefusedWhenTheDenyListOrTheRoleForbidsIt(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	hub := h.typeOf(t, h.siteID, pagetype.Category)
	links := pagetype.DefaultScope(pagetype.Category, field.Body, hub.ID).Links
	links.Rules.MaxLinks = 8
	set, err := h.service.SetScope(t.Context(), pagetypes.SetScopeRequest{
		PageTypeID: hub.ID, Field: "body", Enabled: true, AIWrites: true, Instruction: "Lead with the products the hub lists.",
		MinLength: 300, MaxLength: 1200, Links: links,
	})
	if err != nil || set.Instruction != "Lead with the products the hub lists." || set.Links.Rules.MaxLinks != 8 || !set.Stored {
		t.Fatalf("SetScope = %+v %v", set, err)
	}

	_, err = h.service.SetScope(t.Context(), pagetypes.SetScopeRequest{PageTypeID: hub.ID, Field: "sku", Enabled: true, Links: field.NoLinks()})
	if name, reason := reasonOf(t, err); name != "field" || reason != field.ReasonDenied {
		t.Errorf("a scope on the SKU = %v (%s %s), want field_denied", err, name, reason)
	}
	_, err = h.service.SetScope(t.Context(), pagetypes.SetScopeRequest{PageTypeID: hub.ID, Field: "meta:_price", Enabled: true, Label: "Price", Provider: "store", Links: field.NoLinks()})
	if _, reason := reasonOf(t, err); reason != field.ReasonDenied {
		t.Errorf("a scope on a protected meta key = %v, want field_denied", err)
	}
	store := h.typeOf(t, h.siteID, pagetype.StoreCategory)
	_, err = h.service.SetScope(t.Context(), pagetypes.SetScopeRequest{PageTypeID: store.ID, Field: "body", Enabled: true, Links: field.NoLinks()})
	if name, reason := reasonOf(t, err); name != "enabled" || reason != "role_read_only" {
		t.Errorf("enabling a store category's body = %v (%s %s), want role_read_only", err, name, reason)
	}
	if _, err = h.service.SetScope(t.Context(), pagetypes.SetScopeRequest{PageTypeID: store.ID, Field: "body", Instruction: "kept for later", Links: field.NoLinks()}); err != nil {
		t.Errorf("an instruction on a disabled read-only scope: %v", err)
	}
	product := h.typeOf(t, h.siteID, pagetype.Product)
	_, err = h.service.SetScope(t.Context(), pagetypes.SetScopeRequest{PageTypeID: product.ID, Field: "excerpt", Enabled: true, AIWrites: true, Links: field.NoLinks()})
	if name, reason := reasonOf(t, err); name != "field" || reason != "excerpt_is_short_description" {
		t.Errorf("an excerpt on a product = %v (%s %s), want it refused: a product's excerpt is its short description", err, name, reason)
	}

	custom, err := h.service.SetScope(t.Context(), pagetypes.SetScopeRequest{
		PageTypeID: hub.ID, Field: "meta:spec_sheet", Enabled: true, Label: "Spec sheet", Provider: "plugin", Links: field.NoLinks(),
	})
	if err != nil || custom.Label != "Spec sheet" || custom.Provider != "plugin" {
		t.Fatalf("a custom field = %+v %v", custom, err)
	}
	if got := scopeOf(h.typeOf(t, h.siteID, pagetype.Category), field.Instance(field.Meta, "spec_sheet")); !got.Enabled {
		t.Errorf("the custom field is not listed on its type: %+v", got)
	}
}

func TestTheSetResolvesAPagesPolicy(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	hub := h.typeOf(t, h.siteID, pagetype.Category)
	set, err := h.service.Set(t.Context(), h.siteID)
	if err != nil {
		t.Fatalf("Set: %v", err)
	}
	page := pagemap.Page{SiteID: h.siteID, WPType: pagemap.WPPage, PageTypeID: &hub.ID}
	if policy := set.Policy(page); !policy.Links || policy.Rules.MaxLinks != 20 || !policy.Rules.ChildrenSection {
		t.Errorf("a hub's policy = %+v, want the hub's rules", policy)
	}
}
