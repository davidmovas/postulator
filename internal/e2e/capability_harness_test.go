//go:build e2e

package e2e_test

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/llm/fake"
	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/app"
	"github.com/davidmovas/postulator/internal/application/agent"
	"github.com/davidmovas/postulator/internal/application/findings"
	port "github.com/davidmovas/postulator/internal/application/llm"
	"github.com/davidmovas/postulator/internal/application/pages"
	"github.com/davidmovas/postulator/internal/application/sites"
	"github.com/davidmovas/postulator/internal/application/tools"
	domainagent "github.com/davidmovas/postulator/internal/domain/agent"
	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/runtime/steps"
)

const (
	capabilityHub      = "/peptides/"
	capabilityCategory = "/healing-peptides/"
	capabilityBPC      = "/bpc-157/"
	capabilityTB       = "/tb-500/"
	capabilityGuides   = "/lab-guides/"
	capabilityRecon    = "/reconstitution/"
	capabilityFront    = "/"
)

type graphShape int

const (
	withoutAGraph graphShape = iota
	withTheTree
	withAFlatGraph
)

type capability struct {
	live     *site
	core     *app.Core
	script   *fake.Scripted
	pages    map[string]pages.Page
	entities map[string]string
	siteID   string
	seeds    []seededEntity
}

func openCapability(t *testing.T, script *fake.Scripted, shape graphShape) *capability {
	t.Helper()

	live := newSite(t)
	requirePlugin(t, live.env, true)
	if live.env.stand != "replica" {
		t.Skipf("the capability suite reads the replica's hub tree; run it with E2E_STAND=replica, not %s", live.env.stand)
	}
	seeds := writerStandOf(t, live.env).graph
	switch shape {
	case withoutAGraph:
		seeds = nil
	case withAFlatGraph:
		seeds = flatGraph(seeds)
	default:
	}
	core := openCoreWith(t, script)
	owner, err := core.Sites.Create(t.Context(), sites.CreateRequest{
		Name: "Capability Shop", BaseURL: live.env.baseURL, Username: live.env.user, Password: live.env.pass, AllowInsecure: true,
	})
	if err != nil {
		t.Fatalf("create the site: %v", err)
	}
	c := &capability{live: live, core: core, script: script, siteID: owner.Site.ID, seeds: seeds}
	started := time.Now()
	runSync(t, core, c.siteID)
	t.Logf("the replica synced in %s", time.Since(started).Round(time.Millisecond))
	clearTheSitesMap(t, core, c.siteID)
	c.entities = map[string]string{}
	if len(seeds) > 0 {
		c.entities = seedStandGraph(t, core, c.siteID, seeds)
	}
	c.pages = pagesByPath(t, core.Pages, c.siteID)
	return c
}

func flatGraph(seeds []seededEntity) []seededEntity {
	out := make([]seededEntity, 0, len(seeds))
	for _, seed := range seeds {
		seed.parent = ""
		out = append(out, seed)
	}
	return out
}

func (c *capability) page(t *testing.T, path string) pages.Page {
	t.Helper()

	held, found := c.pages[path]
	if !found {
		t.Fatalf("the replica holds no %s; reset the stand", path)
	}
	return held
}

func (c *capability) conversation(t *testing.T) string {
	t.Helper()
	return openConversation(t, c.core, c.siteID)
}

func (c *capability) refresh(t *testing.T) {
	t.Helper()
	c.pages = pagesByPath(t, c.core.Pages, c.siteID)
}

func argsOf(t *testing.T, args any) string {
	t.Helper()

	encoded, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("encode the arguments %+v: %v", args, err)
	}
	return string(encoded)
}

func ask[T any](t *testing.T, c *capability, conversation, tool string, args any) T {
	t.Helper()
	return decoded[T](t, callTool(t, c.core, conversation, tool, argsOf(t, args)))
}

func askACard(t *testing.T, c *capability, conversation, tool string, args any) tools.Confirmation {
	t.Helper()

	asked := ask[tools.Confirmation](t, c, conversation, tool, args)
	if asked.Status != domainagent.ConfirmationRequired || asked.ActionID == "" || asked.Summary == "" {
		t.Fatalf("%s %s answered %+v, want a card waiting for the person", tool, argsOf(t, args), asked)
	}
	return asked
}

type planStep struct {
	tool      string
	args      any
	reason    string
	dependsOn []int
}

func proposeAPlan(t *testing.T, c *capability, conversation, title string, planned ...planStep) tools.PlanView {
	t.Helper()

	rows := make([]map[string]any, 0, len(planned))
	for i := range planned {
		row := map[string]any{"tool": planned[i].tool, "args": argsOf(t, planned[i].args), "reason": planned[i].reason}
		if len(planned[i].dependsOn) > 0 {
			row["dependsOn"] = planned[i].dependsOn
		}
		rows = append(rows, row)
	}
	answered := ask[tools.PlansResponse](t, c, conversation, "plans", map[string]any{"op": "propose", "title": title, "steps": rows})
	if answered.Status != tools.PlanProposed || answered.Plan == nil || answered.Plan.Status != domainagent.PlanProposed {
		t.Fatalf("plans propose answered %+v, want a proposed plan", answered)
	}
	if len(answered.Plan.Steps) != len(planned) {
		t.Fatalf("the plan holds %d steps, want %d: %+v", len(answered.Plan.Steps), len(planned), answered.Plan.Steps)
	}
	for i := range answered.Plan.Steps {
		held := &answered.Plan.Steps[i]
		if held.Tool != planned[i].tool || held.Summary == "" || held.Status != domainagent.StepProposed {
			t.Fatalf("step %d of the plan is %+v, want a proposed %s step with its summary", i+1, *held, planned[i].tool)
		}
		t.Logf("plan step %d (%s): %s", held.Number, held.Tool, held.Summary)
	}
	return *answered.Plan
}

