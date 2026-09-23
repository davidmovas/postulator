//go:build e2e

package e2e_test

import (
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/app"
	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/application/imports"
	"github.com/davidmovas/postulator/internal/application/pages"
	"github.com/davidmovas/postulator/internal/application/reports"
	"github.com/davidmovas/postulator/internal/application/runs"
	"github.com/davidmovas/postulator/internal/application/sites"
	appsync "github.com/davidmovas/postulator/internal/application/sync"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/dto"
	"github.com/davidmovas/postulator/internal/runtime/steps"
)

const (
	menuBody = `<h1>Our Menu</h1><p>The card runs from appetizers through main courses to desserts, ` +
		`and the bar pours drinks all evening.</p>`
	mainsBody = `<h1>Main Courses</h1><p>The grill sends out steaks every night, the counter plates seafood ` +
		`to order and the kitchen rolls pasta every morning.</p>`
)

func recipe() []template.StepSpec {
	return []template.StepSpec{
		{Name: steps.NameResolveContext, Enabled: true},
		{Name: steps.NameGenerateBody, Enabled: true},
		{Name: steps.NameGenerateMeta, Enabled: true},
		{Name: steps.NameInsertLinks, Enabled: true},
		{Name: steps.NameRepairLinks, Enabled: true},
		{Name: steps.NameValidate, Enabled: true},
		{Name: steps.NameJudge, Enabled: true},
		{Name: steps.NamePublish, Enabled: true},
		{Name: steps.NameRelinkNeighbors, Enabled: true},
		{Name: steps.NameSyncBack, Enabled: true},
		{Name: steps.NameReport, Enabled: true},
	}
}

