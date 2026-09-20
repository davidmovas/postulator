//go:build uiharness

package main

import (
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"go.uber.org/zap/zaptest"

	"github.com/davidmovas/postulator/internal/app"
	"github.com/davidmovas/postulator/internal/application/agent"
	"github.com/davidmovas/postulator/internal/application/graph"
	"github.com/davidmovas/postulator/internal/application/pages"
	"github.com/davidmovas/postulator/internal/application/runs"
	"github.com/davidmovas/postulator/internal/application/schedules"
	"github.com/davidmovas/postulator/internal/application/sites"
	"github.com/davidmovas/postulator/internal/application/sync"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/kernel/dto"
	"github.com/davidmovas/postulator/internal/transport/wails"
)

func freeAddress(t *testing.T) string {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve a port: %v", err)
	}
	address := listener.Addr().String()
	if closeErr := listener.Close(); closeErr != nil {
		t.Fatalf("release the port: %v", closeErr)
	}
	return address
}

func seeded(t *testing.T) *app.Core {
	t.Helper()

	home := t.TempDir()
	t.Setenv(app.HomeVariable, home)
	t.Setenv(siteVariable, freeAddress(t))

	cfg, err := app.DefaultConfig()
	if err != nil {
		t.Fatalf("DefaultConfig: %v", err)
	}

	tuned, err := configure(cfg)
	if err != nil {
		t.Fatalf("configure: %v", err)
	}
	if tuned.Seed == nil {
		t.Fatal("an empty home must be seeded")
	}
	if tuned.Config.Provider == nil || tuned.Config.AgentProvider == nil {
		t.Fatal("the harness must compose over both fakes")
	}
	if tuned.Config.DatabasePath != filepath.Join(home, "postulator.db") {
		t.Fatalf("DatabasePath = %q, want it under %q", tuned.Config.DatabasePath, home)
	}

	core, err := app.Open(t.Context(), tuned.Config, zaptest.NewLogger(t))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := core.Close(); closeErr != nil {
			t.Errorf("Close: %v", closeErr)
		}
	})

	if seedErr := tuned.Seed(t.Context(), core); seedErr != nil {
		t.Fatalf("seed: %v", seedErr)
	}
	return core
}

func TestTheHarnessSeedsASiteWorthLookingAt(t *testing.T) {
	core := seeded(t)

	listed, err := core.Sites.List(t.Context(), sites.ListRequest{ListRequest: dto.ListRequest{Limit: 10}})
	if err != nil {
		t.Fatalf("List sites: %v", err)
	}
	if len(listed.Items) != 1 {
		t.Fatalf("seeded sites = %d, want 1", len(listed.Items))
	}
	siteID := listed.Items[0].ID

	loaded, err := core.Graph.LoadGraph(t.Context(), graph.LoadGraphRequest{SiteID: siteID})
	if err != nil {
		t.Fatalf("LoadGraph: %v", err)
	}
	if len(loaded.Entities) != 41 {
		t.Errorf("entities = %d, want 41", len(loaded.Entities))
	}
	if len(loaded.Edges) != 46 {
		t.Errorf("edges = %d, want 46", len(loaded.Edges))
	}

	proposed := 0
	for _, edge := range loaded.Edges {
		if edge.Status == "proposed" {
			proposed++
		}
	}
	if proposed != 6 {
		t.Errorf("proposed edges = %d, want 6", proposed)
	}
}

func TestTheHarnessSeedsMappedAndUnmappedPages(t *testing.T) {
	core := seeded(t)

	listed, err := core.Sites.List(t.Context(), sites.ListRequest{ListRequest: dto.ListRequest{Limit: 10}})
	if err != nil {
		t.Fatalf("List sites: %v", err)
	}
	siteID := listed.Items[0].ID

	total, mapped := 0, 0
	cursor := ""
	for {
		page, listErr := core.Pages.List(t.Context(), pages.ListRequest{
			SiteID: siteID, ListRequest: dto.ListRequest{Limit: 100, Cursor: cursor},
		})
		if listErr != nil {
			t.Fatalf("List pages: %v", listErr)
		}
		for _, item := range page.Items {
			total++
			if item.EntityID != nil && *item.EntityID != "" {
				mapped++
			}
		}
		if page.Next == "" {
			break
		}
		cursor = string(page.Next)
	}

	if total != 62 {
		t.Errorf("pages = %d, want 62", total)
	}
	if mapped != 44 {
		t.Errorf("mapped pages = %d, want 44", mapped)
	}
}

