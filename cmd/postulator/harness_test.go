//go:build uiharness

package main

import (
	"net"
	"path/filepath"
	"testing"

	"github.com/wailsapp/wails/v3/pkg/application"
	"go.uber.org/zap/zaptest"

	"github.com/davidmovas/postulator/internal/app"
	"github.com/davidmovas/postulator/internal/application/agent"
	"github.com/davidmovas/postulator/internal/application/graph"
	"github.com/davidmovas/postulator/internal/application/pages"
	"github.com/davidmovas/postulator/internal/application/runs"
	"github.com/davidmovas/postulator/internal/application/schedules"
	"github.com/davidmovas/postulator/internal/application/sites"
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
	if again.Seed != nil {
		t.Error("a home that already carries a database must not be seeded again")
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