func approveThePlan(t *testing.T, c *capability, conversation string, plan tools.PlanView, edits map[int]map[string]string,
	declined ...int) tools.PlanView {
	t.Helper()

	choices := make([]agent.StepChoice, 0, len(plan.Steps))
	for i := range plan.Steps {
		held := &plan.Steps[i]
		choices = append(choices, agent.StepChoice{
			StepID: held.ID, Approve: !slices.Contains(declined, held.Number), Edits: edits[held.Number],
		})
	}
	confirmed, err := c.core.Agent.ConfirmPlan(t.Context(), agent.ConfirmPlanRequest{PlanID: plan.ID, Steps: choices})
	if err != nil {
		t.Fatalf("approve the plan %q: %v", plan.Title, err)
	}
	settle(t, c.core, conversation)
	return confirmed.Plan
}

func planOf(t *testing.T, c *capability, conversation, planID string) tools.PlanView {
	t.Helper()

	listed, err := c.core.Agent.ListPlans(t.Context(), agent.ListPlansRequest{ConversationID: conversation})
	if err != nil {
		t.Fatalf("list the conversation's plans: %v", err)
	}
	at := slices.IndexFunc(listed.Plans, func(view tools.PlanView) bool { return view.ID == planID })
	if at < 0 {
		t.Fatalf("the conversation holds no plan %s", planID)
	}
	return listed.Plans[at]
}

func awaitPlan(t *testing.T, c *capability, conversation, planID string) tools.PlanView {
	t.Helper()

	var held tools.PlanView
	waitFor(t, "the plan "+planID+" to finish", func() bool {
		held = planOf(t, c, conversation, planID)
		return held.Status == domainagent.PlanDone || held.Status == domainagent.PlanDeclined
	})
	settle(t, c.core, conversation)
	return held
}

func assertEveryStep(t *testing.T, plan tools.PlanView, want domainagent.StepStatus) {
	t.Helper()

	for i := range plan.Steps {
		if held := &plan.Steps[i]; held.Status != want {
			t.Errorf("step %d (%s, %s) is %s: %s, want %s", held.Number, held.Tool, held.Summary, held.Status, held.Error, want)
		}
	}
}

func toldThePlan(t *testing.T, c *capability, conversation, title string) string {
	t.Helper()

	told := ""
	chats := chatRequests(c.script)
	for i := range chats {
		if chats[i].Meta.ConversationID != conversation {
			continue
		}
		for _, message := range chats[i].Messages {
			if message.Role == port.RoleUser && strings.HasPrefix(message.Text, "The plan “"+title+"”") {
				told = message.Text
			}
		}
	}
	if told == "" {
		t.Errorf("the model of the conversation was never told how the plan %q went", title)
	}
	return told
}

var stepOrder = []string{
	steps.NameResolveContext, steps.NameGenerateBody, steps.NameGenerateMeta, steps.NameGenerateFields, steps.NameInsertLinks,
	steps.NameRepairLinks, steps.NameGenerateImages, steps.NameValidate, steps.NameJudge, steps.NamePublish,
}

func lastWritten(t *testing.T, c *capability, itemID string, kind run.ArtifactKind) string {
	t.Helper()

	stored, err := sqlite.NewArtifactRepo(c.core.Store).ByItem(t.Context(), itemID)
	if err != nil {
		t.Fatalf("read the artifacts of item %s: %v", itemID, err)
	}
	written, rank := "", -1
	for i := range stored {
		if at := slices.Index(stepOrder, stored[i].Step); stored[i].Kind == kind && at > rank {
			written, rank = string(stored[i].Blob), at
		}
	}
	if rank < 0 {
		t.Fatalf("item %s holds no %s", itemID, kind)
	}
	return written
}

func openFindings(t *testing.T, c *capability, code finding.Code) []findings.Finding {
	t.Helper()
	return findingsIn(t, c, code, finding.Open)
}

func findingsIn(t *testing.T, c *capability, code finding.Code, status finding.Status) []findings.Finding {
	t.Helper()

	out := make([]findings.Finding, 0)
	cursor := ""
	for {
		listed, err := c.core.Findings.List(t.Context(), findings.ListRequest{
			SiteID: c.siteID, Status: string(status), Code: string(code), Cursor: cursor, Limit: 200,
		})
		if err != nil {
			t.Fatalf("list the %s %s findings: %v", status, code, err)
		}
		out = append(out, listed.Items...)
		if listed.Next == "" {
			return out
		}
		cursor = string(listed.Next)
	}
}

func idsOf(held []findings.Finding) []string {
	out := make([]string, 0, len(held))
	for i := range held {
		out = append(out, held[i].ID)
	}
	return out
}
