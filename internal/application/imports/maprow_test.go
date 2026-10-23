package imports_test

import (
	"slices"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/application/imports"
	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/domain/importmap"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/template"
	kctx "github.com/davidmovas/postulator/internal/kernel/ctx"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

func (h harness) mapRow(t *testing.T, findingID, pageID string) {
	t.Helper()

	got, err := h.service.MapRow(t.Context(), imports.MapRowRequest{FindingID: findingID, PageID: pageID, By: finding.ResolvedByPerson})
	if err != nil {
		t.Fatalf("MapRow: %v", err)
	}
	if got.FindingID != findingID || got.PageID != pageID {
		t.Fatalf("MapRow = %+v, want the finding settled on %s", got, pageID)
	}
}

func (h harness) pageByID(t *testing.T, pageID string) (pagemap.Page, bool) {
	t.Helper()

	held, err := sqlite.NewPageRepo(h.store).Get(t.Context(), pageID)
	if errors.IsCode(err, errors.NotFound) {
		return pagemap.Page{}, false
	}
	if err != nil {
		t.Fatalf("Get the page: %v", err)
	}
	return held, true
}

func (h harness) runOn(t *testing.T, pageID string) {
	t.Helper()

	record := run.Run{
		ID: id.New(), SiteID: h.siteID, Kind: run.KindGenerate, Status: run.StatusPending, Targets: []string{pageID},
		Recipe:     []template.StepSpec{{Name: "generate_body", Enabled: true}},
		TemplateID: id.New(), TemplateVersion: 1, PublishMode: run.PublishDraft, Budget: run.Budget{MaxUSD: 1, MaxTokens: 1000},
		CreatedBy: kctx.ActorUser, DeadlineAt: sqlitetest.Stamp.Add(time.Hour), CreatedAt: sqlitetest.Stamp,
	}
	if err := sqlite.NewRunRepo(h.store).Insert(t.Context(), record); err != nil {
		t.Fatalf("insert the run: %v", err)
	}
	item := run.Item{
		ID: id.New(), RunID: record.ID, SiteID: h.siteID, TargetID: pageID, Status: run.StatusPending, CurrentStep: "generate_body",
		Checkpoint: run.NewCheckpoint(), CreatedAt: sqlitetest.Stamp, UpdatedAt: sqlitetest.Stamp,
	}
	if err := sqlite.NewRunItemRepo(h.store).Insert(t.Context(), item); err != nil {
		t.Fatalf("insert the run item: %v", err)
	}
}

func TestAMappedRowIsFoundByItsMappingOnTheNextImport(t *testing.T) {
	t.Parallel()

	t.Run("an ambiguous row mapped to one of its candidates", func(t *testing.T) {
		t.Parallel()

		h := newHarness(t)
		first := h.sitePage(t, "/a/guide/", "Guide", 31, nil)
		second := h.sitePage(t, "/b/guide/", "Guide", 32, nil)
		mapping := h.mapping(map[string]string{string(importmap.FieldTitle): "title", string(importmap.FieldKeywords): "keywords"})
		sheet := "title,keywords\nGuide,guide\n"
		h.applyCSV(t, sheet, mapping)
		ambiguous := h.one(t, finding.RowAmbiguous)

		h.mapRow(t, ambiguous.ID, second.ID)
		settled := h.one(t, finding.RowAmbiguous)
		switch {
		case settled.Status != finding.Resolved || settled.Resolution != imports.ResolutionMapped || settled.ResolvedBy != finding.ResolvedByPerson:
			t.Fatalf("the finding = %+v, want it resolved as mapped by the person", settled)
		case settled.PageID == nil || *settled.PageID != second.ID:
			t.Fatalf("the finding sits on %v, want the chosen page %s", settled.PageID, second.ID)
		}
		if mapped, _ := h.pageByID(t, second.ID); mapped.RowKey != ambiguous.SubjectKey {
			t.Fatalf("the chosen page's row key = %q, want %q", mapped.RowKey, ambiguous.SubjectKey)
		}

		report := h.preview(t, h.file(t, "again.csv", sheet), mapping)
		if view, found := page(report, "/b/guide/"); !found || view.MatchedBy != string(pagemap.RuleMapped) || view.Action != string(imports.ActionUpdate) {
			t.Fatalf("pages = %+v, want the chosen page updated by the mapping", report.Pages)
		}
		h.applyCSV(t, sheet, mapping)
		if updated, _ := h.pageByID(t, second.ID); updated.Keywords.Main() != "guide" {
			t.Fatalf("the chosen page = %+v, want the row's keyword on it", updated)
		}
		if untouched, _ := h.pageByID(t, first.ID); len(untouched.Keywords) != 0 {
			t.Fatalf("the other candidate = %+v, want it untouched", untouched)
		}
		if again := h.one(t, finding.RowAmbiguous); again.Status != finding.Resolved || again.Resolution != imports.ResolutionMapped {
			t.Fatalf("the finding after the next import = %+v, want it to stay settled", again)
		}
	})

	cases := []struct {
		name  string
		named bool
	}{
		{name: "a row without a page mapped to a page of the site drops its planned page"},
		{name: "a planned page a run item names is kept", named: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := newHarness(t)
			guide := h.sitePage(t, "/guides/bpc/", "BPC-157 explained", 40, nil)
			mapping := trailMapping(h, "Category")
			sheet := "Category,URL,Title\nPeptides,,Dosing guide\n"
			h.applyCSV(t, sheet, mapping)
			planned, found := h.pageAt(t, "/dosing-guide/")
			if !found || !slices.Equal(planned.PlannedTrail, []string{"Peptides"}) {
				t.Fatalf("pages = %+v, want the row planned with its trail", h.pages(t))
			}
			if tc.named {
				h.runOn(t, planned.ID)
			}
			without := h.one(t, finding.RowWithoutPage)

			h.mapRow(t, without.ID, guide.ID)
			mapped, _ := h.pageByID(t, guide.ID)
			switch {
			case mapped.RowKey != without.SubjectKey || !slices.Equal(mapped.PlannedTrail, []string{"Peptides"}):
				t.Fatalf("the chosen page = %+v, want the row key and the planned page's trail", mapped)
			case h.one(t, finding.RowWithoutPage).Status != finding.Resolved:
				t.Fatalf("the finding = %+v, want it settled", h.one(t, finding.RowWithoutPage))
			}
			if _, kept := h.pageByID(t, planned.ID); kept != tc.named {
				t.Fatalf("the planned page kept = %t, want %t", kept, tc.named)
			}
			if tc.named {
				return
			}

			again := h.applyCSV(t, sheet, mapping)
			if again.Counts.PagesCreated != 0 {
				t.Fatalf("the next import = %+v, want nothing planned", again.Counts)
			}
			if view, found := page(again.Report, "/guides/bpc/"); !found || view.MatchedBy != string(pagemap.RuleMapped) {
				t.Fatalf("pages = %+v, want the chosen page matched by the mapping", again.Report.Pages)
			}
			if held := h.one(t, finding.RowWithoutPage); held.Status != finding.Resolved || held.Resolution != imports.ResolutionMapped {
				t.Fatalf("the finding = %+v, want it to stay settled", held)
			}
		})
	}
}

