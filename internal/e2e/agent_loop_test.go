//go:build e2e

package e2e_test

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/llm/fake"
	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/app"
	"github.com/davidmovas/postulator/internal/application/agent"
	"github.com/davidmovas/postulator/internal/application/findings"
	"github.com/davidmovas/postulator/internal/application/pages"
	"github.com/davidmovas/postulator/internal/application/tools"
	domainagent "github.com/davidmovas/postulator/internal/domain/agent"
	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/kernel/dto"
	"github.com/davidmovas/postulator/internal/runtime/steps"
)

const (
	sweptDescription = "Research peptides for the laboratory, with storage and handling notes for each compound."
	sweptMeta        = `{"title":"Research peptides | Lab","description":"` + sweptDescription + `","ogTitle":"","ogDescription":""}`
)

func sweepScript() *fake.Scripted {
	return fake.NewScripted(fake.Reply{Step: steps.NameGenerateMeta, Text: sweptMeta})
}

func openConversation(t *testing.T, core *app.Core, siteID string) string {
	t.Helper()

	opened, err := core.Agent.CreateConversation(t.Context(), agent.CreateConversationRequest{
		SiteID: siteID, Mode: string(domainagent.ModeConfirm),
	})
	if err != nil {
		t.Fatalf("open a conversation on the site: %v", err)
	}
	return opened.Conversation.ID
}

func waitingCard(t *testing.T, core *app.Core, conversation, tool string) agent.PendingAction {
	t.Helper()

	waiting, err := core.Agent.ListPendingActions(t.Context(), agent.ListPendingActionsRequest{
		ConversationID: conversation, Status: string(domainagent.ActionPending), ListRequest: dto.ListRequest{Limit: 10},
	})
	if err != nil {
		t.Fatalf("list the pending actions: %v", err)
	}
	if len(waiting.Items) != 1 || waiting.Items[0].Tool != tool {
		t.Fatalf("pending actions = %+v, want the one %s call waiting for the person", waiting.Items, tool)
	}
	return waiting.Items[0]
}

func approveTheCard(t *testing.T, core *app.Core, conversation, tool string) agent.PendingAction {
	t.Helper()

	card := waitingCard(t, core, conversation, tool)
	t.Logf("the person reads the card of %s: %s", tool, card.Summary)
	confirmed, err := core.Agent.Confirm(t.Context(), agent.ConfirmRequest{ActionID: card.ID, Approve: true})
	if err != nil {
		t.Fatalf("approve the %s call: %v", tool, err)
	}
	settle(t, core, conversation)
	if confirmed.Action.Status != string(domainagent.ActionExecuted) {
		t.Fatalf("the approved %s call is %s: %s", tool, confirmed.Action.Status, confirmed.Action.Error)
	}
	confirmed.Action.Summary = card.Summary
	return confirmed.Action
}

func decoded[T any](t *testing.T, raw json.RawMessage) T {
	t.Helper()

	var out T
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("decode %s: %v", raw, err)
	}
	return out
}

type findingsTurn struct {
	core         *app.Core
	conversation string
}

func (f findingsTurn) resolve(t *testing.T, args string) {
	t.Helper()

	asked := decoded[tools.Confirmation](t, callTool(t, f.core, f.conversation, "findings", args))
	if asked.Status != domainagent.ConfirmationRequired || !strings.Contains(asked.Summary, "nothing is written to the site") {
		t.Fatalf("findings %s = %+v, want a card that says the site is untouched", args, asked)
	}
	approveTheCard(t, f.core, f.conversation, "findings")
}