func TestTheHarnessSeedsRunsInEveryStatusARealPathReaches(t *testing.T) {
	core := seeded(t)

	listed, err := core.Runs.List(t.Context(), runs.ListRequest{ListRequest: dto.ListRequest{Limit: 20}})
	if err != nil {
		t.Fatalf("List runs: %v", err)
	}

	seen := make(map[string]bool, len(listed.Items))
	for _, item := range listed.Items {
		seen[item.Status] = true
	}
	for _, want := range []run.Status{run.StatusCompleted, run.StatusFailed, run.StatusRunning, run.StatusCancelled} {
		if !seen[string(want)] {
			t.Errorf("no run is %q; the seeded runs are %v", want, seen)
		}
	}

	completed := ""
	for _, item := range listed.Items {
		if item.Status == string(run.StatusCompleted) {
			completed = item.ID
		}
	}
	if completed == "" {
		t.Fatal("no completed run to read artifacts from")
	}

	items, err := core.Runs.ListItems(t.Context(), runs.ListItemsRequest{
		RunID: completed, ListRequest: dto.ListRequest{Limit: 20},
	})
	if err != nil {
		t.Fatalf("ListItems: %v", err)
	}
	if len(items.Items) != len(completedRun) {
		t.Fatalf("the completed run carries %d items, want %d", len(items.Items), len(completedRun))
	}

	artifacts, err := core.Runs.ListArtifacts(t.Context(), runs.ListArtifactsRequest{ItemID: items.Items[0].ID})
	if err != nil {
		t.Fatalf("ListArtifacts: %v", err)
	}
	if len(artifacts.Artifacts) == 0 {
		t.Error("the completed item holds no artifacts")
	}
}

func TestTheHarnessSeedsAScheduleAndAConversationWaitingOnApproval(t *testing.T) {
	core := seeded(t)

	planned, err := core.Schedules.List(t.Context(), schedules.ListRequest{ListRequest: dto.ListRequest{Limit: 10}})
	if err != nil {
		t.Fatalf("List schedules: %v", err)
	}
	if len(planned.Items) != 1 {
		t.Fatalf("schedules = %d, want 1", len(planned.Items))
	}

	conversations, err := core.Agent.ListConversations(t.Context(), agent.ListConversationsRequest{
		ListRequest: dto.ListRequest{Limit: 10},
	})
	if err != nil {
		t.Fatalf("ListConversations: %v", err)
	}
	if len(conversations.Items) != 1 {
		t.Fatalf("conversations = %d, want 1", len(conversations.Items))
	}
	if conversations.Items[0].Title == "" {
		t.Error("the seeded conversation has no title")
	}

	messages, err := core.Agent.ListMessages(t.Context(), agent.ListMessagesRequest{
		ConversationID: conversations.Items[0].ID, ListRequest: dto.ListRequest{Limit: 50},
	})
	if err != nil {
		t.Fatalf("ListMessages: %v", err)
	}

	assistants := 0
	for _, message := range messages.Items {
		if message.Role == "assistant" {
			assistants++
		}
	}
	if assistants < 2 {
		t.Errorf("assistant messages = %d, want the two seeded exchanges", assistants)
	}

	pending, err := core.Agent.ListPendingActions(t.Context(), agent.ListPendingActionsRequest{
		Status: "pending", ListRequest: dto.ListRequest{Limit: 10},
	})
	if err != nil {
		t.Fatalf("ListPendingActions: %v", err)
	}
	if len(pending.Items) != 1 {
		t.Fatalf("pending actions = %d, want the one write the agent proposed", len(pending.Items))
	}
	if pending.Items[0].Tool != "pages_update" {
		t.Errorf("the pending action calls %q, want pages_update", pending.Items[0].Tool)
	}
}

