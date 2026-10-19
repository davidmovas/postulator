//go:build e2e

package e2e_test

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/llm/fake"
	"github.com/davidmovas/postulator/internal/app"
	"github.com/davidmovas/postulator/internal/application/agent"
	"github.com/davidmovas/postulator/internal/application/graph"
	port "github.com/davidmovas/postulator/internal/application/llm"
	"github.com/davidmovas/postulator/internal/application/pages"
	"github.com/davidmovas/postulator/internal/application/reports"
	"github.com/davidmovas/postulator/internal/application/sitemodel"
	"github.com/davidmovas/postulator/internal/application/sites"
	domainagent "github.com/davidmovas/postulator/internal/domain/agent"
	"github.com/davidmovas/postulator/internal/domain/contextpack"
	domainllm "github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/kernel/dto"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	agentHub   = "/peptides/"
	agentChild = "/healing-peptides/"
	agentRoot  = "/"
	agentNotes = "/peptides/storage-notes/"

	exploreTool  = "sites_explore"
	proposeTool  = "graph_propose_cards"
	briefOpening = "THE SITE WHEN THIS CONVERSATION BEGAN"

	autonomousDefinition = "Peptides are short chains of amino acids, sold here for laboratory research only."
	confirmedDefinition  = "Healing peptides are the research peptides studied for tissue repair."
)

func TestTheAgentSeesTheSite(t *testing.T) {
	live := newSite(t)
	requirePlugin(t, live.env, true)
	if live.env.stand != "replica" {
		t.Skipf("the agent's proof reads the replica's hub tree; run it with STAND=replica, not %s", live.env.stand)
	}
	stand := writerStandOf(t, live.env)

	script := fake.NewScripted()
	core := openCoreWith(t, script)
	owner, err := core.Sites.Create(t.Context(), sites.CreateRequest{
		Name: "Explored Shop", BaseURL: live.env.baseURL, Username: live.env.user, Password: live.env.pass, AllowInsecure: true,
	})
	if err != nil {
		t.Fatalf("create the site: %v", err)
	}
	siteID := owner.Site.ID
	runSync(t, core, siteID)
	entities := seedStandGraph(t, core, siteID, stand.graph)

	opened, err := core.Agent.CreateConversation(t.Context(), agent.CreateConversationRequest{
		SiteID: siteID, Mode: string(domainagent.ModeAutonomous),
	})
	if err != nil {
		t.Fatalf("open a conversation on the site: %v", err)
	}
	conversation := opened.Conversation.ID

	brief := assertTheBriefStays(t, core, script, conversation, siteID)
	t.Logf("the replica's brief, %d bytes:\n%s", len(brief), brief)

	assertEveryViewAnswersFromTheModel(t, core, conversation)
	assertACardWaitsForAPerson(t, core, conversation, siteID, entities["Peptides"])
	assertAConfirmedCardIsWritten(t, core, script, conversation, siteID, entities["Healing Peptides"])
}

func assertTheBriefStays(t *testing.T, core *app.Core, script *fake.Scripted, conversation, siteID string) string {
	t.Helper()

	built, err := core.SiteModel.Brief(t.Context(), siteID)
	if err != nil {
		t.Fatalf("build the site's brief: %v", err)
	}
	brief := built.Text()
	hubs, _ := sectionOf(brief, "HUBS")
	if !slices.ContainsFunc(hubs, func(line string) bool { return strings.HasPrefix(strings.TrimSpace(line), "- "+agentHub+" ") }) {
		t.Errorf("the brief's hub tree %v does not name %s", hubs, agentHub)
	}
	findings, _ := sectionOf(brief, "OPEN FINDINGS")
	if !slices.ContainsFunc(findings, func(line string) bool { return strings.HasPrefix(line, "- ") }) {
		t.Errorf("the brief names no open finding: %v", findings)
	}

	talk(t, core, conversation, fake.FinalDirective+" The site is read.")
	chats := chatRequests(script)
	if len(chats) == 0 {
		t.Fatal("the conversation sent the model nothing")
	}
	first := chats[0]
	if !strings.Contains(first.System, briefOpening) || !strings.Contains(first.System, strings.TrimSpace(brief)) {
		t.Errorf("the first request's instructions do not carry the brief:\n%s", first.System)
	}
	if want := "chat:" + string(domainagent.ModeAutonomous) + ":" + siteID; first.CacheKey != want {
		t.Errorf("the first request names the cache key %q, want %q", first.CacheKey, want)
	}

	if _, err = core.Pages.Create(t.Context(), pages.CreateRequest{
		SiteID: siteID, Path: agentNotes, Title: "Storage Notes", Status: "planned",
	}); err != nil {
		t.Fatalf("plan %s: %v", agentNotes, err)
	}
	moved, err := core.SiteModel.Brief(t.Context(), siteID)
	if err != nil {
		t.Fatalf("build the site's brief again: %v", err)
	}
	if moved.Text() == brief {
		t.Fatalf("planning %s left the site's brief as it was, so a second turn proves nothing", agentNotes)
	}

	talk(t, core, conversation, fake.FinalDirective+" Still the same site.")
	later := chatRequests(script)
	if len(later) <= len(chats) {
		t.Fatal("the second turn sent the model nothing")
	}
	if second := later[len(chats)]; second.System != first.System || second.CacheKey != first.CacheKey {
		t.Errorf("the second turn's instructions moved with the model; the brief is kept from the conversation's start:\n%s",
			second.System)
	}
	return brief
}

