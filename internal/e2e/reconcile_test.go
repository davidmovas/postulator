//go:build e2e

package e2e_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/llm/fake"
	"github.com/davidmovas/postulator/internal/app"
	"github.com/davidmovas/postulator/internal/application/findings"
	"github.com/davidmovas/postulator/internal/application/imports"
	"github.com/davidmovas/postulator/internal/application/sites"
	"github.com/davidmovas/postulator/internal/domain/finding"
)

var clientWorkbookSheets = []string{"Groups", "Catalog", "Entities", "Variations"}

type reconciled struct {
	core   *app.Core
	siteID string
	path   string
	sheets []imports.SheetMapping
}

func reconcileSite(t *testing.T) (*site, *app.Core, *fake.Scripted, string) {
	t.Helper()

	live := newSite(t)
	requirePlugin(t, live.env, true)
	script := sweepScript()
	core := openCoreWith(t, script)
	owner, err := core.Sites.Create(t.Context(), sites.CreateRequest{
		Name: "Reconciled Shop", BaseURL: live.env.baseURL, Username: live.env.user, Password: live.env.pass, AllowInsecure: true,
	})
	if err != nil {
		t.Fatalf("create the site: %v", err)
	}
	runSync(t, core, owner.Site.ID)
	return live, core, script, owner.Site.ID
}

func (r reconciled) preview(t *testing.T) imports.PreviewReport {
	t.Helper()

	previewed, err := r.core.Imports.Preview(t.Context(), imports.PreviewRequest{SiteID: r.siteID, Path: r.path, Sheets: r.sheets})
	if err != nil {
		t.Fatalf("preview %s: %v", filepath.Base(r.path), err)
	}
	if len(previewed.Report.Errors) != 0 {
		t.Fatalf("the preview reported %+v, want no blocking finding", previewed.Report.Errors)
	}
	return previewed.Report
}

func (r reconciled) apply(t *testing.T) imports.ApplyResponse {
	t.Helper()

	applied, err := r.core.Imports.Apply(t.Context(), imports.ApplyRequest{SiteID: r.siteID, Path: r.path, Sheets: r.sheets})
	if err != nil {
		t.Fatalf("apply %s: %v", filepath.Base(r.path), err)
	}
	if len(applied.Report.Errors) != 0 {
		t.Fatalf("the import reported %+v, want no blocking finding", applied.Report.Errors)
	}
	return applied
}

func (r reconciled) findings(t *testing.T, status finding.Status) []findings.Finding {
	t.Helper()

	out := make([]findings.Finding, 0)
	cursor := ""
	for {
		listed, err := r.core.Findings.List(t.Context(), findings.ListRequest{SiteID: r.siteID, Status: string(status), Cursor: cursor, Limit: 100})
		if err != nil {
			t.Fatalf("list the findings: %v", err)
		}
		out = append(out, listed.Items...)
		if !listed.HasMore {
			return out
		}
		cursor = string(listed.Next)
	}
}

func byCode(held []findings.Finding) map[string]int {
	out := make(map[string]int)
	for i := range held {
		out[held[i].Code]++
	}
	return out
}

func byRule(report imports.PreviewReport) map[string]int {
	out := make(map[string]int)
	for i := range report.Pages {
		switch {
		case report.Pages[i].MatchedBy != "":
			out[report.Pages[i].MatchedBy]++
		case report.Pages[i].Generated:
			out["generated"]++
		default:
			out["planned"]++
		}
	}
	for i := range report.Warnings {
		switch report.Warnings[i].Code {
		case string(imports.CodeRowAmbiguous):
			out["ambiguous"]++
		case string(imports.CodeRowWithoutPage):
			out["without a page"]++
		}
	}
	return out
}

func counted(counts map[string]int) string {
	keys := make([]string, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+" "+strconv.Itoa(counts[key]))
	}
	return strings.Join(parts, ", ")
}

func previewedAt(report imports.PreviewReport, sheet, path string) (imports.PreviewPage, bool) {
	for i := range report.Pages {
		if report.Pages[i].Sheet == sheet && report.Pages[i].Path == path {
			return report.Pages[i], true
		}
	}
	return imports.PreviewPage{}, false
}

func onPage(held []findings.Finding, code, pageID string) (findings.Finding, bool) {
	for i := range held {
		if held[i].Code == code && held[i].PageID == pageID {
			return held[i], true
		}
	}
	return findings.Finding{}, false
}

func subjectText(held findings.Finding, key string) string {
	var subject map[string]any
	if json.Unmarshal(held.Subject, &subject) != nil {
		return ""
	}
	if text, ok := subject[key].(string); ok {
		return text
	}
	return ""
}