func TestASeededHomeIsNotSeededTwice(t *testing.T) {
	core := seeded(t)
	if closeErr := core.Close(); closeErr != nil {
		t.Fatalf("Close: %v", closeErr)
	}

	cfg, err := app.DefaultConfig()
	if err != nil {
		t.Fatalf("DefaultConfig: %v", err)
	}
	again, err := configure(cfg)
	if err != nil {
		t.Fatalf("configure: %v", err)
	}

	reopened, err := app.Open(t.Context(), again.Config, zaptest.NewLogger(t))
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := reopened.Close(); closeErr != nil {
			t.Errorf("Close: %v", closeErr)
		}
	})
	if again.Seed != nil {
		if seedErr := again.Seed(t.Context(), reopened); seedErr != nil {
			t.Fatalf("the restart hook: %v", seedErr)
		}
	}

	listed, err := reopened.Sites.List(t.Context(), sites.ListRequest{ListRequest: dto.ListRequest{Limit: 10}})
	if err != nil {
		t.Fatalf("List sites: %v", err)
	}
	if len(listed.Items) != 1 {
		t.Fatalf("sites after the restart = %d, want the one that was seeded", len(listed.Items))
	}

	loaded, err := reopened.Graph.LoadGraph(t.Context(), graph.LoadGraphRequest{SiteID: listed.Items[0].ID})
	if err != nil {
		t.Fatalf("LoadGraph: %v", err)
	}
	if len(loaded.Entities) != 41 {
		t.Fatalf("entities after the restart = %d, want the forty-one that were seeded", len(loaded.Entities))
	}
}

func TestTheHarnessOpensTheDevtoolsEndpointOnDemand(t *testing.T) {
	cases := []struct {
		name string
		port string
		want []string
	}{
		{name: "no port", port: ""},
		{name: "a port", port: "9222", want: []string{"--remote-debugging-port=9222"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(devtoolsVariable, tc.port)

			got := options(application.Options{Name: "Postulator"}).Windows.AdditionalBrowserArgs
			if len(got) != len(tc.want) {
				t.Fatalf("AdditionalBrowserArgs = %v, want %v", got, tc.want)
			}
			for index, arg := range tc.want {
				if got[index] != arg {
					t.Fatalf("AdditionalBrowserArgs = %v, want %v", got, tc.want)
				}
			}
		})
	}
}

func TestTheSeededGraphIsATreeAndNotAFlatList(t *testing.T) {
	core := seeded(t)

	listed, err := core.Sites.List(t.Context(), sites.ListRequest{ListRequest: dto.ListRequest{Limit: 10}})
	if err != nil {
		t.Fatalf("List sites: %v", err)
	}

	loaded, err := core.Graph.LoadGraph(t.Context(), graph.LoadGraphRequest{SiteID: listed.Items[0].ID})
	if err != nil {
		t.Fatalf("LoadGraph: %v", err)
	}

	parents := make(map[string]int, len(loaded.Entities))
	for _, edge := range loaded.Edges {
		if edge.Kind == "parent" {
			parents[edge.FromEntityID]++
		}
	}

	rooted := 0
	for _, entity := range loaded.Entities {
		switch parents[entity.ID] {
		case 0:
			rooted++
		case 1:
		default:
			t.Errorf("%s carries %d parents, want one", entity.Name, parents[entity.ID])
		}
	}
	if rooted != 6 {
		t.Errorf("entities without a parent = %d, want the six hubs", rooted)
	}
}

func TestLockingTheSeededHarnessLeavesTheProcessStanding(t *testing.T) {
	core := seeded(t)

	logger, err := app.Logger()
	if err != nil {
		t.Fatalf("Logger: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := logger.Close(); closeErr != nil {
			t.Errorf("close the logger: %v", closeErr)
		}
	})
	services := core.Services(logger)

	if err = core.SetMasterPassword(t.Context(), "", "correct horse battery"); err != nil {
		t.Fatalf("SetMasterPassword: %v", err)
	}
	if err = core.Lock(); err != nil {
		t.Fatalf("Lock: %v", err)
	}
	if !core.Locked() {
		t.Fatal("the core did not lock")
	}

	settings, ok := services[len(services)-1].Instance().(*wails.SettingsService)
	if !ok {
		t.Fatalf("the last service is %T, want the settings service", services[len(services)-1].Instance())
	}

	state, err := settings.LockState(t.Context(), wails.LockStateRequest{})
	if err != nil {
		t.Fatalf("LockState: %v", err)
	}
	if !state.Locked || !state.Protected {
		t.Fatalf("LockState = %+v, want a locked and protected core", state)
	}

	if _, err = settings.Unlock(t.Context(), wails.UnlockRequest{Password: "correct horse battery"}); err != nil {
		t.Fatalf("Unlock: %v", err)
	}
	if core.Locked() {
		t.Fatal("the core did not come back up")
	}
}