func theAgentWorksTheFindings(t *testing.T, core *app.Core, script *fake.Scripted, siteID string, open []findings.Finding,
	stored map[string]pages.Page) {
	t.Helper()

	conversation := openConversation(t, core, siteID)
	turn := findingsTurn{core: core, conversation: conversation}
	guide, bpc, reconstitution := stored["/bpc-157/bpc-157-dosing-guide/"], stored["/bpc-157/"], stored["/reconstitution/"]

	listed := decoded[tools.FindingsResponse](t, callTool(t, core, conversation, "findings", `{"op":"list","code":"row_without_page","limit":200}`))
	if listed.List == nil || len(listed.List.Items) == 0 {
		t.Fatalf("the agent listed %+v, want the rows still waiting for a page", listed)
	}
	t.Logf("step 1: the agent lists %d open row_without_page findings, read without a card", len(listed.List.Items))
	rowless, found := onPage(listed.List.Items, string(finding.RowWithoutPage), guide.ID)
	if !found {
		t.Fatalf("the agent's list holds no row for the dosing guide: %+v", listed.List.Items)
	}

	turn.resolve(t, `{"op":"resolve","findingId":"`+rowless.ID+`","option":"map_to_page","pageId":"`+bpc.ID+`"}`)
	mapped, err := core.Findings.Get(t.Context(), findings.GetRequest{SiteID: siteID, FindingID: rowless.ID})
	if err != nil || mapped.Status != string(finding.Resolved) || mapped.ResolvedBy != finding.ResolvedByAgent || mapped.PageID != bpc.ID {
		t.Errorf("the dosing guide's row after the agent mapped it = %+v, %v, want it settled by the agent on /bpc-157/", mapped, err)
	}
	if _, kept := pagesByPath(t, core.Pages, siteID)["/bpc-157/bpc-157-dosing-guide/"]; kept {
		t.Error("the planned dosing guide is still in the model after its row moved onto /bpc-157/")
	}
	t.Logf("step 2: the agent maps the dosing guide's row onto /bpc-157/, the person confirms; the planned page goes")

	differs, found := onPage(open, string(finding.ParentDiffers), reconstitution.ID)
	if !found {
		t.Fatalf("Reconstitution carries no open parent_differs")
	}
	turn.resolve(t, `{"op":"resolve","findingId":"`+differs.ID+`","option":"accept_workbook"}`)
	pinned, err := core.Findings.Get(t.Context(), findings.GetRequest{SiteID: siteID, FindingID: differs.ID})
	if err != nil || pinned.Status != string(finding.Resolved) || pinned.ResolvedBy != finding.ResolvedByAgent {
		t.Errorf("Reconstitution's parent_differs after the agent pinned it = %+v, %v", pinned, err)
	}
	t.Logf("step 3: the agent pins Reconstitution's parent to the workbook, the person confirms")

	for i := range open {
		run := slices.IndexFunc(open[i].Options, func(option findings.Option) bool { return option.Kind == string(finding.Run) })
		if run < 0 || open[i].ID == differs.ID || open[i].ID == rowless.ID {
			continue
		}
		prepared := decoded[tools.FindingsResponse](t, callTool(t, core, conversation, "findings",
			`{"op":"resolve","findingId":"`+open[i].ID+`","option":"`+open[i].Options[run].Name+`"}`))
		if prepared.Resolve == nil || (prepared.Resolve.Start == nil && prepared.Resolve.Sync == nil) || prepared.Resolve.Finding.Status != string(finding.Open) {
			t.Errorf("a run option on %s = %+v, want the request for the normal start and the finding open", open[i].Code, prepared)
		}
		t.Logf("step 4: %s on %s answers a request at once, no card: start %+v sync %+v",
			open[i].Options[run].Name, open[i].Code, prepared.Resolve.Start, prepared.Resolve.Sync)
		break
	}

	theAgentSweepsTheMeta(t, core, script, conversation, siteID, []pages.Page{stored["/peptides/"], reconstitution})
}

func theAgentSweepsTheMeta(t *testing.T, core *app.Core, script *fake.Scripted, conversation, siteID string, swept []pages.Page) {
	t.Helper()

	client := wordpress(t, core, siteID)
	bodies := make(map[string]string, len(swept))
	ids := make([]string, 0, len(swept))
	for i := range swept {
		held, err := client.GetRaw(t.Context(), wp.ItemType(swept[i].WPType), *swept[i].WPID)
		if err != nil {
			t.Fatalf("read %s: %v", swept[i].Path, err)
		}
		bodies[swept[i].ID] = held.Content
		ids = append(ids, `"`+swept[i].ID+`"`)
	}
	asked := decoded[tools.Confirmation](t, callTool(t, core, conversation, "runs_control",
		`{"op":"start","pageIds":[`+strings.Join(ids, ",")+`],"fields":["seo_description"],"publishMode":"publish"}`))
	if !strings.Contains(asked.Summary, "seo_description") || !strings.Contains(asked.Summary, "estimated") {
		t.Fatalf("the sweep's card = %+v, want the scope and the estimate on it", asked)
	}
	started := decoded[map[string]json.RawMessage](t, approveTheCard(t, core, conversation, "runs_control").Result)
	runID := decoded[string](t, started["runId"])
	awaitRun(t, core.Runs, runID)
	t.Logf("step 5: the agent starts a meta sweep over %d pages, the person confirms the card with its estimate (%s)",
		len(swept), asked.Summary)
	if drafts, metas := script.CallsTo(steps.NameGenerateBody), script.CallsTo(steps.NameGenerateMeta); drafts != 0 || metas != len(swept) {
		t.Errorf("the sweep asked the model for %d bodies and %d meta answers, want none and one per page", drafts, metas)
	}

	for i := range swept {
		held, err := client.GetRaw(t.Context(), wp.ItemType(swept[i].WPType), *swept[i].WPID)
		if err != nil || held.Content != bodies[swept[i].ID] {
			t.Errorf("%s after the sweep holds %q (%v), want its body untouched", swept[i].Path, held.Content, err)
		}
		meta, err := client.GetSEOMeta(t.Context(), wp.ItemType(swept[i].WPType), *swept[i].WPID)
		if err != nil || meta.Description != sweptDescription {
			t.Errorf("%s carries the meta description %q (%v), want the sweep's", swept[i].Path, meta.Description, err)
		}
	}
}