func TestTheWorkbookReconcilesWithTheSite(t *testing.T) {
	live, core, script, siteID := reconcileSite(t)
	switch live.env.stand {
	case "replica":
		reconcileTheClientsWorkbook(t, core, script, siteID)
	case "vanilla":
		reconcileNestedURLsBySlug(t, core, siteID)
	default:
		t.Skipf("the %s stand has no workbook to reconcile", live.env.stand)
	}
}

func reconcileTheClientsWorkbook(t *testing.T, core *app.Core, script *fake.Scripted, siteID string) {
	sheets := make([]imports.SheetMapping, 0, len(clientWorkbookSheets))
	for _, sheet := range clientWorkbookSheets {
		sheets = append(sheets, imports.SheetMapping{Sheet: sheet})
	}
	r := reconciled{core: core, siteID: siteID, path: sample(t, "client-sheets.xlsx"), sheets: sheets}

	observed := r.findings(t, "")
	report := r.preview(t)
	rules := byRule(report)
	t.Logf("the client's workbook on the replica, its pages by rule: %s", counted(rules))
	for _, sheet := range clientWorkbookSheets {
		perSheet := imports.PreviewReport{}
		for i := range report.Pages {
			if report.Pages[i].Sheet == sheet {
				perSheet.Pages = append(perSheet.Pages, report.Pages[i])
			}
		}
		for i := range report.Warnings {
			if report.Warnings[i].Sheet == sheet {
				perSheet.Warnings = append(perSheet.Warnings, report.Warnings[i])
			}
		}
		t.Logf("  %s: %s", sheet, counted(byRule(perSheet)))
	}
	if rules["ambiguous"] != 0 {
		t.Errorf("rows by rule = %v, want no row two pages answer", rules)
	}
	for _, want := range []struct{ sheet, path, rule string }{
		{"Groups", "/peptides/", "url"},
		{"Groups", "/bpc-157/", "slug"},
		{"Groups", "/tb-500/", "slug"},
		{"Groups", "/reconstitution/", "title"},
		{"Groups", "/healing-peptides/", "title"},
		{"Entities", "/tb-500/", "entity"},
	} {
		if view, found := previewedAt(report, want.sheet, want.path); !found || view.MatchedBy != want.rule || view.Action != string(imports.ActionUpdate) {
			t.Errorf("the %s sheet shows %s as %+v (%t), want an update matched by %s", want.sheet, want.path, view, found, want.rule)
		}
	}
	if view, found := previewedAt(report, "Groups", "/bpc-157/bpc-157-dosing-guide/"); !found || view.Action != string(imports.ActionCreate) {
		t.Errorf("the dosing guide = %+v (%t), want it planned under BPC-157", view, found)
	}
	entityOnly := 0
	for i := range report.Entities {
		if report.Entities[i].Sheet == "Entities" && (report.Entities[i].Name == "TB-500" || report.Entities[i].Name == "Tissue repair") {
			entityOnly++
		}
	}
	t.Logf("  entity-only rows: %d (TB-500 matched by its entity, Tissue repair an entity alone)", entityOnly)
	for i := range report.Pages {
		if strings.Contains(report.Pages[i].Path, "tissue-repair") {
			t.Errorf("an entity-only row planned %+v", report.Pages[i])
		}
	}
	if stored := r.findings(t, ""); len(stored) != len(observed) {
		t.Errorf("the preview took the findings from %d to %d, want none written", len(observed), len(stored))
	}

	applied := r.apply(t)
	t.Logf("applied: %+v", applied.Counts)
	afterImport := r.findings(t, finding.Open)
	t.Logf("open findings after one import: %s", counted(byCode(afterImport)))

	runSync(t, core, siteID)
	afterSync := r.findings(t, finding.Open)
	t.Logf("open findings after one import and one sync: %s", counted(byCode(afterSync)))
	stored := pagesByPath(t, core.Pages, siteID)
	pathOf := make(map[string]string, len(stored))
	for path := range stored {
		pathOf[stored[path].ID] = path
	}
	for i := range afterSync {
		if afterSync[i].Code == string(finding.ParentDiffers) {
			t.Logf("  parent_differs on %s: %s", pathOf[afterSync[i].PageID], afterSync[i].Subject)
		}
	}

	guide, planned := stored["/bpc-157/bpc-157-dosing-guide/"]
	reconstitution, onSite := stored["/reconstitution/"]
	peptides := stored["/peptides/"]
	if !planned || !onSite {
		t.Fatalf("the model holds the dosing guide %t and Reconstitution %t, want both", planned, onSite)
	}
	rowless, found := onPage(afterSync, string(finding.RowWithoutPage), guide.ID)
	if !found || subjectText(rowless, "plannedPath") != "/bpc-157/bpc-157-dosing-guide/" || !strings.HasPrefix(rowless.SubjectKey, "row:Groups:") {
		t.Errorf("the dosing guide's row_without_page = %+v (%t), want it open under its row's key", rowless, found)
	}
	differs, found := onPage(afterSync, string(finding.ParentDiffers), reconstitution.ID)
	if !found {
		t.Fatalf("Reconstitution carries no open parent_differs, want the sheet's Peptides against the site's Lab Guides: %+v", afterSync)
	}
	if same := byCode(afterSync)[string(finding.ParentDiffers)]; same != byCode(afterImport)[string(finding.ParentDiffers)] {
		t.Errorf("parent_differs went from %d after the import to %d after the sync, want the import and the observation to raise one row", byCode(afterImport)[string(finding.ParentDiffers)], same)
	}

	other := slices.IndexFunc(afterSync, func(f findings.Finding) bool {
		return f.Code == string(finding.RowWithoutPage) && f.PageID != guide.ID
	})
	if other < 0 {
		t.Fatalf("the replica holds no other row waiting for a page: %+v", afterSync)
	}
	waiting := afterSync[other]
	ignored, err := core.Findings.Resolve(t.Context(), findings.ResolveRequest{FindingID: waiting.ID, Option: string(finding.Ignore)})
	if err != nil || ignored.Finding.Status != string(finding.Dismissed) || ignored.Finding.ResolvedBy != finding.ResolvedByPerson {
		t.Errorf("a person ignoring %s = %+v, %v", waiting.SubjectKey, ignored.Finding, err)
	}
	theAgentWorksTheFindings(t, core, script, siteID, afterSync, stored)

	runSync(t, core, siteID)
	moved := pagesByPath(t, core.Pages, siteID)["/reconstitution/"]
	if moved.ParentPageID == nil || *moved.ParentPageID != peptides.ID {
		t.Errorf("Reconstitution's model parent = %v, want Peptides, the page its sheet trail names", moved.ParentPageID)
	}

	again := r.apply(t)
	if again.Counts.PagesCreated != 0 || again.Counts.EntitiesCreated != 0 {
		t.Errorf("the second import = %+v, want nothing created", again.Counts)
	}
	settled := r.findings(t, "")
	for _, id := range []string{waiting.ID, rowless.ID, differs.ID} {
		idx := slices.IndexFunc(settled, func(f findings.Finding) bool { return f.ID == id })
		if idx < 0 {
			t.Errorf("finding %s is gone after a second import, want it kept and settled", id)
			continue
		}
		if settled[idx].Status == string(finding.Open) {
			t.Errorf("finding %s after a second import = %+v, want it to stay settled", id, settled[idx])
		}
	}
	runSync(t, core, siteID)
	final := r.findings(t, finding.Open)
	if slices.ContainsFunc(final, func(f findings.Finding) bool { return f.ID == waiting.ID || f.ID == rowless.ID }) {
		t.Error("a row the person ignored or the agent mapped reopened after a sync, want both kept while their subjects stand")
	}
	if _, planned := pagesByPath(t, core.Pages, siteID)["/bpc-157/bpc-157-dosing-guide/"]; planned {
		t.Error("the second import planned the dosing guide again, want its row found on /bpc-157/ by the agent's mapping")
	}
	t.Logf("open findings after the resolutions, a second import and a sync: %s", counted(byCode(final)))
}