func TestTheWholeLoopReachesTheDockerSite(t *testing.T) {
	live := newSite(t)
	requirePlugin(t, live.env, true)
	live.clear(t)

	menuID := live.publish(t, "Our Menu", "menu", menuBody, 0)
	live.publish(t, "Main Courses", "main-courses", mainsBody, menuID)

	core := openCore(t)
	owner, err := core.Sites.Create(t.Context(), sites.CreateRequest{
		Name:          "Docker Shop",
		BaseURL:       live.env.baseURL,
		Username:      live.env.user,
		Password:      live.env.pass,
		AllowInsecure: true,
	})
	if err != nil {
		t.Fatalf("create the site: %v", err)
	}
	siteID := owner.Site.ID

	checked, err := core.Sync.CheckPlugin(t.Context(), appsync.CheckPluginRequest{SiteID: siteID})
	if err != nil {
		t.Fatalf("check the plugin: %v", err)
	}
	if !checked.Plugin.Installed {
		t.Fatalf("the companion plugin is not installed: %+v", checked)
	}

	synced, err := core.Sync.SyncSite(t.Context(), appsync.SyncSiteRequest{SiteID: siteID})
	if err != nil {
		t.Fatalf("sync the site: %v", err)
	}
	awaitRun(t, core.Runs, synced.RunID)

	stored := pagesByPath(t, core.Pages, siteID)
	for _, path := range []string{"/menu/", "/menu/main-courses/"} {
		page, ok := stored[path]
		if !ok || page.WPID == nil {
			t.Fatalf("the sync did not record %s: %+v", path, page)
		}
	}

	applied, err := core.Imports.Apply(t.Context(), imports.ApplyRequest{
		SiteID:  siteID,
		Path:    samplePath(t),
		Mapping: detected(t, core, siteID),
	})
	if err != nil {
		t.Fatalf("apply the import: %v", err)
	}
	if len(applied.Report.Errors) != 0 {
		t.Fatalf("the import reported %+v", applied.Report.Errors)
	}
	if applied.Counts.EntitiesCreated != 8 || applied.Counts.EdgesCreated != 7 {
		t.Fatalf("the import wrote %+v, want eight entities and seven edges", applied.Counts)
	}

	stored = pagesByPath(t, core.Pages, siteID)
	targets := make([]string, 0, len(generated))
	for _, planned := range generated {
		page, ok := stored[planned.path]
		if !ok {
			t.Fatalf("the import created no %s", planned.path)
		}
		if page.Status != string(pagemap.StatusPlanned) {
			t.Fatalf("%s is %q, want planned", planned.path, page.Status)
		}
		targets = append(targets, page.ID)
	}

	started, err := core.Runs.Start(t.Context(), runs.StartRequest{
		SiteID:      siteID,
		PageIDs:     targets,
		PublishMode: string(run.PublishDraft),
		Recipe:      recipe(),
	})
	if err != nil {
		t.Fatalf("start the run: %v", err)
	}
	awaitRun(t, core.Runs, started.RunID)

	items, err := core.Runs.ListItems(t.Context(), runs.ListItemsRequest{
		RunID: started.RunID, ListRequest: dto.ListRequest{Limit: 50},
	})
	if err != nil {
		t.Fatalf("list the run items: %v", err)
	}
	if len(items.Items) != len(generated) {
		t.Fatalf("the run carries %d items, want %d", len(items.Items), len(generated))
	}

	for i := range items.Items {
		item := items.Items[i]
		if item.Status != string(run.StatusCompleted) {
			t.Fatalf("the item for %s is %q: %s%s", item.TargetID, item.Status, item.Error,
				artifactDump(t, core.Runs, item.ID, string(run.ArtifactValidationReport)))
		}

		final := finalReport(t, core.Runs, item.ID)
		if final.Validation == nil || final.Validation.Compliance.Score != 1 {
			t.Fatalf("%s reports compliance %+v, want 1", final.Path, final.Validation)
		}
		if final.Publish == nil || final.Publish.Status != "draft" || final.Publish.WPID == 0 {
			t.Fatalf("%s reports %+v, want a draft on the site", final.Path, final.Publish)
		}
	}

	drafts := make(map[string]contentItem, len(generated))
	published := live.content(t)
	for i := range published {
		if published[i].Status == "draft" && strings.HasPrefix(published[i].Path, "/menu/") {
			drafts[published[i].Path] = published[i]
		}
	}
	if len(drafts) != len(generated) {
		t.Fatalf("the site holds %d drafts under /menu/, want %d: %v", len(drafts), len(generated), drafts)
	}

	for _, planned := range generated {
		draft, ok := drafts[planned.path]
		if !ok {
			t.Fatalf("the site holds no draft at %s", planned.path)
		}
		if !linksTo(draft.Links, parentOf(planned.path)) {
			t.Fatalf("the draft %s does not link to %s: %+v", planned.path, parentOf(planned.path), draft.Links)
		}
		if draft.Meta.Title == "" || draft.Meta.Description == "" {
			t.Fatalf("the draft %s carries no seo meta: %+v", planned.path, draft.Meta)
		}
	}

	stored = pagesByPath(t, core.Pages, siteID)
	for _, planned := range generated {
		assertDraftPreviews(t, core, live, stored[planned.path], drafts[planned.path].Title)
	}
	assertPublishedPreviews(t, core, live, stored["/menu/"])

	for path, child := range map[string]string{
		"/menu/":              "/menu/drinks/",
		"/menu/main-courses/": "/menu/main-courses/steaks/",
	} {
		parent, ok := live.byPath(t, path)
		if !ok {
			t.Fatalf("the site lost %s", path)
		}
		if !linksTo(parent.Links, child) {
			t.Fatalf("the relink left %s without a link to %s: %+v", path, child, parent.Links)
		}
	}

	overview, err := core.Reports.SiteOverview(t.Context(), reports.SiteOverviewRequest{SiteID: siteID})
	if err != nil {
		t.Fatalf("site overview: %v", err)
	}
	if overview.Edges.Realized == 0 {
		t.Fatalf("the overview reports %+v, want realized edges", overview.Edges)
	}

	backup := filepath.Join(t.TempDir(), "postulator.pstx")
	written, err := core.ExportBackup(t.Context(), backup, "hunter2")
	if err != nil {
		t.Fatalf("export the backup: %v", err)
	}
	if written == 0 {
		t.Fatal("the backup is empty")
	}
	if err = core.ImportBackup(t.Context(), backup, "hunter2"); err != nil {
		t.Fatalf("read the backup back: %v", err)
	}

	restored := pagesByPath(t, core.Pages, siteID)
	for _, planned := range generated {
		if _, ok := restored[planned.path]; !ok {
			t.Fatalf("the restored store lost %s", planned.path)
		}
	}

	t.Logf("%d drafts under /menu/, %d pages in the store, %d realized edges, a backup of %d bytes",
		len(drafts), len(restored), overview.Edges.Realized, written)
}

