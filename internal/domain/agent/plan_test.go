package agent_test

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/domain/agent"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

var planned = time.Date(2026, time.October, 26, 12, 0, 0, 0, time.UTC)

func step(id, tool string, dependsOn ...int) agent.PlanStep {
	return agent.PlanStep{ID: id, Tool: tool, Args: json.RawMessage(`{}`), Reason: "because", Summary: tool, DependsOn: dependsOn}
}

func linkPlan(t *testing.T) agent.Plan {
	t.Helper()

	plan, err := agent.NewPlan(agent.Plan{
		ID: "plan-1", ConversationID: "conv-1", SiteID: "site-1", Title: "Link the serums section",
		Steps: []agent.PlanStep{
			step("edge", "graph_edit"),
			step("relink", "runs_control", 0),
			step("audit", "pages_judge", 1),
			step("type", "pagetypes"),
		},
		CreatedAt: planned, UpdatedAt: planned,
	})
	if err != nil {
		t.Fatalf("NewPlan: %v", err)
	}
	return plan
}

func statuses(plan agent.Plan) []agent.StepStatus {
	out := make([]agent.StepStatus, 0, len(plan.Steps))
	for _, held := range plan.Steps {
		out = append(out, held.Status)
	}
	return out
}

func TestAPlanIsHeldToItsShape(t *testing.T) {
	t.Parallel()

	if plan := linkPlan(t); plan.Status != agent.PlanProposed || !slices.Equal(statuses(plan), []agent.StepStatus{
		agent.StepProposed, agent.StepProposed, agent.StepProposed, agent.StepProposed,
	}) {
		t.Fatalf("a new plan = %s %v", plan.Status, statuses(plan))
	}
	many := make([]agent.PlanStep, agent.MaxPlanSteps+1)
	for i := range many {
		many[i] = step("s"+strings.Repeat("x", i), "graph_edit")
	}
	cases := map[string]agent.Plan{
		"no steps":                {ID: "p", ConversationID: "c", Title: "t"},
		"no title":                {ID: "p", ConversationID: "c", Steps: []agent.PlanStep{step("a", "x")}},
		"no reason":               {ID: "p", ConversationID: "c", Title: "t", Steps: []agent.PlanStep{{ID: "a", Tool: "x", Args: json.RawMessage(`{}`)}}},
		"a dependency forward":    {ID: "p", ConversationID: "c", Title: "t", Steps: []agent.PlanStep{step("a", "x", 1), step("b", "x")}},
		"a dependency on itself":  {ID: "p", ConversationID: "c", Title: "t", Steps: []agent.PlanStep{step("a", "x", 0)}},
		"two steps of one id":     {ID: "p", ConversationID: "c", Title: "t", Steps: []agent.PlanStep{step("a", "x"), step("a", "y")}},
		"args that are no object": {ID: "p", ConversationID: "c", Title: "t", Steps: []agent.PlanStep{{ID: "a", Tool: "x", Reason: "r", Args: json.RawMessage(`[1]`)}}},
		"more steps than a card":  {ID: "p", ConversationID: "c", Title: "t", Steps: many},
	}
	for name, plan := range cases {
		if _, err := agent.NewPlan(plan); !errors.IsCode(err, errors.Invalid) {
			t.Errorf("%s = %v, want INVALID", name, err)
		}
	}
}

func TestADeclinedStepTakesItsDependantsWithIt(t *testing.T) {
	t.Parallel()

	plan := linkPlan(t)
	if err := plan.Approve(map[string]bool{"edge": false, "relink": true, "audit": true, "type": true}); err != nil {
		t.Fatalf("Approve: %v", err)
	}
	want := []agent.StepStatus{agent.StepDeclined, agent.StepDeclined, agent.StepDeclined, agent.StepPending}
	if got := statuses(plan); !slices.Equal(got, want) || plan.Status != agent.PlanRunning {
		t.Fatalf("after declining the edge = %s %v, want %v", plan.Status, got, want)
	}
	if err := plan.Approve(map[string]bool{"edge": true}); !errors.IsCode(err, errors.Conflict) {
		t.Errorf("a second approval = %v, want CONFLICT", err)
	}

	none := linkPlan(t)
	if err := none.Approve(map[string]bool{}); err != nil || none.Status != agent.PlanDeclined {
		t.Errorf("approving nothing = %v, %s; want the plan declined", err, none.Status)
	}
}

func TestStepsRunInOrderAndAFailureSkipsWhatDependsOnIt(t *testing.T) {
	t.Parallel()

	plan := linkPlan(t)
	if err := plan.Approve(map[string]bool{"edge": true, "relink": true, "audit": true, "type": true}); err != nil {
		t.Fatalf("Approve: %v", err)
	}
	if got := plan.Runnable(); !slices.Equal(got, []int{0, 3}) {
		t.Fatalf("runnable at first = %v, want the edge and the type", got)
	}
	plan.Settle(0, agent.StepDone, json.RawMessage(`{"edge":"e-1"}`), "")
	plan.Settle(3, agent.StepDone, nil, "")
	if got := plan.Runnable(); !slices.Equal(got, []int{1}) {
		t.Fatalf("runnable after the edge = %v, want the relink", got)
	}
	plan.Settle(1, agent.StepWaiting, json.RawMessage(`{"runId":"run-1"}`), "")
	plan.Steps[1].RunID = "run-1"
	if at, waiting := plan.WaitingOn("run-1"); !waiting || at != 1 || len(plan.Runnable()) != 0 || plan.Finished() {
		t.Fatalf("a step waiting on its run = %d %v, runnable %v", at, waiting, plan.Runnable())
	}
	plan.Settle(1, agent.StepFailed, nil, "the run was cancelled")
	if plan.Steps[2].Status != agent.StepSkipped || !strings.Contains(plan.Steps[2].Error, "the run was cancelled") {
		t.Fatalf("the audit after a failed relink = %+v, want it skipped with the reason", plan.Steps[2])
	}
	if !plan.Finished() {
		t.Fatal("a plan with nothing left to run is not finished")
	}
	plan.Finish(planned)
	if plan.Status != agent.PlanDone {
		t.Errorf("a finished plan reads %s", plan.Status)
	}
}

func TestAnEditTouchesOnlyAFieldTheStepDeclares(t *testing.T) {
	t.Parallel()

	plan := linkPlan(t)
	plan.Steps[0].Args = json.RawMessage(`{"op":"add_edge","anchor":"serums","nested":{"goal":"sell"}}`)
	plan.Steps[0].Editable = []agent.EditField{{Name: "anchor", Path: "anchor"}, {Name: "goal", Path: "nested.goal"}}
	edited, err := plan.Steps[0].Edited(map[string]string{"anchor": "face serums", "goal": "teach"})
	if err != nil {
		t.Fatalf("Edited: %v", err)
	}
	var args map[string]any
	if err = json.Unmarshal(edited, &args); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if args["anchor"] != "face serums" || args["op"] != "add_edge" || args["nested"].(map[string]any)["goal"] != "teach" {
		t.Errorf("the edited args = %s", edited)
	}
	if _, err = plan.Steps[0].Edited(map[string]string{"op": "delete_edge"}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("editing an undeclared field = %v, want INVALID", err)
	}
}
