package templates_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/application/applicationtest"
	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/application/templates"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/dto"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type harness struct {
	service  *templates.Service
	store    *sqlite.Store
	recorder *applicationtest.Recorder
	clock    *clock.Fake
	siteID   string
}

func newHarness(t *testing.T) harness {
	t.Helper()

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")
	recorder := &applicationtest.Recorder{}
	clk := clock.NewFake(time.Date(2026, time.September, 18, 9, 0, 0, 0, time.UTC))
	return harness{
		service:  templates.New(sqlite.NewTemplateRepo(store), sqlite.NewLinkPolicyRepo(store), sqlite.NewPageRepo(store), sqlite.NewSiteRepo(store), store, recorder, clk),
		store:    store,
		recorder: recorder,
		clock:    clk,
		siteID:   owner.ID,
	}
}

func ptr[T any](v T) *T {
	return &v
}

func (h harness) wantEvents(t *testing.T, count int) {
	t.Helper()
	got := h.recorder.Events()
	if len(got) != count {
		t.Fatalf("published %d events, want %d: %+v", len(got), count, got)
	}
	for i, event := range got {
		if event.Type != events.TemplatesChanged {
			t.Errorf("event[%d] = %s, want templates.changed", i, event.Type)
		}
		if _, ok := event.Payload.(events.TemplatesChangedPayload); !ok {
			t.Errorf("event[%d] carries %T", i, event.Payload)
		}
	}
	h.recorder.Reset()
}

func (h harness) hub(t *testing.T) templates.Template {
	t.Helper()
	seed := template.Seed()[3]
	created, err := h.service.CreateTemplate(t.Context(), templates.CreateTemplateRequest{Name: "Hub", PageKind: seed.PageKind, Spec: seed.Spec})
	if err != nil {
		t.Fatalf("CreateTemplate: %v", err)
	}
	return created.Template
}

func TestEnsureSeededIsIdempotent(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	for range 2 {
		if err := h.service.EnsureSeeded(t.Context()); err != nil {
			t.Fatalf("EnsureSeeded: %v", err)
		}
	}
	seeded, err := h.service.ListTemplates(t.Context(), templates.ListTemplatesRequest{Scope: "global"})
	if err != nil || len(seeded.Items) != 5 {
		t.Fatalf("seeded templates = %d, %v", len(seeded.Items), err)
	}
	for i := range seeded.Items {
		item := &seeded.Items[i]
		if item.ID == "" || item.Version != 1 || item.CreatedAt.String() != "2026-09-18T09:00:00Z" {
			t.Errorf("seed %s = %+v", item.Name, item)
		}
	}
	if len(h.recorder.Events()) != 0 {
		t.Error("seeding at startup must not publish")
	}
}