func TestMapRowRefusesWhatItCannotMap(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	guide := h.sitePage(t, "/a/guide/", "Guide", 31, nil)
	h.sitePage(t, "/b/guide/", "Guide", 32, nil)
	h.applyCSV(t, "path,title,h1,post type,keywords\n,Guide,,,guide\n/product/mak-liquid/,,Mak Liquid,product,liquid mak\n",
		h.mapping(map[string]string{
			string(importmap.FieldPath): "path", string(importmap.FieldTitle): "title", string(importmap.FieldH1): "h1",
			string(importmap.FieldWPType): "post type", string(importmap.FieldKeywords): "keywords",
		}))
	ambiguous := h.one(t, finding.RowAmbiguous)
	waiting := h.one(t, finding.RowWithoutPage)
	if string(waiting.Subject) != `{"sheet":"","plannedPath":"/product/mak-liquid/","product":true}` {
		t.Fatalf("the waiting row's subject = %s, want a product row", waiting.Subject)
	}

	repo := sqlite.NewFindingRepo(h.store)
	raise := func(code finding.Code, pageID *string, key string) finding.Finding {
		held, newErr := finding.New(finding.Finding{
			ID: id.New(), SiteID: h.siteID, PageID: pageID, Code: code, SubjectKey: key, CreatedAt: settledAt, UpdatedAt: settledAt,
		})
		if newErr != nil {
			t.Fatalf("New: %v", newErr)
		}
		if newErr = repo.Upsert(t.Context(), held); newErr != nil {
			t.Fatalf("Upsert: %v", newErr)
		}
		return held
	}
	other := raise(finding.ParentDiffers, &guide.ID, finding.SubjectPage(guide.ID))
	settled := raise(finding.RowAmbiguous, nil, pagemap.RowKeyPrefix("Elsewhere")+"settled")
	if err := repo.Settle(t.Context(), settled.ID, finding.Dismissed, "ignored", finding.ResolvedByPerson, settledAt); err != nil {
		t.Fatalf("Settle: %v", err)
	}
	archived, err := pagemap.NewPage(pagemap.Page{
		ID: id.New(), SiteID: h.siteID, Path: "/old/", WPType: pagemap.WPPage, Title: "Old", Status: pagemap.StatusArchived,
		CreatedAt: observedAt, UpdatedAt: observedAt,
	})
	if err != nil {
		t.Fatalf("NewPage: %v", err)
	}
	if err = sqlite.NewPageRepo(h.store).Insert(t.Context(), archived); err != nil {
		t.Fatalf("insert the archived page: %v", err)
	}
	stranger := sqlitetest.Page(t, h.store, sqlitetest.Site(t, h.store, "other").ID, "/elsewhere/")

	cases := []struct {
		name      string
		findingID string
		pageID    string
		by        string
		code      errors.Code
	}{
		{name: "no finding named", pageID: guide.ID, by: finding.ResolvedByPerson, code: errors.Invalid},
		{name: "no page named", findingID: ambiguous.ID, by: finding.ResolvedByPerson, code: errors.Invalid},
		{name: "a hand that neither a person nor the agent is", findingID: ambiguous.ID, pageID: guide.ID, by: finding.ResolvedByImport, code: errors.Invalid},
		{name: "an unknown finding", findingID: id.New(), pageID: guide.ID, by: finding.ResolvedByPerson, code: errors.NotFound},
		{name: "a finding that is no row's", findingID: other.ID, pageID: guide.ID, by: finding.ResolvedByPerson, code: errors.Invalid},
		{name: "a settled row", findingID: settled.ID, pageID: guide.ID, by: finding.ResolvedByAgent, code: errors.Conflict},
		{name: "an unknown page", findingID: ambiguous.ID, pageID: id.New(), by: finding.ResolvedByPerson, code: errors.NotFound},
		{name: "a page of another site", findingID: ambiguous.ID, pageID: stranger.ID, by: finding.ResolvedByPerson, code: errors.Invalid},
		{name: "an archived page", findingID: ambiguous.ID, pageID: archived.ID, by: finding.ResolvedByPerson, code: errors.Invalid},
		{name: "a product row onto a page", findingID: waiting.ID, pageID: guide.ID, by: finding.ResolvedByAgent, code: errors.Invalid},
	}
	for _, tc := range cases {
		_, err := h.service.MapRow(t.Context(), imports.MapRowRequest{FindingID: tc.findingID, PageID: tc.pageID, By: tc.by})
		if !errors.IsCode(err, tc.code) {
			t.Errorf("%s: MapRow = %v, want %s", tc.name, err, tc.code)
		}
	}
	for _, held := range []finding.Finding{ambiguous, waiting} {
		if again, getErr := repo.Get(t.Context(), held.ID); getErr != nil || again.Status != finding.Open {
			t.Fatalf("%s after the refusals = %+v, %v; want it open", held.Code, again, getErr)
		}
	}
	for _, held := range h.pages(t) {
		if held.RowKey != "" {
			t.Fatalf("%s carries the row key %q after the refusals", held.Path, held.RowKey)
		}
	}
}