func assertEveryViewAnswersFromTheModel(t *testing.T, core *app.Core, conversation string) {
	t.Helper()

	page := explore(t, core, conversation, `{"view":"page","path":"`+agentHub+`"}`).Page
	if page == nil || page.Page.Path != agentHub || len(page.Products) == 0 || (page.ListedFrom != "listing" && page.ListedFrom != "twin") {
		t.Errorf("the page view of %s is %+v, want its products from its listing or its twin's term", agentHub, page)
	}

	top := explore(t, core, conversation, `{"view":"tree"}`).Tree
	if top == nil || !treeHolds(top, agentRoot) {
		t.Errorf("the top of the tree is %+v, want a row for %s", top, agentRoot)
	}
	underRoot := explore(t, core, conversation, `{"view":"tree","path":"`+agentRoot+`"}`).Tree
	if underRoot == nil || !treeHolds(underRoot, agentHub) {
		t.Errorf("the tree under %s is %+v, want a row for %s", agentRoot, underRoot, agentHub)
	}
	underHub := explore(t, core, conversation, `{"view":"tree","path":"`+agentHub+`"}`).Tree
	if underHub == nil || underHub.Parent == nil || underHub.Parent.Path != agentHub || !treeHolds(underHub, agentChild) {
		t.Errorf("the tree under %s is %+v, want a row for %s", agentHub, underHub, agentChild)
	}

	links := explore(t, core, conversation, `{"view":"links","path":"`+agentHub+`"}`).Links
	if links == nil || links.Page.Path != agentHub || !slices.ContainsFunc(links.Required, func(owed reports.RequiredLink) bool {
		return owed.TargetPath == agentChild
	}) {
		t.Errorf("the links view of %s is %+v, want the link it owes %s", agentHub, links, agentChild)
	}

	gaps := explore(t, core, conversation, `{"view":"gaps"}`).Gaps
	if gaps == nil || len(gaps.Items) == 0 || !slices.ContainsFunc(gaps.Items, func(row contextpack.GapRow) bool { return len(row.Gaps) > 0 }) {
		t.Errorf("the gaps view is %+v, want the pages with a gap", gaps)
	}

	pack := explore(t, core, conversation, `{"view":"pack","path":"`+agentHub+`"}`).Pack
	if pack == nil || pack.Path != agentHub || pack.Tokens == 0 || !namesProducts(pack.Text) {
		t.Fatalf("the pack view of %s is %+v, want the writer's pack with its products", agentHub, pack)
	}
	t.Logf("the replica's %s %s pack through %s, %d tokens of a %d-token budget, dropped %+v:\n%s",
		pack.Path, pack.Field, exploreTool, pack.Tokens, pack.Budget, pack.Dropped, pack.Text)
}

