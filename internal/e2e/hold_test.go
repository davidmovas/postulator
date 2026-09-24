//go:build e2e

package e2e_test

import (
	"strings"
	"sync"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/llm/fake"
	"github.com/davidmovas/postulator/internal/application/graph"
	"github.com/davidmovas/postulator/internal/application/imports"
	port "github.com/davidmovas/postulator/internal/application/llm"
	"github.com/davidmovas/postulator/internal/application/pages"
	"github.com/davidmovas/postulator/internal/application/runs"
	"github.com/davidmovas/postulator/internal/application/sites"
	appsync "github.com/davidmovas/postulator/internal/application/sync"
	"github.com/davidmovas/postulator/internal/application/templates"
	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/kernel/dto"
	"github.com/davidmovas/postulator/internal/runtime/steps"
)

const (
	heldParent = "/menu/drinks/"
	heldChild  = "/menu/drinks/cocktails/"
	childTopic = "house cocktails"
)

const writerAttempts = 3

func unfinishedDraftOf(keyword string) string {
	return `{"title":"How we serve ` + keyword + `: Step-by-Step Guide | Shop",` +
		`"h1":"How we serve ` + keyword + `",` +
		`"sections":[{"slot":1,"heading":"Introduction","html":"<p>This guide to ` + keyword + ` stops after its ` +
		`first section, so the brief finds the rest of it missing.</p>"}],` +
		`"summary":"A guide to ` + keyword + ` that stops early."}`
}

type firstDraftsFail struct {
	mu    sync.Mutex
	calls int
}

func (f *firstDraftsFail) reply(port.Request) string {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	if f.calls <= writerAttempts {
		return unfinishedDraftOf("drinks and cocktails")
	}
	return draftOf("drinks and cocktails")
}

func TestAChildWaitsForItsParentAndGoesOnOnceTheParentIsRegenerated(t *testing.T) {
	live := newSite(t)
	requirePlugin(t, live.env, true)
	live.clear(t)
	defer func() {
		if !t.Failed() {
			live.clear(t)
		}
	}()

	menuID := live.publish(t, "Our Menu", "menu", menuBody, 0)
	live.publish(t, "Main Courses", "main-courses", mainsBody, menuID)

	parentBody := &firstDraftsFail{}
	script := append([]fake.Reply{
		{Step: steps.NameGenerateBody, Match: "Path: " + heldChild + "\n", Text: draftOf(childTopic)},
		{Step: steps.NameGenerateMeta, Match: heldChild, Text: metaOf(childTopic)},
		{Step: steps.NameGenerateBody, Match: "Path: " + heldParent + "\n", Make: parentBody.reply},
	}, replies()...)
	core := openCoreWith(t, fake.NewScripted(script...))

	owner, err := core.Sites.Create(t.Context(), sites.CreateRequest{
		Name: "Docker Shop", BaseURL: live.env.baseURL, Username: live.env.user, Password: live.env.pass,
		AllowInsecure: true,
	})
	if err != nil {
		t.Fatalf("create the site: %v", err)
	}
	siteID := owner.Site.ID

	synced, err := core.Sync.SyncSite(t.Context(), appsync.SyncSiteRequest{SiteID: siteID})
	if err != nil {
		t.Fatalf("sync the site: %v", err)
	}
	awaitRun(t, core.Runs, synced.RunID)
	if _, err = core.Imports.Apply(t.Context(), imports.ApplyRequest{
		SiteID: siteID, Path: samplePath(t), Mapping: detected(t, core, siteID),
	}); err != nil {
		t.Fatalf("apply the import: %v", err)
	}

	parent := pagesByPath(t, core.Pages, siteID)[heldParent]
	if parent.ID == "" || parent.WPID != nil || parent.EntityID == nil {
		t.Fatalf("%s is %+v, want a planned page mapped to an entity", heldParent, parent)
	}
	child := plantChild(t, core.Graph, core.Pages, core.Templates, siteID, parent)

	started, err := core.Runs.Start(t.Context(), runs.StartRequest{
		SiteID: siteID, PageIDs: []string{child.ID}, PublishMode: string(run.PublishDraft), Recipe: recipe(),
	})
	if err != nil {
		t.Fatalf("start the run: %v", err)
	}
	if len(started.Added) != 1 || started.Added[0].Path != heldParent || started.Added[0].NeededBy != heldChild {
		t.Fatalf("the run added %+v, want %s pulled in by %s", started.Added, heldParent, heldChild)
	}

	awaitPause(t, core.Runs, started.RunID)
	paused, err := core.Runs.Get(t.Context(), runs.GetRequest{RunID: started.RunID})
	if err != nil {
		t.Fatalf("read the run: %v", err)
	}
	if paused.Run.Status != string(run.StatusPaused) || paused.Run.PauseReason != string(run.PauseAwaitingParent) {
		t.Fatalf("the run is %s/%s, want it waiting on a parent%s", paused.Run.Status, paused.Run.PauseReason,
			itemFailures(t, core.Runs, started.RunID))
	}
	byPath := itemsByTarget(t, core.Runs, started.RunID)
	failed, held := byPath[parent.ID], byPath[child.ID]
	if failed.Status != string(run.StatusFailed) || failed.CurrentStep != steps.NameGenerateBody {
		t.Fatalf("the parent item is %s at %s, want it failed at the writer once its attempts ran out: %s",
			failed.Status, failed.CurrentStep, failed.Error)
	}
	if !strings.Contains(failed.Error, content.ReasonIncompleteAnswer) && !strings.Contains(failed.Error, "left out") {
		t.Fatalf("the parent failed with %q, want the incomplete draft named", failed.Error)
	}
	if held.Status != string(run.StatusPaused) || held.PauseReason != string(run.PauseAwaitingParent) {
		t.Fatalf("the child item is %s/%s, want it waiting for its parent", held.Status, held.PauseReason)
	}
	if held.WaitingFor == nil || held.WaitingFor.Path != heldParent || held.WaitingFor.ItemID != failed.ID ||
		held.WaitingFor.ItemStatus != string(run.StatusFailed) {
		t.Fatalf("the child waits for %+v, want the failed parent item", held.WaitingFor)
	}
	if _, onSite := live.byPath(t, heldChild); onSite {
		t.Fatalf("%s reached the site before its parent", heldChild)
	}

	if _, err = core.Runs.Regenerate(t.Context(), runs.RegenerateRequest{
		RunID: started.RunID, ItemIDs: []string{failed.ID},
	}); err != nil {
		t.Fatalf("regenerate the parent: %v", err)
	}
	awaitRun(t, core.Runs, started.RunID)

	placedParent, ok := live.byPath(t, heldParent)
	if !ok {
		t.Fatalf("the regenerated %s is not on the site", heldParent)
	}
	placedChild, ok := live.byPath(t, heldChild)
	if !ok || placedChild.Parent != placedParent.ID {
		t.Fatalf("%s is %+v, want it under %s (%d) without a human", heldChild, placedChild, heldParent, placedParent.ID)
	}
}

