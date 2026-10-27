package agent_test

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	agentapp "github.com/davidmovas/postulator/internal/application/agent"
	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/application/tools"
	domainagent "github.com/davidmovas/postulator/internal/domain/agent"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

func (h *harness) propose(t *testing.T, conversationID, steps string) tools.PlanView {
	t.Helper()

	out, err := h.registry.Call(t.Context(), tools.Binding{SiteID: h.siteID, ConversationID: conversationID, Mode: domainagent.ModeConfirm},
		"plans", json.RawMessage(`{"op":"propose","title":"Plan the serums section","steps":`+steps+`}`))
	if err != nil {
		t.Fatalf("propose: %v", err)
	}
	answer, ok := out.(tools.PlansResponse)
	if !ok || answer.Plan == nil {
		t.Fatalf("propose answered %+v", out)
	}
	return *answer.Plan
}

func jsonOf(t *testing.T, value any) []byte {
	t.Helper()

	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("encode %v: %v", value, err)
	}
	return encoded
}

func createStep(t *testing.T, path, reason string, dependsOn ...int) string {
	t.Helper()

	args := jsonOf(t, map[string]string{"op": "create", "path": path, "title": strings.Trim(path, "/")})
	step := map[string]any{"tool": "pages_manage", "args": string(args), "reason": reason}
	if len(dependsOn) > 0 {
		step["dependsOn"] = dependsOn
	}
	return string(jsonOf(t, step))
}

func (h *harness) planned(t *testing.T, path string) bool {
	t.Helper()

	held, err := h.pages.ListBySite(t.Context(), h.siteID)
	if err != nil {
		t.Fatalf("list the pages: %v", err)
	}
	return slices.ContainsFunc(held, func(page pagemap.Page) bool { return page.Path == path })
}

func stepStatuses(view tools.PlanView) []domainagent.StepStatus {
	out := make([]domainagent.StepStatus, 0, len(view.Steps))
	for i := range view.Steps {
		out = append(out, view.Steps[i].Status)
	}
	return out
}

func choices(view tools.PlanView, approve ...int) []agentapp.StepChoice {
	out := make([]agentapp.StepChoice, 0, len(view.Steps))
	for i := range view.Steps {
		out = append(out, agentapp.StepChoice{StepID: view.Steps[i].ID, Approve: slices.Contains(approve, i+1)})
	}
	return out
}

func TestAnApprovedPlanRunsInOrderAndAFailureSkipsWhatDependsOnIt(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	conversation := h.conversation(t, domainagent.ModeConfirm)
	proposed := h.propose(t, conversation, "["+strings.Join([]string{
		createStep(t, "/serums/", "The hub the section hangs from"),
		createStep(t, "/serums/", "A second hub at the same address, which the page map refuses", 1),
		createStep(t, "/serums/retinol/", "A page under the second hub", 2),
		createStep(t, "/oils/", "A hub of its own"),
	}, ",")+"]")
	if h.planned(t, "/serums/") {
		t.Fatal("a proposed plan wrote before it was approved")
	}

	confirmed, err := h.service.ConfirmPlan(t.Context(), agentapp.ConfirmPlanRequest{PlanID: proposed.ID, Steps: choices(proposed, 1, 2, 3, 4)})
	if err != nil {
		t.Fatalf("ConfirmPlan: %v", err)
	}
	want := []domainagent.StepStatus{domainagent.StepDone, domainagent.StepFailed, domainagent.StepSkipped, domainagent.StepDone}
	if got := stepStatuses(confirmed.Plan); !slices.Equal(got, want) || confirmed.Plan.Status != domainagent.PlanDone {
		t.Fatalf("the plan ran to %s %v, want done %v", confirmed.Plan.Status, got, want)
	}
	if confirmed.Plan.Steps[1].Error == "" || !strings.Contains(confirmed.Plan.Steps[2].Error, "step 2") {
		t.Errorf("the failure and the skip = %q / %q, want each to say why", confirmed.Plan.Steps[1].Error, confirmed.Plan.Steps[2].Error)
	}
	if !h.planned(t, "/serums/") || !h.planned(t, "/oils/") || h.planned(t, "/serums/retinol/") {
		t.Fatal("the pages the plan left behind are not the ones its steps landed")
	}
	if !slices.Contains(h.seen(), events.AgentPlanUpdated) {
		t.Errorf("no plan update reached the window: %v", h.seen())
	}
	h.settled(t)
	told := false
	for _, req := range h.model.Requests() {
		for _, message := range req.Messages {
			if strings.Contains(message.Text, "“Plan the serums section” has run") && strings.Contains(message.Text, "step 3") {
				told = true
			}
		}
	}
	if !told {
		t.Error("the model was never told what the plan did, step by step")
	}
	if _, err = h.service.ConfirmPlan(t.Context(), agentapp.ConfirmPlanRequest{PlanID: proposed.ID, Steps: choices(proposed, 4)}); err == nil {
		t.Error("a plan was approved twice")
	}
}