func TestTemplateLifecycle(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	hub := h.hub(t)
	if hub.Scope != "global" || hub.SiteID != nil || hub.Version != 1 || len(hub.Spec.Sections) == 0 {
		t.Errorf("CreateTemplate = %+v", hub)
	}
	h.wantEvents(t, 1)

	encoded, err := json.Marshal(hub)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	for _, key := range []string{`"pageKind":"hub"`, `"siteId":null`, `"spec":{"sections":[`, `"linkRules":{"upDepth":1`, `"createdAt":"2026-09-18T09:00:00Z"`} {
		if !strings.Contains(string(encoded), key) {
			t.Errorf("view lacks %s: %s", key, encoded)
		}
	}

	h.clock.Advance(time.Minute)
	spec := hub.Spec
	spec.Tone = "warm"
	updated, err := h.service.UpdateTemplate(t.Context(), templates.UpdateTemplateRequest{ID: hub.ID, Name: ptr("Hub Page"), Spec: &spec})
	if err != nil {
		t.Fatalf("UpdateTemplate: %v", err)
	}
	if updated.Template.Name != "Hub Page" || updated.Template.Version != 2 || updated.Template.Spec.Tone != "warm" || updated.Template.UpdatedAt.String() != "2026-09-18T09:01:00Z" {
		t.Errorf("UpdateTemplate = %+v", updated.Template)
	}
	h.wantEvents(t, 1)

	if _, err = h.service.CreateTemplate(t.Context(), templates.CreateTemplateRequest{Name: "hub page", PageKind: "hub", Spec: spec}); !errors.IsCode(err, errors.Conflict) {
		t.Errorf("duplicate name code = %q, want CONFLICT", errors.CodeOf(err))
	}
	broken := spec
	broken.Sections = nil
	if _, err = h.service.CreateTemplate(t.Context(), templates.CreateTemplateRequest{Name: "Broken", PageKind: "hub", Spec: broken}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("invalid spec code = %q, want INVALID", errors.CodeOf(err))
	}
	if _, err = h.service.CreateTemplate(t.Context(), templates.CreateTemplateRequest{Scope: "site", Name: "Local", PageKind: "hub", Spec: spec}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("site scope without a site code = %q, want INVALID", errors.CodeOf(err))
	}
	if _, err = h.service.CreateTemplate(t.Context(), templates.CreateTemplateRequest{Scope: "galaxy", Name: "Local", PageKind: "hub", Spec: spec}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("unknown scope code = %q, want INVALID", errors.CodeOf(err))
	}
	if _, err = h.service.CreateTemplate(t.Context(), templates.CreateTemplateRequest{Scope: "site", SiteID: ptr("missing"), Name: "Local", PageKind: "hub", Spec: spec}); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("unknown site code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
	local, err := h.service.CreateTemplate(t.Context(), templates.CreateTemplateRequest{Scope: "site", SiteID: &h.siteID, Name: "Hub Page", PageKind: "hub", Spec: spec})
	if err != nil || local.Template.SiteID == nil {
		t.Fatalf("site-scoped template = %+v, %v", local.Template, err)
	}
	h.recorder.Reset()

	got, err := h.service.GetTemplate(t.Context(), templates.GetTemplateRequest{ID: hub.ID})
	if err != nil || got.Template.Version != 2 || len(got.Overrides) != 0 {
		t.Errorf("GetTemplate = %+v, %v", got, err)
	}
	if _, err = h.service.GetTemplate(t.Context(), templates.GetTemplateRequest{ID: "missing"}); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("GetTemplate missing code = %q, want NOT_FOUND", errors.CodeOf(err))
	}

	bySite, err := h.service.ListTemplates(t.Context(), templates.ListTemplatesRequest{Scope: "site", SiteID: h.siteID})
	if err != nil || len(bySite.Items) != 1 || bySite.Items[0].ID != local.Template.ID {
		t.Errorf("ListTemplates by site = %+v, %v", bySite, err)
	}
	byKind, err := h.service.ListTemplates(t.Context(), templates.ListTemplatesRequest{ListRequest: dto.ListRequest{Sort: &dto.Sort{Field: "name", Desc: true}}, PageKind: "hub"})
	if err != nil || len(byKind.Items) != 2 {
		t.Errorf("ListTemplates by kind = %+v, %v", byKind, err)
	}
	if _, err = h.service.ListTemplates(t.Context(), templates.ListTemplatesRequest{Scope: "galaxy"}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("bad scope code = %q, want INVALID", errors.CodeOf(err))
	}
	if _, err = h.service.ListTemplates(t.Context(), templates.ListTemplatesRequest{ListRequest: dto.ListRequest{Sort: &dto.Sort{Field: "version"}}}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("bad sort code = %q, want INVALID", errors.CodeOf(err))
	}

	if _, err = h.service.DeleteTemplate(t.Context(), templates.DeleteTemplateRequest{ID: hub.ID}); err != nil {
		t.Fatalf("DeleteTemplate: %v", err)
	}
	h.wantEvents(t, 1)
	if _, err = h.service.DeleteTemplate(t.Context(), templates.DeleteTemplateRequest{ID: hub.ID}); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("DeleteTemplate twice code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
	if _, err = h.service.UpdateTemplate(t.Context(), templates.UpdateTemplateRequest{ID: hub.ID, Name: ptr("x")}); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("UpdateTemplate missing code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
	if _, err = h.service.UpdateTemplate(t.Context(), templates.UpdateTemplateRequest{ID: local.Template.ID, Spec: &broken}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("UpdateTemplate with a broken spec code = %q, want INVALID", errors.CodeOf(err))
	}
}

func TestOverridesAndResolveForPage(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	hub := h.hub(t)
	page := sqlitetest.Page(t, h.store, h.siteID, "/shop/")
	orphan := sqlitetest.Page(t, h.store, h.siteID, "/orphan/")
	h.recorder.Reset()

	if _, err := h.service.ResolveForPage(t.Context(), templates.ResolveForPageRequest{PageID: page.ID}); !errors.IsCode(err, errors.NotFound) {
		t.Fatalf("a page without a template and a site without a default code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
	if _, err := h.service.ResolveForPage(t.Context(), templates.ResolveForPageRequest{PageID: "missing"}); !errors.IsCode(err, errors.NotFound) {
		t.Fatalf("unknown page code = %q, want NOT_FOUND", errors.CodeOf(err))
	}

	owner, err := sqlite.NewSiteRepo(h.store).Get(t.Context(), h.siteID)
	if err != nil {
		t.Fatalf("Get site: %v", err)
	}
	owner.Defaults.TemplateID = &hub.ID
	if err = sqlite.NewSiteRepo(h.store).Update(t.Context(), owner); err != nil {
		t.Fatalf("set the site default: %v", err)
	}
	viaDefault, err := h.service.ResolveForPage(t.Context(), templates.ResolveForPageRequest{PageID: orphan.ID})
	if err != nil || viaDefault.TemplateID != hub.ID || viaDefault.Version != 1 || viaDefault.Spec.Tone != hub.Spec.Tone {
		t.Errorf("ResolveForPage via the site default = %+v, %v", viaDefault, err)
	}

	siteOverride, err := h.service.SetOverride(t.Context(), templates.SetOverrideRequest{TemplateID: hub.ID, Scope: "site", TargetID: h.siteID, Patch: json.RawMessage(`{"tone":"warm","linkRules":{"maxLinks":5}}`)})
	if err != nil {
		t.Fatalf("SetOverride site: %v", err)
	}
	if siteOverride.Override.Scope != "site" || siteOverride.Override.TargetID != h.siteID {
		t.Errorf("site override = %+v", siteOverride.Override)
	}
	h.wantEvents(t, 1)

	page.TemplateID = &hub.ID
	if err = sqlite.NewPageRepo(h.store).Update(t.Context(), page); err != nil {
		t.Fatalf("assign the page template: %v", err)
	}
	if _, err = h.service.SetOverride(t.Context(), templates.SetOverrideRequest{TemplateID: hub.ID, Scope: "page", TargetID: page.ID, Patch: json.RawMessage(`{"length":{"min":100,"max":200},"images":{"inline":3}}`)}); err != nil {
		t.Fatalf("SetOverride page: %v", err)
	}
	h.wantEvents(t, 1)

	resolved, err := h.service.ResolveForPage(t.Context(), templates.ResolveForPageRequest{PageID: page.ID})
	if err != nil {
		t.Fatalf("ResolveForPage: %v", err)
	}
	if resolved.Spec.Tone != "warm" || resolved.Spec.LinkRules.MaxLinks != 5 || resolved.Spec.Length.Min != 100 || resolved.Spec.Images.Inline != 3 || !resolved.Spec.Images.Featured {
		t.Errorf("ResolveForPage = %+v", resolved.Spec)
	}

	replaced, err := h.service.SetOverride(t.Context(), templates.SetOverrideRequest{TemplateID: hub.ID, Scope: "site", TargetID: h.siteID, Patch: json.RawMessage(`{"tone":"cold"}`)})
	if err != nil || replaced.Override.ID != siteOverride.Override.ID {
		t.Errorf("replacing an override keeps its id: %+v, %v", replaced.Override, err)
	}
	h.recorder.Reset()

	if _, err = h.service.SetOverride(t.Context(), templates.SetOverrideRequest{TemplateID: hub.ID, Scope: "site", TargetID: h.siteID, Patch: json.RawMessage(`{"sections":null}`)}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("an override that breaks the spec code = %q, want INVALID", errors.CodeOf(err))
	}
	if _, err = h.service.SetOverride(t.Context(), templates.SetOverrideRequest{TemplateID: hub.ID, Scope: "site", TargetID: h.siteID, Patch: json.RawMessage(`[1]`)}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("a non-object patch code = %q, want INVALID", errors.CodeOf(err))
	}
	if _, err = h.service.SetOverride(t.Context(), templates.SetOverrideRequest{TemplateID: hub.ID, Scope: "page", TargetID: "missing", Patch: json.RawMessage(`{}`)}); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("unknown page target code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
	if _, err = h.service.SetOverride(t.Context(), templates.SetOverrideRequest{TemplateID: hub.ID, Scope: "site", TargetID: "missing", Patch: json.RawMessage(`{}`)}); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("unknown site target code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
	if _, err = h.service.SetOverride(t.Context(), templates.SetOverrideRequest{TemplateID: "missing", Scope: "site", TargetID: h.siteID, Patch: json.RawMessage(`{}`)}); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("unknown template code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
	if _, err = h.service.SetOverride(t.Context(), templates.SetOverrideRequest{TemplateID: hub.ID, Scope: "galaxy", TargetID: h.siteID, Patch: json.RawMessage(`{}`)}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("bad scope code = %q, want INVALID", errors.CodeOf(err))
	}
	if len(h.recorder.Events()) != 0 {
		t.Error("refused overrides must not publish")
	}

	got, err := h.service.GetTemplate(t.Context(), templates.GetTemplateRequest{ID: hub.ID})
	if err != nil || len(got.Overrides) != 2 {
		t.Errorf("GetTemplate overrides = %+v, %v", got.Overrides, err)
	}
	if _, err = h.service.DeleteOverride(t.Context(), templates.DeleteOverrideRequest{ID: siteOverride.Override.ID}); err != nil {
		t.Fatalf("DeleteOverride: %v", err)
	}
	h.wantEvents(t, 1)
	if _, err = h.service.DeleteOverride(t.Context(), templates.DeleteOverrideRequest{ID: siteOverride.Override.ID}); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("DeleteOverride twice code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
	afterDelete, err := h.service.ResolveForPage(t.Context(), templates.ResolveForPageRequest{PageID: page.ID})
	if err != nil || afterDelete.Spec.Tone != hub.Spec.Tone || afterDelete.Spec.Length.Min != 100 {
		t.Errorf("ResolveForPage after deleting the site override = %+v, %v", afterDelete.Spec, err)
	}
}