func plantChild(t *testing.T, entities *graph.Service, service *pages.Service, specs *templates.Service,
	siteID string, parent pages.Page) pages.Page {
	t.Helper()

	kin, err := entities.GetEntity(t.Context(), graph.GetEntityRequest{ID: *parent.EntityID})
	if err != nil {
		t.Fatalf("read the parent entity: %v", err)
	}
	created, err := entities.CreateEntity(t.Context(), graph.CreateEntityRequest{
		SiteID: siteID, Name: "House Cocktails", Kind: kin.Entity.Kind, PrimaryKeyword: childTopic,
	})
	if err != nil {
		t.Fatalf("create the child entity: %v", err)
	}
	resolved, err := specs.ResolveForPage(t.Context(), templates.ResolveForPageRequest{PageID: parent.ID})
	if err != nil {
		t.Fatalf("resolve the parent template: %v", err)
	}

	entityID, templateID := created.Entity.ID, resolved.TemplateID
	planted, err := service.Create(t.Context(), pages.CreateRequest{
		SiteID: siteID, Path: heldChild, Title: "House Cocktails", EntityID: &entityID, TemplateID: &templateID,
	})
	if err != nil {
		t.Fatalf("plan %s: %v", heldChild, err)
	}
	if planted.Page.ParentPageID == nil || *planted.Page.ParentPageID != parent.ID {
		t.Fatalf("%s sits under %v, want %s", heldChild, planted.Page.ParentPageID, parent.ID)
	}
	return planted.Page
}

func itemsByTarget(t *testing.T, service *runs.Service, runID string) map[string]runs.Item {
	t.Helper()

	listed, err := service.ListItems(t.Context(), runs.ListItemsRequest{RunID: runID, ListRequest: dto.ListRequest{Limit: 50}})
	if err != nil {
		t.Fatalf("list the run items: %v", err)
	}
	out := make(map[string]runs.Item, len(listed.Items))
	for i := range listed.Items {
		out[listed.Items[i].TargetID] = listed.Items[i]
	}
	return out
}