func TestADeclinedStepTakesItsDependantsWithItAndRunsNothingOfThem(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	conversation := h.conversation(t, domainagent.ModeConfirm)
	proposed := h.propose(t, conversation, "["+createStep(t, "/serums/", "hub")+","+createStep(t, "/serums/retinol/", "child", 1)+","+
		createStep(t, "/oils/", "another hub")+"]")
	confirmed, err := h.service.ConfirmPlan(t.Context(), agentapp.ConfirmPlanRequest{PlanID: proposed.ID, Steps: choices(proposed, 2, 3)})
	if err != nil {
		t.Fatalf("ConfirmPlan: %v", err)
	}
	want := []domainagent.StepStatus{domainagent.StepDeclined, domainagent.StepDeclined, domainagent.StepDone}
	if got := stepStatuses(confirmed.Plan); !slices.Equal(got, want) {
		t.Fatalf("the steps = %v, want %v", got, want)
	}
	if h.planned(t, "/serums/retinol/") || !h.planned(t, "/oils/") {
		t.Fatal("a declined step's dependant ran, or the chosen step did not")
	}
}

func (h *harness) storedPlan(t *testing.T, conversationID string, steps ...domainagent.PlanStep) domainagent.Plan {
	t.Helper()

	plan, err := domainagent.NewPlan(domainagent.Plan{
		ID: id.New(), ConversationID: conversationID, SiteID: h.siteID, Title: "Relink and then plan", Status: domainagent.PlanRunning,
		Steps: steps, CreatedAt: sqlitetest.Stamp, UpdatedAt: sqlitetest.Stamp,
	})
	if err != nil {
		t.Fatalf("NewPlan: %v", err)
	}
	if err = sqlite.NewPlanRepo(h.store).Insert(t.Context(), plan); err != nil {
		t.Fatalf("insert the plan: %v", err)
	}
	return plan
}

func planStep(t *testing.T, status domainagent.StepStatus, path string, dependsOn ...int) domainagent.PlanStep {
	t.Helper()

	return domainagent.PlanStep{
		ID: id.New(), Tool: "pages_manage", Args: jsonOf(t, map[string]string{"op": "create", "path": path}), Reason: "r", Summary: "Plan " + path, Status: status, DependsOn: dependsOn,
	}
}

func TestAStepWaitingOnItsRunGoesOnWhenTheRunEnds(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	conversation := h.conversation(t, domainagent.ModeConfirm)
	waiting := planStep(t, domainagent.StepWaiting, "/relinked/")
	waiting.RunID = "run-landed"
	landed := h.storedPlan(t, conversation, waiting, planStep(t, domainagent.StepPending, "/after-the-run/", 0))
	failing := planStep(t, domainagent.StepWaiting, "/relinked-too/")
	failing.RunID = "run-cancelled"
	stopped := h.storedPlan(t, conversation, failing, planStep(t, domainagent.StepPending, "/never/", 0))

	if err := h.service.RunEnded(t.Context(), agentapp.RunOutcome{RunID: "run-landed"}); err != nil {
		t.Fatalf("RunEnded: %v", err)
	}
	if err := h.service.RunEnded(t.Context(), agentapp.RunOutcome{RunID: "run-cancelled", Failure: "the run was cancelled"}); err != nil {
		t.Fatalf("RunEnded: %v", err)
	}
	plans := sqlite.NewPlanRepo(h.store)
	done, err := plans.Get(t.Context(), landed.ID)
	if err != nil || done.Status != domainagent.PlanDone || done.Steps[1].Status != domainagent.StepDone || !h.planned(t, "/after-the-run/") {
		t.Fatalf("the plan behind a landed run = %+v, %v", done, err)
	}
	skipped, err := plans.Get(t.Context(), stopped.ID)
	if err != nil || skipped.Steps[0].Status != domainagent.StepFailed || skipped.Steps[1].Status != domainagent.StepSkipped || h.planned(t, "/never/") {
		t.Fatalf("the plan behind a cancelled run = %+v, %v", skipped, err)
	}
}

func TestAStepTheAppStoppedInIsSettledWhenItOpensAgain(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	conversation := h.conversation(t, domainagent.ModeConfirm)
	interrupted := h.storedPlan(t, conversation, planStep(t, domainagent.StepRunning, "/half-done/"), planStep(t, domainagent.StepPending, "/after/", 0),
		planStep(t, domainagent.StepPending, "/beside/"))
	if err := h.service.ResumePlans(t.Context()); err != nil {
		t.Fatalf("ResumePlans: %v", err)
	}
	resumed, err := sqlite.NewPlanRepo(h.store).Get(t.Context(), interrupted.ID)
	if err != nil {
		t.Fatalf("read the plan: %v", err)
	}
	want := []domainagent.StepStatus{domainagent.StepFailed, domainagent.StepSkipped, domainagent.StepDone}
	got := make([]domainagent.StepStatus, 0, len(resumed.Steps))
	for _, held := range resumed.Steps {
		got = append(got, held.Status)
	}
	if !slices.Equal(got, want) || !strings.Contains(resumed.Steps[0].Error, "stopped") || resumed.Status != domainagent.PlanDone {
		t.Fatalf("the plan after a restart = %s %v (%q), want done %v", resumed.Status, got, resumed.Steps[0].Error, want)
	}
}