func TestAPendingConfirmationSurvivesARestart(t *testing.T) {
	core := seeded(t)

	before, err := core.Agent.ListPendingActions(t.Context(), agent.ListPendingActionsRequest{
		Status: "pending", ListRequest: dto.ListRequest{Limit: 10},
	})
	if err != nil {
		t.Fatalf("ListPendingActions: %v", err)
	}
	if len(before.Items) != 1 {
		t.Fatalf("pending actions before the restart = %d, want 1", len(before.Items))
	}
	if closeErr := core.Close(); closeErr != nil {
		t.Fatalf("Close: %v", closeErr)
	}

	cfg, err := app.DefaultConfig()
	if err != nil {
		t.Fatalf("DefaultConfig: %v", err)
	}
	again, err := configure(cfg)
	if err != nil {
		t.Fatalf("configure: %v", err)
	}

	reopened, err := app.Open(t.Context(), again.Config, zaptest.NewLogger(t))
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := reopened.Close(); closeErr != nil {
			t.Errorf("Close: %v", closeErr)
		}
	})

	after, err := reopened.Agent.ListPendingActions(t.Context(), agent.ListPendingActionsRequest{
		Status: "pending", ListRequest: dto.ListRequest{Limit: 10},
	})
	if err != nil {
		t.Fatalf("ListPendingActions: %v", err)
	}
	if len(after.Items) != 1 {
		t.Fatalf("pending actions after the restart = %d, want the one that was waiting", len(after.Items))
	}
	if after.Items[0].ID != before.Items[0].ID {
		t.Fatalf("the pending action is %s, want %s", after.Items[0].ID, before.Items[0].ID)
	}
}

func TestTheSeededConfirmationNamesARealPage(t *testing.T) {
	core := seeded(t)

	pending, err := core.Agent.ListPendingActions(t.Context(), agent.ListPendingActionsRequest{
		Status: "pending", ListRequest: dto.ListRequest{Limit: 10},
	})
	if err != nil {
		t.Fatalf("ListPendingActions: %v", err)
	}
	if len(pending.Items) != 1 {
		t.Fatalf("pending actions = %d, want 1", len(pending.Items))
	}

	var args struct {
		ID        string `json:"id"`
		MetaTitle string `json:"metaTitle"`
	}
	if unmarshalErr := json.Unmarshal(pending.Items[0].Args, &args); unmarshalErr != nil {
		t.Fatalf("read the pending arguments %s: %v", pending.Items[0].Args, unmarshalErr)
	}
	if args.ID == "" {
		t.Fatalf("the pending action carries no page id: %s", pending.Items[0].Args)
	}
	if args.MetaTitle == "" {
		t.Fatalf("the pending action carries no new title: %s", pending.Items[0].Args)
	}

	page, err := core.Pages.Get(t.Context(), pages.GetRequest{ID: args.ID})
	if err != nil {
		t.Fatalf("the pending action names a page that does not exist: %v", err)
	}
	if page.Page.Path != "/espresso-machines/under-500/" {
		t.Fatalf("the pending action names %s, want the under-500 page", page.Page.Path)
	}
}

func TestTheCompletedRunSucceedsOnEveryItem(t *testing.T) {
	core := seeded(t)

	listed, err := core.Runs.List(t.Context(), runs.ListRequest{
		Status: string(run.StatusCompleted), ListRequest: dto.ListRequest{Limit: 10},
	})
	if err != nil {
		t.Fatalf("List runs: %v", err)
	}
	if len(listed.Items) != 1 {
		t.Fatalf("completed runs = %d, want 1", len(listed.Items))
	}

	items, err := core.Runs.ListItems(t.Context(), runs.ListItemsRequest{
		RunID: listed.Items[0].ID, ListRequest: dto.ListRequest{Limit: 20},
	})
	if err != nil {
		t.Fatalf("ListItems: %v", err)
	}
	if len(items.Items) != len(completedRun) {
		t.Fatalf("items = %d, want %d", len(items.Items), len(completedRun))
	}
	for _, item := range items.Items {
		if item.Status != string(run.StatusCompleted) {
			t.Errorf("the item for %s is %q: %s", item.TargetID, item.Status, item.Error)
		}
	}
}