func assertACardWaitsForAPerson(t *testing.T, core *app.Core, conversation, siteID, entityID string) {
	t.Helper()

	before := entityOf(t, core, entityID)
	result := callTool(t, core, conversation, proposeTool, cardsArgs(entityID, autonomousDefinition))
	var proposed graph.ProposeCardsResponse
	if err := json.Unmarshal(result, &proposed); err != nil || len(proposed.Proposals) != 1 {
		t.Fatalf("the autonomous %s answered %s (%v), want one proposal filed", proposeTool, result, err)
	}
	if held := entityOf(t, core, entityID); held.Definition != before.Definition || len(held.Facts) != len(before.Facts) {
		t.Fatalf("a proposal wrote the card of %s: %+v", before.Name, held)
	}

	filed := proposalFor(t, core, siteID, entityID)
	if filed.Proposed.Definition != autonomousDefinition {
		t.Fatalf("the filed proposal is %+v, want the definition the agent proposed", filed)
	}
	if _, err := core.Graph.AcceptCard(t.Context(), graph.AcceptCardRequest{FindingID: filed.FindingID, ProposalHash: filed.ProposalHash}); err != nil {
		t.Fatalf("accept the proposed card: %v", err)
	}
	if held := entityOf(t, core, entityID); held.Definition != autonomousDefinition || len(held.Facts) == 0 {
		t.Errorf("after the person accepted it the card of %s is %+v", before.Name, held)
	}
}

func assertAConfirmedCardIsWritten(t *testing.T, core *app.Core, script *fake.Scripted, conversation, siteID, entityID string) {
	t.Helper()

	if _, err := core.Agent.SetMode(t.Context(), agent.SetModeRequest{
		ConversationID: conversation, Mode: string(domainagent.ModeConfirm),
	}); err != nil {
		t.Fatalf("switch the conversation to confirm mode: %v", err)
	}
	asked := len(chatRequests(script))
	before := entityOf(t, core, entityID)
	callTool(t, core, conversation, proposeTool, cardsArgs(entityID, confirmedDefinition))
	if chats := chatRequests(script); len(chats) > asked {
		if want := "chat:" + string(domainagent.ModeConfirm) + ":" + siteID; chats[asked].CacheKey != want {
			t.Errorf("the confirm-mode request names the cache key %q, want %q", chats[asked].CacheKey, want)
		}
	}

	waiting, err := core.Agent.ListPendingActions(t.Context(), agent.ListPendingActionsRequest{
		ConversationID: conversation, Status: string(domainagent.ActionPending), ListRequest: dto.ListRequest{Limit: 10},
	})
	if err != nil {
		t.Fatalf("list the pending actions: %v", err)
	}
	if len(waiting.Items) != 1 || waiting.Items[0].Tool != proposeTool {
		t.Fatalf("pending actions = %+v, want the one card call waiting for a person", waiting.Items)
	}
	if held := entityOf(t, core, entityID); held.Definition != before.Definition {
		t.Fatalf("a call waiting for approval wrote the card of %s: %+v", before.Name, held)
	}
	if proposals := openProposals(t, core, siteID); slices.ContainsFunc(proposals, func(p graph.CardProposal) bool { return p.EntityID == entityID }) {
		t.Fatalf("a call waiting for approval filed a proposal for %s", before.Name)
	}

	confirmed, err := core.Agent.Confirm(t.Context(), agent.ConfirmRequest{ActionID: waiting.Items[0].ID, Approve: true})
	if err != nil {
		t.Fatalf("approve the card call: %v", err)
	}
	settle(t, core, conversation)
	if confirmed.Action.Status != string(domainagent.ActionExecuted) {
		t.Fatalf("the approved call is %s: %s", confirmed.Action.Status, confirmed.Action.Error)
	}
	if held := entityOf(t, core, entityID); held.Definition != confirmedDefinition || len(held.Facts) == 0 {
		t.Errorf("after the person approved the call the card of %s is %+v", before.Name, held)
	}
}

func talk(t *testing.T, core *app.Core, conversation, text string) {
	t.Helper()

	waitFor(t, "the conversation to take the message", func() bool {
		_, err := core.Agent.Send(t.Context(), agent.SendRequest{ConversationID: conversation, Text: text})
		if errors.IsCode(err, errors.Conflict) {
			return false
		}
		if err != nil {
			t.Fatalf("send %q: %v", text, err)
		}
		return true
	})
	settle(t, core, conversation)
}

