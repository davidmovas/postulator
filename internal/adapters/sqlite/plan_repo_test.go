package sqlite_test

import (
	"encoding/json"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/domain/agent"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

func TestAPlanIsKeptWithItsStepsAndMovesOnce(t *testing.T) {
	t.Parallel()

	f := newAgentFixture(t)
	conversation := f.conversationOf(t, agent.ModeConfirm)
	repo := sqlite.NewPlanRepo(f.store)
	plan, err := agent.NewPlan(agent.Plan{
		ID: id.New(), ConversationID: conversation.ID, SiteID: f.siteID, Title: "Link the serums",
		Steps: []agent.PlanStep{
			{ID: "s1", Tool: "graph_edit", Args: json.RawMessage(`{"op":"add_edge"}`), Reason: "the tree says so", Summary: "Add an edge"},
			{ID: "s2", Tool: "runs_control", Args: json.RawMessage(`{"op":"start"}`), Reason: "place the links", Summary: "Relink", DependsOn: []int{0},
				Editable: []agent.EditField{{Name: "goal", Path: "brief.goal"}}, EstimateUSD: 0.04},
		},
		CreatedAt: sqlitetest.Stamp, UpdatedAt: sqlitetest.Stamp,
	})
	if err != nil {
		t.Fatalf("NewPlan: %v", err)
	}
	if err = repo.Insert(t.Context(), plan); err != nil {
		t.Fatalf("Insert: %v", err)
	}
	stored, err := repo.Get(t.Context(), plan.ID)
	if err != nil || stored.Title != plan.Title || len(stored.Steps) != 2 || stored.Steps[1].DependsOn[0] != 0 ||
		stored.Steps[1].Editable[0].Path != "brief.goal" || stored.Steps[1].EstimateUSD != 0.04 || stored.Status != agent.PlanProposed {
		t.Fatalf("the stored plan = %+v, %v", stored, err)
	}

	if err = stored.Approve(map[string]bool{"s1": true, "s2": true}); err != nil {
		t.Fatalf("Approve: %v", err)
	}
	moved, err := repo.Transition(t.Context(), stored, agent.PlanProposed)
	if err != nil || !moved {
		t.Fatalf("the first approval moved %v, %v", moved, err)
	}
	if again, againErr := repo.Transition(t.Context(), stored, agent.PlanProposed); againErr != nil || again {
		t.Fatalf("a second approval of the same proposal moved %v, %v", again, againErr)
	}
	stored.Settle(0, agent.StepDone, json.RawMessage(`{"edge":"e-1"}`), "")
	if err = repo.Update(t.Context(), stored); err != nil {
		t.Fatalf("Update: %v", err)
	}
	running, err := repo.ListRunning(t.Context())
	if err != nil || len(running) != 1 || running[0].Steps[0].Status != agent.StepDone || string(running[0].Steps[0].Result) != `{"edge":"e-1"}` {
		t.Fatalf("the running plans = %+v, %v", running, err)
	}
	held, err := repo.ListByConversation(t.Context(), conversation.ID)
	if err != nil || len(held) != 1 {
		t.Fatalf("the conversation's plans = %+v, %v", held, err)
	}
	if _, err = repo.Get(t.Context(), id.New()); !errors.IsCode(err, errors.NotFound) {
		t.Fatalf("an absent plan = %v", err)
	}
}