func reconcileNestedURLsBySlug(t *testing.T, core *app.Core, siteID string) {
	path := filepath.Join(t.TempDir(), "nested.csv")
	sheet := "URL,Title,Keywords\n" +
		"/guide/,Guide,coffee guide\n" +
		"/brewing/setup/,Setting up your grinder,grinder setup\n" +
		",Cold brew at home,cold brew (480)\n"
	if err := os.WriteFile(path, []byte(sheet), 0o600); err != nil {
		t.Fatalf("write the sheet: %v", err)
	}
	r := reconciled{core: core, siteID: siteID, path: path}

	report := r.preview(t)
	t.Logf("the nested sheet on vanilla, its pages by rule: %s", counted(byRule(report)))
	for _, want := range []struct{ path, rule string }{{"/guide/", "url"}, {"/guide/setup/", "slug"}} {
		if view, found := previewedAt(report, "", want.path); !found || view.MatchedBy != want.rule {
			t.Errorf("%s = %+v (%t), want it matched by %s", want.path, view, found, want.rule)
		}
	}

	r.apply(t)
	runSync(t, core, siteID)
	open := r.findings(t, finding.Open)
	t.Logf("open findings after one import and one sync: %s", counted(byCode(open)))
	stored := pagesByPath(t, core.Pages, siteID)
	if gap, held := stored["/brewing/"]; held {
		t.Errorf("the row matched at /guide/setup/ planned %+v, want no page at /brewing/", gap)
	}
	planned, found := stored["/cold-brew-at-home/"]
	if !found {
		t.Fatalf("the URL-less row was not planned at the slug of its title")
	}
	if _, raised := onPage(open, string(finding.RowWithoutPage), planned.ID); !raised {
		t.Errorf("the URL-less row's row_without_page is not open: %+v", open)
	}
}