func settle(t *testing.T, core *app.Core, conversation string) {
	t.Helper()

	waitFor(t, "the agent to answer", func() bool {
		status, err := core.Agent.Status(t.Context(), agent.StatusRequest{ConversationID: conversation})
		if err != nil {
			t.Fatalf("read the conversation's status: %v", err)
		}
		return !status.Running
	})
}

func chatRequests(script *fake.Scripted) []port.Request {
	requests := script.Requests()
	return slices.DeleteFunc(requests, func(req port.Request) bool { return req.Meta.Step != domainllm.StepChat })
}

func toolMessages(t *testing.T, core *app.Core, conversation, tool string) []agent.Message {
	t.Helper()

	out := make([]agent.Message, 0)
	cursor := ""
	for {
		listed, err := core.Agent.ListMessages(t.Context(), agent.ListMessagesRequest{
			ConversationID: conversation, ListRequest: dto.ListRequest{Limit: 100, Cursor: cursor},
		})
		if err != nil {
			t.Fatalf("list the conversation's messages: %v", err)
		}
		for i := range listed.Items {
			if message := &listed.Items[i]; message.Role == string(domainagent.RoleTool) && message.Tool == tool {
				out = append(out, *message)
			}
		}
		if listed.Next == "" {
			return out
		}
		cursor = string(listed.Next)
	}
}

func callTool(t *testing.T, core *app.Core, conversation, tool, args string) json.RawMessage {
	t.Helper()

	seen := len(toolMessages(t, core, conversation, tool))
	talk(t, core, conversation, fake.ToolDirective+tool+args)
	answered := toolMessages(t, core, conversation, tool)
	if len(answered) != seen+1 {
		t.Fatalf("the turn ran %s %d time(s), want once", tool, len(answered)-seen)
	}
	message := answered[len(answered)-1]
	if message.ToolStatus != string(domainagent.CallOK) {
		t.Fatalf("%s%s answered %s: %s", tool, args, message.ToolStatus, message.Text)
	}
	return message.Payload
}

func explore(t *testing.T, core *app.Core, conversation, args string) sitemodel.ExploreResponse {
	t.Helper()

	payload := callTool(t, core, conversation, exploreTool, args)
	var answered sitemodel.ExploreResponse
	if err := json.Unmarshal(payload, &answered); err != nil {
		t.Fatalf("decode what %s%s answered: %v: %s", exploreTool, args, err, payload)
	}
	return answered
}

func namesProducts(text string) bool {
	_, listed := sectionOf(text, headingListed)
	_, twin := sectionOf(text, headingTwin)
	return listed || twin
}

func treeHolds(tree *sitemodel.TreeView, path string) bool {
	return slices.ContainsFunc(tree.Nodes.Items, func(node contextpack.Node) bool { return node.Page.Path == path })
}

func cardsArgs(entityID, definition string) string {
	encoded, err := json.Marshal(map[string]any{"cards": []map[string]any{{
		"entityId": entityID, "definition": definition, "reason": "Read from the hub's text and its products.",
		"facts": []map[string]string{{"label": "Storage", "value": "Frozen, sealed under nitrogen"}},
	}}})
	if err != nil {
		return "{}"
	}
	return string(encoded)
}

func entityOf(t *testing.T, core *app.Core, entityID string) graph.Entity {
	t.Helper()

	held, err := core.Graph.GetEntity(t.Context(), graph.GetEntityRequest{ID: entityID})
	if err != nil {
		t.Fatalf("read the entity %s: %v", entityID, err)
	}
	return held.Entity
}

func openProposals(t *testing.T, core *app.Core, siteID string) []graph.CardProposal {
	t.Helper()

	listed, err := core.Graph.CardProposals(t.Context(), graph.CardProposalsRequest{SiteID: siteID, Limit: 50})
	if err != nil {
		t.Fatalf("list the card proposals: %v", err)
	}
	return listed.Items
}

func proposalFor(t *testing.T, core *app.Core, siteID, entityID string) graph.CardProposal {
	t.Helper()

	proposals := openProposals(t, core, siteID)
	at := slices.IndexFunc(proposals, func(p graph.CardProposal) bool { return p.EntityID == entityID })
	if at < 0 {
		t.Fatalf("no open proposal names the entity %s: %+v", entityID, proposals)
	}
	return proposals[at]
}
