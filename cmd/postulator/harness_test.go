//go:build uiharness

package main

import (
	"encoding/json"
	"maps"
	"net"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"go.uber.org/zap/zaptest"

	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
	"github.com/davidmovas/postulator/internal/app"
	"github.com/davidmovas/postulator/internal/application/agent"
	"github.com/davidmovas/postulator/internal/application/findings"
	"github.com/davidmovas/postulator/internal/application/graph"
	"github.com/davidmovas/postulator/internal/application/imports"
	"github.com/davidmovas/postulator/internal/application/models"
	"github.com/davidmovas/postulator/internal/application/observe"
	"github.com/davidmovas/postulator/internal/application/pages"
	"github.com/davidmovas/postulator/internal/application/pagetypes"
	"github.com/davidmovas/postulator/internal/application/reports"
	"github.com/davidmovas/postulator/internal/application/runs"
	"github.com/davidmovas/postulator/internal/application/schedules"
	"github.com/davidmovas/postulator/internal/application/sitemodel"
	"github.com/davidmovas/postulator/internal/application/sites"
	"github.com/davidmovas/postulator/internal/application/sync"
	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/site"
	"github.com/davidmovas/postulator/internal/kernel/dto"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/runtime/steps"
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
	if tuned.Config.Provider == nil {
		t.Fatal("the harness must compose over the scripted provider")
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

	listed, err := core.Sites.List(t.Context(), seededSites)
	if err != nil {
		t.Fatalf("List sites: %v", err)
	}
	if len(listed.Items) != 2 || listed.Items[0].Name != siteName || listed.Items[1].Name != secondSiteName {
		t.Fatalf("seeded sites = %+v, want %s and then %s to switch to", listed.Items, siteName, secondSiteName)
	}
	siteID := listed.Items[0].ID

	loaded, err := core.Graph.LoadGraph(t.Context(), graph.LoadGraphRequest{SiteID: siteID})
	if err != nil {
		t.Fatalf("LoadGraph: %v", err)
	}
	if len(loaded.Entities) != 45 {
		t.Errorf("entities = %d, want 45, the workbook's planned guide included", len(loaded.Entities))
	}
	if len(loaded.Edges) != 49 {
		t.Errorf("edges = %d, want 49, the planned guide's parent included", len(loaded.Edges))
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

	listed, err := core.Sites.List(t.Context(), seededSites)
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

	if total != 68 {
		t.Errorf("pages = %d, want 68, the duplicated review and the workbook's planned guide included", total)
	}
	if mapped != 48 {
		t.Errorf("mapped pages = %d, want 48", mapped)
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
		seen[item.Status+"/"+item.PauseReason] = true
	}
	for _, want := range []run.Status{run.StatusCompleted, run.StatusFailed, run.StatusRunning, run.StatusCancelled} {
		if !seen[string(want)] {
			t.Errorf("no run is %q; the seeded runs are %v", want, seen)
		}
	}
	if !seen[string(run.StatusPaused)+"/"+string(run.PauseAwaitingParent)] {
		t.Errorf("no run waits for a parent that failed; the seeded runs are %v", seen)
	}

	completed := ""
	for _, item := range listed.Items {
		if item.Status == string(run.StatusCompleted) && item.Kind == string(run.KindGenerate) &&
			item.Stats.Items == len(completedRun) {
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

func TestTheSeededConversationShowsEveryToolOutcomeAScreenRenders(t *testing.T) {
	core := seeded(t)

	conversations, err := core.Agent.ListConversations(t.Context(), agent.ListConversationsRequest{
		ListRequest: dto.ListRequest{Limit: 10},
	})
	if err != nil {
		t.Fatalf("ListConversations: %v", err)
	}
	if len(conversations.Items) != 1 {
		t.Fatalf("conversations = %d, want 1", len(conversations.Items))
	}

	messages, err := core.Agent.ListMessages(t.Context(), agent.ListMessagesRequest{
		ConversationID: conversations.Items[0].ID, ListRequest: dto.ListRequest{Limit: 100},
	})
	if err != nil {
		t.Fatalf("ListMessages: %v", err)
	}

	seen := make(map[string]string, 4)
	cut := ""
	for i := range messages.Items {
		message := messages.Items[i]
		if message.Tool == "" {
			continue
		}
		seen[message.ToolStatus] = message.Tool
		if truncated(t, message.Payload) {
			cut = message.Tool
		}
	}

	for _, status := range []string{"ok", "error"} {
		if seen[status] == "" {
			t.Errorf("no tool call in the seeded conversation is %q; the transcript shows %v", status, seen)
		}
	}
	if cut == "" {
		t.Errorf("no tool result in the seeded conversation was cut to fit, so the shortened row "+
			"never renders; the transcript shows %v", seen)
	}
}

func TestTheSeededConversationShowsADeniedToolCall(t *testing.T) {
	t.Skip("a tool call can only be denied when agent.Deps.Allowed names fewer tools than the registry, " +
		"and nothing sets it: internal/app/core.go builds the agent with no Allowed, so Permit always " +
		"passes and the transcript's refused row, with the copy that says nothing was read or written, " +
		"cannot be reached. A tool the registry does not carry is reported as an error, not a refusal. " +
		"Either per-conversation tool permissions get a producer or the refused state goes.")
}

func truncated(t *testing.T, payload json.RawMessage) bool {
	t.Helper()

	if len(payload) == 0 {
		return false
	}
	var held map[string]any
	if err := json.Unmarshal(payload, &held); err != nil {
		return false
	}
	return held[agent.TruncatedKey] == true
}

func TestTheSeededRunsCoverEveryKindAScreenOffers(t *testing.T) {
	core := seeded(t)

	listed, err := core.Runs.List(t.Context(), runs.ListRequest{ListRequest: dto.ListRequest{Limit: 50}})
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	settled := make(map[string]struct{}, len(listed.Items))
	shown := make(map[string][]string, 4)
	for i := range listed.Items {
		held := listed.Items[i]
		settled[held.Kind+":"+held.Status] = struct{}{}
		shown[held.Kind] = append(shown[held.Kind], held.Status)
	}

	for _, kind := range []string{string(run.KindGenerate), string(run.KindRelink), string(run.KindRevert)} {
		if _, ok := settled[kind+":"+string(run.StatusCompleted)]; !ok {
			t.Errorf("the seeded runs hold no completed %s run, only %v; the screens cannot show one",
				kind, shown[kind])
		}
	}
}

func TestTheHarnessSeedsAnEditItsReviewAndTheRevertOfAField(t *testing.T) {
	core := seeded(t)

	listed, err := core.Sites.List(t.Context(), seededSites)
	if err != nil {
		t.Fatalf("List sites: %v", err)
	}
	held := pagesByPath(t, core, listed.Items[0].ID)

	cases := []struct {
		name     string
		kind     run.Kind
		mode     run.PublishMode
		path     string
		status   run.Status
		pause    run.PauseReason
		artifact run.ArtifactKind
		check    func(t *testing.T, content string)
	}{
		{
			name: "a live edit of a page the client wrote", kind: run.KindGenerate, mode: run.PublishLive, path: editedPath,
			status: run.StatusCompleted, artifact: run.ArtifactPublishResult,
			check: func(t *testing.T, content string) {
				var published steps.PublishResult
				decodeArtifact(t, content, &published)
				if !slices.ContainsFunc(published.Fields, func(change field.Change) bool {
					return change.Key == revertedField && change.Moves()
				}) {
					t.Errorf("the publish result's fields are %+v, want the %s it wrote among them", published.Fields, revertedField)
				}
				if !oursBesideTheClients(published.Blocks, editedPhrase) {
					t.Errorf("the publish result's blocks are %+v, want ours added beside the client's kept block", published.Blocks)
				}
			},
		},
		{
			name: "a draft edit waiting for a person", kind: run.KindGenerate, mode: run.PublishDraft, path: reviewedPath,
			status: run.StatusPaused, pause: run.PauseAwaitingReview, artifact: run.ArtifactEditPlan,
			check: func(t *testing.T, content string) {
				var plan steps.EditPlan
				decodeArtifact(t, content, &plan)
				if len(plan.Pages) != 1 || plan.Pages[0].Path != reviewedPath {
					t.Fatalf("the edit plan holds %+v, want the one page %s", plan.Pages, reviewedPath)
				}
				if len(plan.Pages[0].Fields) == 0 || !oursBesideTheClients(plan.Pages[0].Blocks, reviewedPhrase) {
					t.Errorf("the edit plan of %s changes the fields %+v and the blocks %+v, want fields and ours beside the client's",
						reviewedPath, plan.Pages[0].Fields, plan.Pages[0].Blocks)
				}
			},
		},
		{
			name: "the revert of one field of the live edit", kind: run.KindRevert, mode: run.PublishLive, path: editedPath,
			status: run.StatusCompleted, artifact: run.ArtifactRevertResult,
			check: func(t *testing.T, content string) {
				var reverted steps.RevertResult
				decodeArtifact(t, content, &reverted)
				if len(reverted.Fields) != 1 || reverted.Fields[0].Key != revertedField || reverted.Fields[0].Outcome != steps.OutcomeRestored {
					t.Errorf("the revert result names %+v, want the %s alone, put back", reverted.Fields, revertedField)
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			page, found := held[tc.path]
			if !found || page.WPID == nil {
				t.Fatalf("the seed holds no page on the site at %s", tc.path)
			}
			item := seededItem(t, core, tc.kind, tc.mode, page.ID)
			if item.Status != string(tc.status) || item.PauseReason != string(tc.pause) {
				t.Fatalf("the %s %s item for %s is %s %q (%s), want %s %q", tc.mode, tc.kind, tc.path, item.Status,
					item.PauseReason, item.Note, tc.status, tc.pause)
			}
			stored, getErr := core.Runs.GetArtifact(t.Context(), runs.GetArtifactRequest{ItemID: item.ID, Kind: string(tc.artifact)})
			if getErr != nil {
				t.Fatalf("the %s item for %s holds no %s: %v", tc.kind, tc.path, tc.artifact, getErr)
			}
			tc.check(t, stored.Artifact.Content)
		})
	}
}

func seededItem(t *testing.T, core *app.Core, kind run.Kind, mode run.PublishMode, pageID string) runs.Item {
	t.Helper()

	listed, err := core.Runs.List(t.Context(), runs.ListRequest{Kind: string(kind), ListRequest: dto.ListRequest{Limit: 50}})
	if err != nil {
		t.Fatalf("List runs: %v", err)
	}
	found := make([]runs.Item, 0, 1)
	for index := range listed.Items {
		if listed.Items[index].PublishMode != string(mode) {
			continue
		}
		items, itemsErr := core.Runs.ListItems(t.Context(), runs.ListItemsRequest{
			RunID: listed.Items[index].ID, ListRequest: dto.ListRequest{Limit: 20},
		})
		if itemsErr != nil {
			t.Fatalf("ListItems: %v", itemsErr)
		}
		for item := range items.Items {
			if items.Items[item].TargetID == pageID {
				found = append(found, items.Items[item])
			}
		}
	}
	if len(found) != 1 {
		t.Fatalf("the seed holds %d %s %s items for %s, want one", len(found), mode, kind, pageID)
	}
	return found[0]
}

func decodeArtifact(t *testing.T, content string, out any) {
	t.Helper()

	if err := json.Unmarshal([]byte(content), out); err != nil {
		t.Fatalf("decode the artifact %s: %v", content, err)
	}
}

func oursBesideTheClients(changes []block.Change, phrase string) bool {
	return slices.ContainsFunc(changes, func(change block.Change) bool {
		return change.Owner == block.OwnerOurs && change.Change == block.ChangeAdded
	}) && slices.ContainsFunc(changes, func(change block.Change) bool {
		return change.Owner == block.OwnerClient && change.Change == block.ChangeKept && strings.Contains(change.Raw, phrase)
	})
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

	listed, err := reopened.Sites.List(t.Context(), seededSites)
	if err != nil {
		t.Fatalf("List sites: %v", err)
	}
	if len(listed.Items) != 2 || listed.Items[0].Name != siteName {
		t.Fatalf("sites after the restart = %+v, want the two that were seeded, %s first", listed.Items, siteName)
	}

	loaded, err := reopened.Graph.LoadGraph(t.Context(), graph.LoadGraphRequest{SiteID: listed.Items[0].ID})
	if err != nil {
		t.Fatalf("LoadGraph: %v", err)
	}
	if len(loaded.Entities) != 45 {
		t.Fatalf("entities after the restart = %d, want the forty-five that were seeded", len(loaded.Entities))
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

	listed, err := core.Sites.List(t.Context(), seededSites)
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
	if rooted != 7 {
		t.Errorf("entities without a parent = %d, want the seven hubs", rooted)
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

func TestTheHarnessSeedsAProductTheStoreLetsARunEdit(t *testing.T) {
	core := seeded(t)

	listed, err := core.Sites.List(t.Context(), seededSites)
	if err != nil {
		t.Fatalf("List sites: %v", err)
	}
	if listed.Items[0].Commerce != "ready" {
		t.Errorf("the seeded site's store reads %q, want ready", listed.Items[0].Commerce)
	}

	mapped, err := allPages(t.Context(), core, listed.Items[0].ID)
	if err != nil {
		t.Fatalf("list the pages: %v", err)
	}
	productID := ""
	for i := range mapped {
		if mapped[i].Path == productPath && mapped[i].WPType == "product" && mapped[i].EntityID != nil {
			productID = mapped[i].ID
		}
	}
	if productID == "" {
		t.Fatalf("no mapped product at %s", productPath)
	}

	report, err := core.Reports.PageReport(t.Context(), reports.PageReportRequest{PageID: productID})
	if err != nil {
		t.Fatalf("PageReport: %v", err)
	}
	if report.Status != string(run.StatusCompleted) || len(report.Product) == 0 {
		t.Fatalf("the product's last run is %q with outputs %s, want a completed run that wrote them", report.Status, report.Product)
	}
	var outputs struct {
		ShortDescription string `json:"shortDescription"`
	}
	if err = json.Unmarshal(report.Product, &outputs); err != nil || !strings.Contains(outputs.ShortDescription, "rewards a good grinder") {
		t.Errorf("the product's outputs are %s (%v), want the short description generate_fields wrote", report.Product, err)
	}
	var published struct {
		PreviousProduct *struct {
			Added          []string `json:"added"`
			AttributesSent bool     `json:"attributesSent"`
			ShortWritten   bool     `json:"shortWritten"`
		} `json:"previousProduct"`
	}
	if err = json.Unmarshal(report.Publish, &published); err != nil || published.PreviousProduct == nil {
		t.Fatalf("the publish result %s keeps no snapshot of the product (%v)", report.Publish, err)
	}
	if snapshot := published.PreviousProduct; len(snapshot.Added) != 0 || snapshot.AttributesSent || !snapshot.ShortWritten {
		t.Errorf("the run filled %v (attributes sent %t, short description %t), want the short description alone: the attributes are the store's",
			snapshot.Added, snapshot.AttributesSent, snapshot.ShortWritten)
	}
}

func TestTheCompletedRunSucceedsOnEveryItem(t *testing.T) {
	core := seeded(t)

	listed, err := core.Runs.List(t.Context(), runs.ListRequest{
		Status: string(run.StatusCompleted), Kind: string(run.KindGenerate),
		ListRequest: dto.ListRequest{Limit: 10},
	})
	if err != nil {
		t.Fatalf("List runs: %v", err)
	}
	if len(listed.Items) < 1 {
		t.Fatal("the seed left no completed generate run")
	}

	for i := range listed.Items {
		items, itemErr := core.Runs.ListItems(t.Context(), runs.ListItemsRequest{
			RunID: listed.Items[i].ID, ListRequest: dto.ListRequest{Limit: 20},
		})
		if itemErr != nil {
			t.Fatalf("ListItems: %v", itemErr)
		}
		if len(items.Items) == 0 {
			t.Fatalf("the completed run %s carries no items", listed.Items[i].ID)
		}
		for _, item := range items.Items {
			if item.Status != string(run.StatusCompleted) {
				t.Errorf("the item for %s is %q: %s", item.TargetID, item.Status, item.Error)
			}
		}
	}
}

func TestTheSeededGraphCarriesScores(t *testing.T) {
	core := seeded(t)

	listed, err := core.Sites.List(t.Context(), seededSites)
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

	listed, err := core.Sites.List(t.Context(), seededSites)
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

func TestARestorePutsBackTheBodiesTheSiteHeld(t *testing.T) {
	core := seeded(t)

	listed, err := core.Sites.List(t.Context(), seededSites)
	if err != nil {
		t.Fatalf("List sites: %v", err)
	}
	siteID := listed.Items[0].ID

	restored := wptest.New(t)
	if err = placeOnSite(t.Context(), core, restored, siteID); err != nil {
		t.Fatalf("placeOnSite: %v", err)
	}

	held := pagesByPath(t, core, siteID)
	for _, path := range []string{hubPath, editedPath, reviewedPath, completedRun[0]} {
		page := held[path]
		if page.WPID == nil {
			t.Fatalf("the seed holds no page on the site at %s", path)
		}
		model, modelErr := core.Observe.Page(t.Context(), page.ID)
		if modelErr != nil || model.Body == nil {
			t.Fatalf("the model of %s holds no body (%v)", path, modelErr)
		}
		item, found := restored.Lookup(*page.WPID)
		if !found || item.Content != model.Body.Raw {
			t.Errorf("the restore put %q at %s, want the body the site held %q", item.Content, path, model.Body.Raw)
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

func TestTheHarnessRunsBesideTheRealApplication(t *testing.T) {
	cases := []struct {
		name  string
		port  string
		want  string
		other string
	}{
		{name: "a devtools port", port: "9222", want: ProductionInstanceID + ".uiharness.9222"},
		{name: "another worker's port", port: "9444", want: ProductionInstanceID + ".uiharness.9444"},
		{name: "no port", port: "", want: ProductionInstanceID + ".uiharness.default"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(devtoolsVariable, tc.port)

			only := options(application.Options{Name: "Postulator"}).SingleInstance
			if only == nil {
				t.Fatal("the harness build declares no single instance")
			}
			if only.UniqueID != tc.want {
				t.Fatalf("UniqueID = %q, want %q", only.UniqueID, tc.want)
			}
			if only.UniqueID == ProductionInstanceID {
				t.Fatal("the harness took the production id, so it would refuse to start beside the real application")
			}
		})
	}
}

func reopen(t *testing.T) (*app.Core, harness) {
	t.Helper()

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
	return reopened, again
}

var seededSites = sites.ListRequest{ListRequest: dto.ListRequest{Limit: 10, Sort: &dto.Sort{Field: string(site.SortName)}}}

func publishedPage(t *testing.T, core *app.Core) pages.Page {
	t.Helper()

	listed, err := core.Sites.List(t.Context(), seededSites)
	if err != nil {
		t.Fatalf("List sites: %v", err)
	}

	stored, err := core.Pages.List(t.Context(), pages.ListRequest{
		SiteID: listed.Items[0].ID, ListRequest: dto.ListRequest{Limit: 100},
	})
	if err != nil {
		t.Fatalf("List pages: %v", err)
	}
	for index := range stored.Items {
		if stored.Items[index].WPID != nil {
			return stored.Items[index]
		}
	}
	t.Fatal("the seed published nothing")
	return pages.Page{}
}

func TestARestartOfALockedHomeRestoresOnceItIsUnlocked(t *testing.T) {
	core := seeded(t)
	target := publishedPage(t, core)

	if err := core.SetMasterPassword(t.Context(), "", "correct horse battery"); err != nil {
		t.Fatalf("SetMasterPassword: %v", err)
	}
	if closeErr := core.Close(); closeErr != nil {
		t.Fatalf("Close: %v", closeErr)
	}

	reopened, again := reopen(t)
	if !reopened.Locked() {
		t.Fatal("a home with a master password must reopen locked")
	}
	if again.Seed == nil {
		t.Fatal("a restart still needs its hook")
	}
	if seedErr := again.Seed(t.Context(), reopened); seedErr != nil {
		t.Fatalf("the restart hook refused a locked core: %v", seedErr)
	}

	if err := reopened.Unlock(t.Context(), "correct horse battery"); err != nil {
		t.Fatalf("Unlock: %v", err)
	}

	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		link, err := reopened.Pages.PreviewLink(t.Context(), pages.PreviewLinkRequest{PageID: target.ID})
		if err == nil && link.URL != "" {
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatal("the fake site was never restored after the unlock")
}

func TestARestartOfAnOpenHomeRestoresAtOnce(t *testing.T) {
	core := seeded(t)
	target := publishedPage(t, core)

	if closeErr := core.Close(); closeErr != nil {
		t.Fatalf("Close: %v", closeErr)
	}

	reopened, again := reopen(t)
	if reopened.Locked() {
		t.Fatal("a home without a master password must reopen composed")
	}
	if seedErr := again.Seed(t.Context(), reopened); seedErr != nil {
		t.Fatalf("the restart hook: %v", seedErr)
	}

	link, err := reopened.Pages.PreviewLink(t.Context(), pages.PreviewLinkRequest{PageID: target.ID})
	if err != nil {
		t.Fatalf("PreviewLink after the restart: %v", err)
	}
	if link.URL == "" {
		t.Fatal("PreviewLink answered no address")
	}
}

func TestASyncKeepsThePagesTheSiteStillHolds(t *testing.T) {
	core := seeded(t)

	listed, err := core.Sites.List(t.Context(), seededSites)
	if err != nil {
		t.Fatalf("List sites: %v", err)
	}
	siteID := listed.Items[0].ID

	before := pagesByPath(t, core, siteID)
	target, held := before["/accessories/knock-boxes/"]
	if !held {
		t.Fatal("the seed never mapped /accessories/knock-boxes/")
	}
	t.Logf("before the sync: status %q, wpId %v", target.Status, target.WPID)

	stopActiveRuns(t, core)

	started, err := core.Sync.SyncSite(t.Context(), sync.SyncSiteRequest{SiteID: siteID})
	if err != nil {
		t.Fatalf("SyncSite: %v", err)
	}
	if err = waitForRun(t.Context(), core, started.RunID, run.StatusCompleted); err != nil {
		t.Fatalf("the sync run: %v", err)
	}

	after := pagesByPath(t, core, siteID)
	settled, still := after["/accessories/knock-boxes/"]
	if !still {
		for path := range after {
			if _, known := before[path]; !known {
				t.Logf("new path after the sync: %s (wpId %v, status %s)", path, after[path].WPID, after[path].Status)
			}
		}
		for path := range before {
			if _, known := after[path]; !known {
				t.Logf("gone after the sync: %s (wpId %v, status %s)", path, before[path].WPID, before[path].Status)
			}
		}
		t.Fatalf("the sync dropped /accessories/knock-boxes/; the map went from %d to %d pages",
			len(before), len(after))
	}
	t.Logf("after the sync: status %q, wpId %v", settled.Status, settled.WPID)

	if settled.Status == statusArchived {
		t.Errorf("the sync archived /accessories/knock-boxes/ although the fake site still holds it")
	}
	if len(after) != len(before) {
		t.Errorf("the map went from %d to %d pages", len(before), len(after))
	}
}

func TestTheSeededSiteIsObservedIntoTheModel(t *testing.T) {
	core := seeded(t)

	listed, err := core.Sites.List(t.Context(), seededSites)
	if err != nil {
		t.Fatalf("List sites: %v", err)
	}
	siteID := listed.Items[0].ID

	summary, err := core.Observe.SummaryOf(t.Context(), observe.SummaryRequest{SiteID: siteID})
	if err != nil {
		t.Fatalf("SummaryOf: %v", err)
	}
	if summary.Rendered == 0 || summary.TrailsResolved == 0 || summary.LinksByType["breadcrumb"] == 0 || summary.LinksByType["navigation"] == 0 {
		t.Fatalf("summary = %+v, want rendered pages with trails, breadcrumb and navigation links", summary)
	}

	shown := 0
	held := pagesByPath(t, core, siteID)
	for path, page := range held {
		parent := parentOf(path)
		if above, nested := held[parent]; page.WPID == nil || page.Status != "published" || !nested || above.WPID == nil {
			continue
		}
		view, modelErr := core.Observe.Model(t.Context(), observe.ModelRequest{PageID: page.ID})
		if modelErr != nil {
			t.Fatalf("Model of %s: %v", path, modelErr)
		}
		if view.Render == nil || len(view.Render.Trails) == 0 || view.Hierarchy.ParentPath != parent || view.Hierarchy.Reason != "jsonld_breadcrumb" {
			t.Errorf("%s reads render %v hierarchy %+v, want its trail naming %s", path, view.Render != nil, view.Hierarchy, parent)
		}
		shown++
	}
	if shown == 0 {
		t.Fatal("the seed holds no nested published page to read as the model")
	}
}

func TestTheSeededSiteCarriesPageTypes(t *testing.T) {
	core := seeded(t)

	listed, err := core.Sites.List(t.Context(), seededSites)
	if err != nil {
		t.Fatalf("List sites: %v", err)
	}
	siteID := listed.Items[0].ID

	typed, err := core.PageTypes.List(t.Context(), pagetypes.ListRequest{SiteID: siteID})
	if err != nil {
		t.Fatalf("List page types: %v", err)
	}
	byPreset := make(map[pagetype.Preset]pagetypes.Type, len(typed.Types))
	for index := range typed.Types {
		preset := pagetype.Preset(typed.Types[index].Preset)
		if _, twice := byPreset[preset]; twice {
			t.Errorf("the site carries two %s types", preset)
		}
		byPreset[preset] = typed.Types[index]
	}
	for _, preset := range pagetype.Seeded() {
		if kind, held := byPreset[preset]; !held || kind.Name != preset.DefaultName() {
			t.Errorf("the %s type is %+v, want it seeded as %q", preset, kind, preset.DefaultName())
		}
	}

	type typedPage struct {
		path   string
		preset pagetype.Preset
		source pagemap.TypeSource
		reason pagemap.TypeReason
	}
	cases := make([]typedPage, 0, len(seedTypes())+1)
	for _, declared := range seedTypes() {
		cases = append(cases, typedPage{declared.Path, declared.Preset, pagemap.TypeSourceManual, pagemap.TypeByUser})
	}
	cases = append(cases, typedPage{productPath, pagetype.Product, pagemap.TypeSourceAuto, pagemap.TypeByRule})

	held := pagesByPath(t, core, siteID)
	for _, want := range cases {
		page, found := held[want.path]
		if !found {
			t.Errorf("the seed holds no page at %s", want.path)
			continue
		}
		kind, typedAs := byPreset[want.preset], ""
		if page.PageTypeID != nil {
			typedAs = *page.PageTypeID
		}
		if typedAs != kind.ID || page.TypeSource != string(want.source) || page.TypeReason != string(want.reason) {
			t.Errorf("%s is typed %q by %q %q, want %s (%s) by %s %s", want.path, typedAs, page.TypeSource, page.TypeReason,
				kind.Name, kind.ID, want.source, want.reason)
		}
	}
	rules, err := core.PageTypes.ListRules(t.Context(), pagetypes.ListRulesRequest{SiteID: siteID})
	if err != nil {
		t.Fatalf("List rules: %v", err)
	}
	article := slices.IndexFunc(rules.Rules, func(rule pagetypes.Rule) bool { return rule.Name == articleRuleName })
	if grind := held["/grinders/grind-size/"]; article < 0 || grind.PageTypeID == nil || *grind.PageTypeID != byPreset[pagetype.Guide].ID ||
		grind.TypeRuleID != rules.Rules[article].ID {
		t.Errorf("/grinders/grind-size/ is typed %+v by the rule %q, want Guide by the site's own %q", grind.PageTypeID, grind.TypeRuleID, articleRuleName)
	}

	hub := byPreset[pagetype.Category]
	if body, found := scopeOf(hub, field.Body); !found || !body.Stored || !body.Enabled || !body.AIWrites || body.Links != harnessLinks() {
		t.Errorf("the Category type's body scope is %+v, want the harness's link rules with rotating anchors %+v", body, harnessLinks())
	}

	custom, found := scopeOf(hub, field.Instance(field.Meta, subtitleKey))
	if !found || !custom.Stored || !custom.Enabled || custom.AIWrites || custom.Label != subtitleLabel || custom.Provider != string(field.ProviderPlugin) {
		t.Errorf("the Category type's custom scope is %+v, want %s named %q, read through the plugin", custom, subtitleKey, subtitleLabel)
	}
	for path, subtitle := range seedSubtitles() {
		model, modelErr := core.Observe.Page(t.Context(), held[path].ID)
		if modelErr != nil {
			t.Fatalf("Page model of %s: %v", path, modelErr)
		}
		if !slices.ContainsFunc(model.Fields, func(value field.Value) bool {
			return value.Key == field.Instance(field.Meta, subtitleKey) && value.Current == subtitle
		}) {
			t.Errorf("%s holds the fields %+v, want its %s read as %q", path, model.Fields, subtitleKey, subtitle)
		}
	}
}

func TestTheSeededHubShowsTheWriterItsPictureAndTheGraphAProposal(t *testing.T) {
	core := seeded(t)

	listed, err := core.Sites.List(t.Context(), seededSites)
	if err != nil {
		t.Fatalf("List sites: %v", err)
	}
	siteID := listed.Items[0].ID

	logger, err := app.Logger()
	if err != nil {
		t.Fatalf("Logger: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := logger.Close(); closeErr != nil {
			t.Errorf("close the logger: %v", closeErr)
		}
	})
	var (
		pagesService *wails.PagesService
		graphService *wails.GraphService
	)
	for _, service := range core.Services(logger) {
		switch bound := service.Instance().(type) {
		case *wails.PagesService:
			pagesService = bound
		case *wails.GraphService:
			graphService = bound
		}
	}
	if pagesService == nil || graphService == nil {
		t.Fatal("the core binds no pages or graph service")
	}

	held := pagesByPath(t, core, siteID)
	if term, found := held[storeCategoryPath]; !found || term.WPType != "product_cat" || term.WPID == nil || *term.WPID != storeCategoryID {
		t.Fatalf("the store category %s is %+v, want the term the hub twins with", storeCategoryPath, term)
	}
	pack, err := pagesService.WriterPack(t.Context(), sitemodel.PackRequest{PageID: held[hubPath].ID, Field: string(field.Body)})
	if err != nil {
		t.Fatalf("WriterPack of %s: %v", hubPath, err)
	}
	shown := make(map[string]string, len(pack.Sections))
	for index := range pack.Sections {
		section := &pack.Sections[index]
		shown[section.Heading] = strings.Join(append(slices.Clone(section.Lines), section.Items...), "\n")
	}
	for heading, want := range map[string]string{
		"The field's current value (the client's)":            hubBodyPhrase,
		"Its other fields":                                    "- Title: Espresso machines: every type explained",
		"Products of its store category " + storeCategoryName: "- " + productName + " (" + productPath + ")",
	} {
		if !strings.Contains(shown[heading], want) {
			t.Errorf("the writer's pack of %s shows %q under %q, want %q in it; it holds %v", hubPath, shown[heading], heading, want,
				slices.Sorted(maps.Keys(shown)))
		}
	}
	if pack.Tokens == 0 || len(pack.Dropped) != 0 {
		t.Errorf("the pack carries %d tokens and dropped %+v, want a whole pack", pack.Tokens, pack.Dropped)
	}

	proposals, err := graphService.CardProposals(t.Context(), graph.CardProposalsRequest{SiteID: siteID})
	if err != nil {
		t.Fatalf("CardProposals: %v", err)
	}
	if len(proposals.Items) != 1 {
		t.Fatalf("open card proposals = %d, want 1", len(proposals.Items))
	}
	proposal := proposals.Items[0]
	if proposal.Name != productHub || proposal.Current.Definition != "" || proposal.Proposed.Definition != proposedDefinition ||
		len(proposal.Proposed.Facts) != len(proposedFacts()) || proposal.Changed {
		t.Errorf("the proposal is %+v, want the seeded card for %s against its empty card", proposal, productHub)
	}
}

func TestTheHarnessRecordsSpendWorthReading(t *testing.T) {
	core := seeded(t)

	month, err := core.Models.SpendReport(t.Context(), models.SpendReportRequest{Days: 30})
	if err != nil {
		t.Fatalf("SpendReport: %v", err)
	}
	purposes := make(map[string]bool, len(month.Slices))
	for index := range month.Slices {
		purposes[month.Slices[index].Purpose] = true
	}
	for _, want := range []llm.Purpose{
		llm.PurposeRun, llm.PurposeChat, llm.PurposeTitle, llm.PurposeGraph, llm.PurposeAudit, llm.PurposeProbe,
	} {
		if !purposes[string(want)] {
			t.Errorf("the month's spend has no %s slice; it has %v", want, purposes)
		}
	}
	if month.Totals.FlexShare <= 0 || month.Totals.ReasoningShare <= 0 || month.Totals.CachedShare <= 0 {
		t.Errorf("the month's shares are flex %.2f, reasoning %.2f, cached %.2f; want every one above zero",
			month.Totals.FlexShare, month.Totals.ReasoningShare, month.Totals.CachedShare)
	}

	week, err := core.Models.SpendReport(t.Context(), models.SpendReportRequest{Days: 7})
	if err != nil {
		t.Fatalf("SpendReport: %v", err)
	}
	if week.Totals.USD >= month.Totals.USD {
		t.Errorf("the week spent %.4f and the month %.4f; want older writer calls the week leaves out",
			week.Totals.USD, month.Totals.USD)
	}

	calls, err := core.Models.ListCalls(t.Context(), models.ListCallsRequest{ListRequest: dto.ListRequest{Limit: 100}})
	if err != nil {
		t.Fatalf("ListCalls: %v", err)
	}
	if len(calls.Items) <= 25 {
		t.Errorf("recent calls = %d, want more than one page of 25", len(calls.Items))
	}
	failed := make(map[string]bool)
	for index := range calls.Items {
		if calls.Items[index].Status == string(llm.CallError) {
			failed[calls.Items[index].ErrorCode] = true
		}
	}
	for _, want := range []errors.Code{errors.NeedsHuman, errors.RateLimited, errors.Unauthorized} {
		if !failed[string(want)] {
			t.Errorf("no recent call failed with %s; the failures are %v", want, failed)
		}
	}

	listed, err := core.Sites.List(t.Context(), seededSites)
	if err != nil {
		t.Fatalf("List sites: %v", err)
	}
	finished, err := finishedRunItems(t.Context(), core, listed.Items[0].ID)
	if err != nil {
		t.Fatalf("find the finished run: %v", err)
	}
	byStep, err := core.Models.SpendReport(t.Context(), models.SpendReportRequest{RunID: finished.runID})
	if err != nil {
		t.Fatalf("SpendReport of the run: %v", err)
	}
	stepped := make(map[string]bool, len(byStep.Slices))
	for index := range byStep.Slices {
		stepped[byStep.Slices[index].Step] = true
	}
	for _, want := range []string{steps.NameGenerateBody, steps.NameGenerateMeta, steps.NameJudge, steps.NameRepairLinks} {
		if !stepped[want] {
			t.Errorf("the finished run spent nothing on %s; its steps are %v", want, stepped)
		}
	}
}

func stopActiveRuns(t *testing.T, core *app.Core) {
	t.Helper()

	listed, err := core.Runs.List(t.Context(), runs.ListRequest{ListRequest: dto.ListRequest{Limit: 20}})
	if err != nil {
		t.Fatalf("List runs: %v", err)
	}
	for index := range listed.Items {
		if !run.Status(listed.Items[index].Status).Active() {
			continue
		}
		if _, cancelErr := core.Runs.Cancel(t.Context(), runs.CancelRequest{RunID: listed.Items[index].ID}); cancelErr != nil {
			t.Fatalf("Cancel %s: %v", listed.Items[index].ID, cancelErr)
		}
	}
}

func openFindings(t *testing.T, core *app.Core, siteID, code string) []findings.Finding {
	t.Helper()

	out := make([]findings.Finding, 0)
	cursor := ""
	for {
		listed, err := core.Findings.List(t.Context(), findings.ListRequest{
			SiteID: siteID, Status: string(finding.Open), Code: code, Cursor: cursor, Limit: 100,
		})
		if err != nil {
			t.Fatalf("List findings: %v", err)
		}
		out = append(out, listed.Items...)
		if !listed.HasMore {
			return out
		}
		cursor = string(listed.Next)
	}
}

func TestTheSeededWorkbookFillsTheInboxTheMapAndThePreview(t *testing.T) {
	core := seeded(t)

	listed, err := core.Sites.List(t.Context(), seededSites)
	if err != nil {
		t.Fatalf("List sites: %v", err)
	}
	siteID := listed.Items[0].ID

	counts, err := core.Findings.Counts(t.Context(), findings.CountsRequest{SiteID: siteID, Status: string(finding.Open)})
	if err != nil {
		t.Fatalf("Counts: %v", err)
	}
	t.Logf("the inbox opens on %d findings: %+v", counts.Total, counts.Codes)
	for _, code := range []finding.Code{finding.RowAmbiguous, finding.RowWithoutPage, finding.ParentDiffers} {
		if held := openFindings(t, core, siteID, string(code)); len(held) == 0 {
			t.Errorf("the seeded workbook raised no %s", code)
		}
	}
	misplaced := openFindings(t, core, siteID, string(finding.ParentDiffers))
	if !slices.ContainsFunc(misplaced, func(f findings.Finding) bool { return f.PagePath == misplacedPath }) {
		t.Errorf("parent_differs = %+v, want one on %s named by its path", misplaced, misplacedPath)
	}

	workbookPath := filepath.Join(os.Getenv(app.HomeVariable), workbookName)
	preview, err := core.Imports.Preview(t.Context(), imports.PreviewRequest{SiteID: siteID, Path: workbookPath, Mapping: workbookMapping(siteID)})
	if err != nil {
		t.Fatalf("Preview the kept workbook: %v", err)
	}
	at := slices.IndexFunc(preview.Report.Pages, func(p imports.PreviewPage) bool { return p.Action == string(imports.ActionAmbiguous) })
	if at < 0 || len(preview.Report.Pages[at].Candidates) != 2 || preview.Report.Pages[at].RowKey == "" {
		t.Fatalf("the preview's pages = %+v, want the duplicated review as one row with two candidates", preview.Report.Pages)
	}
	ambiguous := preview.Report.Pages[at]
	picked, err := core.Imports.Preview(t.Context(), imports.PreviewRequest{
		SiteID: siteID, Path: workbookPath, Mapping: workbookMapping(siteID),
		Choices: []imports.Choice{{RowKey: ambiguous.RowKey, PageID: ambiguous.Candidates[0].PageID}},
	})
	if err != nil {
		t.Fatalf("Preview with the pick: %v", err)
	}
	if !slices.ContainsFunc(picked.Report.Pages, func(p imports.PreviewPage) bool {
		return p.Path == ambiguous.Candidates[0].Path && p.MatchedBy == string(pagemap.RuleMapped)
	}) {
		t.Errorf("the picked preview = %+v, want %s matched by the mapping", picked.Report.Pages, ambiguous.Candidates[0].Path)
	}

	byPath := pagesByPath(t, core, siteID)
	hub := byPath[hubPath]
	first, err := core.SiteModel.SiteMap(t.Context(), sitemodel.MapRequest{SiteID: siteID, ParentID: hub.ID, Limit: 3})
	if err != nil {
		t.Fatalf("SiteMap of the hub: %v", err)
	}
	if first.Parent == nil || first.Parent.Twin == nil || len(first.Nodes.Items) != 3 || !first.Nodes.HasMore {
		t.Fatalf("the hub's first page = %+v, want three children, more to come, and its store category beside it", first)
	}
	rest, err := core.SiteModel.SiteMap(t.Context(), sitemodel.MapRequest{SiteID: siteID, ParentID: hub.ID, Limit: 200, Cursor: first.Nodes.Next})
	if err != nil {
		t.Fatalf("SiteMap of the hub's rest: %v", err)
	}
	if got := len(first.Nodes.Items) + len(rest.Nodes.Items); got != first.Parent.Children {
		t.Errorf("the hub's children come back %d over two pages, want %d", got, first.Parent.Children)
	}

	code, many := "", []findings.Finding{}
	for _, held := range counts.Codes {
		found := openFindings(t, core, siteID, held.Code)
		if len(found) >= 2 && slices.ContainsFunc(found[0].Options, func(o findings.Option) bool { return o.Name == string(finding.Ignore) }) {
			code, many = held.Code, found
			break
		}
	}
	if code == "" {
		t.Fatalf("no code of %+v holds two findings to ignore at once", counts.Codes)
	}
	ids := make([]string, 0, len(many))
	for i := range many {
		ids = append(ids, many[i].ID)
	}
	settled, err := core.Findings.ResolveMany(t.Context(), findings.ResolveManyRequest{SiteID: siteID, Code: code, Option: string(finding.Ignore), FindingIDs: ids})
	if err != nil {
		t.Fatalf("ResolveMany: %v", err)
	}
	for _, held := range settled.Findings {
		if held.Status != string(finding.Dismissed) || held.ResolvedBy != finding.ResolvedByPerson {
			t.Errorf("%s = %+v, want it dismissed by the person", held.ID, held)
		}
	}
	if left := openFindings(t, core, siteID, code); len(left) != 0 {
		t.Errorf("%d findings %s stay open after the bulk ignore", len(left), code)
	}
	t.Logf("a bulk ignore dismissed %d findings %s", len(settled.Findings), code)
}

func pagesByPath(t *testing.T, core *app.Core, siteID string) map[string]pages.Page {
	t.Helper()

	out := make(map[string]pages.Page, 64)
	cursor := ""
	for {
		page, err := core.Pages.List(t.Context(), pages.ListRequest{
			SiteID: siteID, ListRequest: dto.ListRequest{Limit: 100, Cursor: cursor},
		})
		if err != nil {
			t.Fatalf("List pages: %v", err)
		}
		for index := range page.Items {
			out[page.Items[index].Path] = page.Items[index]
		}
		if page.Next == "" {
			return out
		}
		cursor = string(page.Next)
	}
}

func TestTheScriptAnswersASiteQuestionAndAWriteWithTheirTools(t *testing.T) {
	script := &assistantScript{}
	script.naming("page-1")

	cases := []struct {
		prompt string
		tool   string
		args   map[string]any
	}{
		{prompt: "Explore the site's tree for me", tool: "sites_explore", args: map[string]any{"view": "tree"}},
		{prompt: "Retitle the espresso page", tool: "pages_update", args: map[string]any{
			"id": "page-1", "metaTitle": "Espresso machines under $500: nine we would buy",
		}},
		{prompt: "How is the map?", tool: "", args: nil},
	}
	for _, tc := range cases {
		t.Run(tc.prompt, func(t *testing.T) {
			turn := script.answer(tc.prompt)
			if turn.Tool != tc.tool {
				t.Fatalf("answer(%q) called %q, want %q", tc.prompt, turn.Tool, tc.tool)
			}
			if turn.Text == "" {
				t.Errorf("answer(%q) says nothing after its call", tc.prompt)
			}
			if tc.args == nil {
				return
			}
			var got map[string]any
			if err := json.Unmarshal(turn.Args, &got); err != nil {
				t.Fatalf("answer(%q) args %s: %v", tc.prompt, turn.Args, err)
			}
			if !maps.Equal(got, tc.args) {
				t.Errorf("answer(%q) args = %v, want %v", tc.prompt, got, tc.args)
			}
		})
	}
}