func samplePath(t *testing.T) string {
	t.Helper()
	return filepath.Join(repoRoot(t), "examples", "sitemap-import-example.xlsx")
}

func detected(t *testing.T, core *app.Core, siteID string) imports.Mapping {
	t.Helper()

	seen, err := core.Imports.Inspect(t.Context(), imports.InspectRequest{SiteID: siteID, Path: samplePath(t)})
	if err != nil {
		t.Fatalf("inspect the workbook: %v", err)
	}
	for _, field := range []string{"path", "title", "entity", "parent_entity", "anchors", "primary_keyword"} {
		if seen.Detected.Columns[field] == "" {
			t.Fatalf("%s is not detected in %v", field, seen.Headers)
		}
	}
	return seen.Detected
}

func parentOf(path string) string {
	trimmed := strings.TrimSuffix(path, "/")
	cut := strings.LastIndex(trimmed, "/")
	if cut <= 0 {
		return "/"
	}
	return trimmed[:cut] + "/"
}

func linksTo(links []link, path string) bool {
	for _, href := range links {
		if strings.TrimSuffix(href.Href, "/") == strings.TrimSuffix(path, "/") {
			return true
		}
	}
	return false
}

func pagesByPath(t *testing.T, service *pages.Service, siteID string) map[string]pages.Page {
	t.Helper()

	out := make(map[string]pages.Page, 32)
	cursor := ""
	for {
		listed, err := service.List(t.Context(), pages.ListRequest{
			SiteID: siteID, ListRequest: dto.ListRequest{Limit: 100, Cursor: cursor},
		})
		if err != nil {
			t.Fatalf("list the pages: %v", err)
		}
		for i := range listed.Items {
			out[listed.Items[i].Path] = listed.Items[i]
		}
		if listed.Next == "" {
			return out
		}
		cursor = string(listed.Next)
	}
}

func awaitRun(t *testing.T, service *runs.Service, runID string) {
	t.Helper()

	var (
		seq    int64
		reason string
		done   bool
	)
	waitFor(t, "run "+runID+" to finish", func() bool {
		listed, err := service.ListEvents(t.Context(), runs.ListEventsRequest{
			RunID: runID, SinceSeq: seq, Limit: 200,
		})
		if err != nil {
			t.Fatalf("list the run events: %v", err)
		}
		for i := range listed.Events {
			event := listed.Events[i]
			seq = event.Seq
			switch events.Type(event.Type) {
			case events.RunCompleted:
				done = true
			case events.RunFailed, events.RunCancelled, events.RunBudgetExceeded:
				done, reason = true, event.Type+" "+string(event.Payload)
			}
		}
		return done
	})
	if reason != "" {
		t.Fatalf("the run ended with %s%s", reason, itemFailures(t, service, runID))
	}
}