func TestTheSeededGraphCarriesScores(t *testing.T) {
	core := seeded(t)

	listed, err := core.Sites.List(t.Context(), sites.ListRequest{ListRequest: dto.ListRequest{Limit: 10}})
	if err != nil {
		t.Fatalf("List sites: %v", err)
	}

	loaded, err := core.Graph.LoadGraph(t.Context(), graph.LoadGraphRequest{SiteID: listed.Items[0].ID})
	if err != nil {
		t.Fatalf("LoadGraph: %v", err)
	}

	scored := 0
	for _, entity := range loaded.Entities {
		if entity.Score > 0 {
			scored++
		}
	}
	if scored == 0 {
		t.Fatal("no entity carries a score; the seed never recomputed them")
	}
}

func TestTheSeededConversationIsNamedLikeAConversation(t *testing.T) {
	core := seeded(t)

	listed, err := core.Agent.ListConversations(t.Context(), agent.ListConversationsRequest{
		ListRequest: dto.ListRequest{Limit: 10},
	})
	if err != nil {
		t.Fatalf("ListConversations: %v", err)
	}
	if len(listed.Items) != 1 {
		t.Fatalf("conversations = %d, want 1", len(listed.Items))
	}

	title := listed.Items[0].Title
	if len(title) < 12 {
		t.Fatalf("the conversation is called %q, which reads like a scripted stub", title)
	}
}

func TestARestartPutsThePublishedPagesBackOnTheFakeSite(t *testing.T) {
	core := seeded(t)

	listed, err := core.Sites.List(t.Context(), sites.ListRequest{ListRequest: dto.ListRequest{Limit: 10}})
	if err != nil {
		t.Fatalf("List sites: %v", err)
	}
	siteID := listed.Items[0].ID

	before, err := core.Pages.List(t.Context(), pages.ListRequest{
		SiteID: siteID, ListRequest: dto.ListRequest{Limit: 100},
	})
	if err != nil {
		t.Fatalf("List pages: %v", err)
	}

	withWP := 0
	for _, page := range before.Items {
		if page.WPID != nil {
			withWP++
		}
	}
	if withWP == 0 {
		t.Fatal("the seed published nothing, so a restart has nothing to put back")
	}
	if closeErr := core.Close(); closeErr != nil {
		t.Fatalf("Close: %v", closeErr)
	}

	cfg, err := app.DefaultConfig()
	if err != nil {
		t.Fatalf("DefaultConfig: %v", err)
	}
	again, err := configure(cfg)
	if err != nil {
		t.Fatalf("configure: %v", err)
	}
	if again.Seed == nil {
		t.Fatal("a restart must still repopulate the fake site")
	}

	reopened, err := app.Open(t.Context(), again.Config, zaptest.NewLogger(t))
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := reopened.Close(); closeErr != nil {
			t.Errorf("Close: %v", closeErr)
		}
	})
	if seedErr := again.Seed(t.Context(), reopened); seedErr != nil {
		t.Fatalf("repopulate: %v", seedErr)
	}

	checked, err := reopened.Sync.CheckPlugin(t.Context(), sync.CheckPluginRequest{SiteID: siteID})
	if err != nil {
		t.Fatalf("CheckPlugin: %v", err)
	}
	if !checked.Plugin.Installed {
		t.Fatal("the restored fake site does not answer as a plugin site")
	}

	for _, page := range before.Items {
		if page.WPID == nil {
			continue
		}
		link, linkErr := reopened.Pages.PreviewLink(t.Context(), pages.PreviewLinkRequest{PageID: page.ID})
		if linkErr != nil {
			t.Fatalf("PreviewLink for %s: %v", page.Path, linkErr)
		}
		if link.URL == "" {
			t.Fatalf("PreviewLink for %s answered no address", page.Path)
		}
		if link.Kind == "preview" && !link.ExpiresAt.Std().After(time.Now()) {
			t.Fatalf("the preview link for %s expired at %s, which is already past", page.Path, link.ExpiresAt)
		}
	}
}

func TestTheHarnessCanHideTorBrowser(t *testing.T) {
	cases := []struct {
		name   string
		hide   string
		wanted string
	}{
		{name: "hidden", hide: "1", wanted: ""},
		{name: "not hidden", hide: "", wanted: os.Getenv("USERPROFILE")},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(torHideVariable, tc.hide)

			if got := environment()("USERPROFILE"); got != tc.wanted {
				t.Fatalf("the injected environment answered %q, want %q", got, tc.wanted)
			}
		})
	}
}
