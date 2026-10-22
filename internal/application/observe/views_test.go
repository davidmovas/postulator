package observe_test

import (
	"slices"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/application/observe"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/render"
	"github.com/davidmovas/postulator/internal/kernel/dto"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func TestThePageModelViewShowsAPageAsTheModel(t *testing.T) {
	t.Parallel()
	s := newScenario(t)
	s.serveTheSite(false)
	s.visitor.pages[baseURL+"/healing-peptides/"] = observe.Visit{Status: 200, Body: "<html><body><main><h1>Elsewhere</h1></main></body></html>"}
	s.run(t, observe.Options{})

	tb500 := s.pageAt(t, "/tb-500/")
	peptides := s.pageAt(t, "/peptides/")
	view, err := s.service.Model(t.Context(), observe.ModelRequest{PageID: tb500.ID})
	if err != nil {
		t.Fatalf("Model: %v", err)
	}
	if view.Path != "/tb-500/" || view.Identity == nil || view.Identity.URL != baseURL+"/tb-500/" || view.Identity.ObservedAt.Std().IsZero() {
		t.Errorf("identity = %+v %+v", view.Path, view.Identity)
	}
	want := observe.HierarchyView{
		ParentID: peptides.ID, ParentPath: "/peptides/", ParentTitle: "Peptides",
		Reason: string(pagemap.ParentJSONLD), Method: string(pagemap.MethodHref),
	}
	if view.Hierarchy != want {
		t.Errorf("hierarchy = %+v, want %+v", view.Hierarchy, want)
	}
	if view.Render == nil || view.Render.Root != string(render.RootFound) || len(view.Render.Trails) == 0 || len(view.Render.Regions) == 0 {
		t.Fatalf("render = %+v", view.Render)
	}
	crumbs := view.Render.Trails[0].Crumbs
	if len(crumbs) != 3 || crumbs[1].TargetID != peptides.ID || crumbs[1].TargetTitle != "Peptides" || !crumbs[2].Self || crumbs[0].TargetID == "" {
		t.Errorf("crumbs = %+v, want Home and Peptides resolved and the page itself last", crumbs)
	}
	if view.BodyKind != "classic" || len(view.Blocks) != 2 || view.Blocks[0].Excerpt != "TB-500 is a synthetic fragment of thymosin beta-4." {
		t.Errorf("blocks = %s %+v", view.BodyKind, view.Blocks)
	}
	contextual := 0
	for _, link := range view.Links {
		if link.Type == string(pagemap.LinkContextual) && link.Source == string(pagemap.SourceRaw) && link.ToPath == "/bpc-157/" && link.Position != nil {
			contextual++
		}
	}
	if contextual != 1 {
		t.Errorf("links = %+v, want the raw contextual link to /bpc-157/ at its block", view.Links)
	}

	bpc := s.pageAt(t, "/bpc-157/")
	hub, err := s.service.Model(t.Context(), observe.ModelRequest{PageID: bpc.ID})
	if err != nil {
		t.Fatalf("Model of bpc-157: %v", err)
	}
	if len(hub.Findings) != 1 || hub.Findings[0].Code != string(finding.ParentUnresolved) || hub.Findings[0].Subject["path"] != "/healing-peptides/" {
		t.Errorf("bpc-157's findings = %+v, want the unresolved crumb with its path", hub.Findings)
	}

	if _, err := s.service.Model(t.Context(), observe.ModelRequest{PageID: " "}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("a model without a page = %v, want INVALID", err)
	}
}

func TestThePageModelNamesTheFieldsARunWrote(t *testing.T) {
	t.Parallel()
	s := newScenario(t)
	s.serveTheSite(false)
	s.run(t, observe.Options{})

	tb500 := s.pageAt(t, "/tb-500/")
	runID, at := "run-written", start.Add(time.Hour)
	s.stores.mu.Lock()
	s.stores.fields[tb500.ID][field.SEOTitle] = field.Value{
		PageID: tb500.ID, Key: field.SEOTitle, Current: "TB-500 guide", CurrentHash: field.Hash("TB-500 guide"),
		Base: "TB-500 guide", BaseHash: field.Hash("TB-500 guide"), WrittenAt: &at, WrittenRunID: &runID,
	}
	s.stores.fields[tb500.ID][field.Excerpt] = field.Value{
		PageID: tb500.ID, Key: field.Excerpt, Current: "The owner's excerpt.", CurrentHash: field.Hash("The owner's excerpt."),
		Base: "Our excerpt.", BaseHash: field.Hash("Our excerpt."), WrittenAt: &at, WrittenRunID: &runID,
	}
	s.stores.mu.Unlock()

	view, err := s.service.Model(t.Context(), observe.ModelRequest{PageID: tb500.ID})
	if err != nil {
		t.Fatalf("Model: %v", err)
	}
	want := []observe.FieldView{
		{Key: string(field.Excerpt), WrittenRunID: runID, WrittenAt: dto.NewTime(at), Ours: false},
		{Key: string(field.SEOTitle), WrittenRunID: runID, WrittenAt: dto.NewTime(at), Ours: true},
	}
	if !slices.Equal(view.Fields, want) {
		t.Errorf("fields = %+v, want the two fields the run wrote, the excerpt changed on the site since", view.Fields)
	}
}

func TestTheSummaryCountsTrailsByMethodAndSource(t *testing.T) {
	t.Parallel()
	s := newScenario(t)
	s.serveTheSite(false)
	s.run(t, observe.Options{})

	summary, err := s.service.SummaryOf(t.Context(), observe.SummaryRequest{SiteID: siteID})
	if err != nil {
		t.Fatalf("SummaryOf: %v", err)
	}
	byMethod := summary.TrailsByMethod
	if byMethod[string(pagemap.MethodName)] != 1 || byMethod[string(pagemap.MethodHref)]+byMethod[string(pagemap.MethodName)] != summary.TrailsResolved ||
		summary.TrailsByReason[string(pagemap.ParentJSONLD)] != summary.TrailsResolved {
		t.Errorf("trails = %d by method %v by reason %v", summary.TrailsResolved, summary.TrailsByMethod, summary.TrailsByReason)
	}
	if summary.LastObservedAt.Std().IsZero() || summary.LastFullWalkAt.Std().IsZero() {
		t.Errorf("walk times = %v %v, want both set after a first walk", summary.LastObservedAt, summary.LastFullWalkAt)
	}
	if _, err := s.service.SummaryOf(t.Context(), observe.SummaryRequest{}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("a summary without a site = %v, want INVALID", err)
	}
}