func itemFailures(t *testing.T, service *runs.Service, runID string) string {
	t.Helper()

	listed, err := service.ListItems(t.Context(), runs.ListItemsRequest{
		RunID: runID, ListRequest: dto.ListRequest{Limit: 50},
	})
	if err != nil {
		return "; the items could not be listed: " + err.Error()
	}

	out := strings.Builder{}
	for i := range listed.Items {
		item := listed.Items[i]
		out.WriteString("; " + item.TargetID + " " + item.Status + " at " + item.CurrentStep + ": " + item.Error)
		if stored, artifactErr := service.GetArtifact(t.Context(), runs.GetArtifactRequest{
			ItemID: item.ID, Kind: "link_context",
		}); artifactErr == nil {
			out.WriteString(" context " + stored.Artifact.Content)
		}
	}
	return out.String()
}

func artifactDump(t *testing.T, service *runs.Service, itemID, kind string) string {
	t.Helper()

	stored, err := service.GetArtifact(t.Context(), runs.GetArtifactRequest{ItemID: itemID, Kind: kind})
	if err != nil {
		return " (no " + kind + ": " + err.Error() + ")"
	}
	return " " + kind + " " + stored.Artifact.Content
}

func finalReport(t *testing.T, service *runs.Service, itemID string) steps.FinalReport {
	t.Helper()

	stored, err := service.GetArtifact(t.Context(), runs.GetArtifactRequest{
		ItemID: itemID, Kind: string(run.ArtifactFinalReport),
	})
	if err != nil {
		t.Fatalf("read the final report of %s: %v", itemID, err)
	}

	var report steps.FinalReport
	if err = json.Unmarshal([]byte(stored.Artifact.Content), &report); err != nil {
		t.Fatalf("decode the final report of %s: %v", itemID, err)
	}
	return report
}

func (s *site) clear(t *testing.T) {
	t.Helper()

	listed := s.content(t)
	for i := range listed {
		if listed[i].Type != "page" || !strings.HasPrefix(listed[i].Path, "/menu") {
			continue
		}
		s.call(t, http.MethodDelete, "/wp-json/wp/v2/pages/"+strconv.Itoa(listed[i].ID)+"?force=true",
			nil, http.StatusOK, nil)
	}
}

func (s *site) anonymousGet(t *testing.T, target string) (status int, body string) {
	t.Helper()

	response, err := s.http.Get(target)
	if err != nil {
		t.Fatalf("GET %s: %v", target, err)
	}
	defer response.Body.Close()

	raw, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read GET %s: %v", target, err)
	}
	return response.StatusCode, string(raw)
}

func assertDraftPreviews(t *testing.T, core *app.Core, live *site, page pages.Page, title string) {
	t.Helper()

	link, err := core.Pages.PreviewLink(t.Context(), pages.PreviewLinkRequest{PageID: page.ID})
	if err != nil {
		t.Fatalf("preview %s: %v", page.Path, err)
	}
	if link.Kind != string(pages.PreviewIssued) || link.ExpiresAt.Std().IsZero() {
		t.Fatalf("preview %s = %+v, want an issued link with an expiry", page.Path, link)
	}
	status, body := live.anonymousGet(t, link.URL)
	if status != http.StatusOK || !strings.Contains(body, title) {
		t.Fatalf("the preview of %s answered %d without the draft's title %q", page.Path, status, title)
	}
}

func assertPublishedPreviews(t *testing.T, core *app.Core, live *site, page pages.Page) {
	t.Helper()

	link, err := core.Pages.PreviewLink(t.Context(), pages.PreviewLinkRequest{PageID: page.ID})
	if err != nil {
		t.Fatalf("preview %s: %v", page.Path, err)
	}
	if link.Kind != string(pages.PreviewPublic) || link.URL != live.env.baseURL+page.Path {
		t.Fatalf("preview %s = %+v, want its public address", page.Path, link)
	}
	if status, _ := live.anonymousGet(t, link.URL); status != http.StatusOK {
		t.Fatalf("the public address of %s answered %d", page.Path, status)
	}
}
