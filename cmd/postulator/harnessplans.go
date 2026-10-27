//go:build uiharness

package main

import (
	"context"
	"encoding/json"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/app"
	"github.com/davidmovas/postulator/internal/application/graph"
	"github.com/davidmovas/postulator/internal/application/tools"
	domainagent "github.com/davidmovas/postulator/internal/domain/agent"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

const (
	misplacedEntity = "Tamping"
	misplacedUnder  = "Accessories"

	plannedAsk     = "Link the grinders section to the grind size guide, and write the hub again first."
	plannedAnswer  = "The plan is in front of you: the edge first, then the hub with a brief, then the guide's links."
	finishedAsk    = "Tidy the titles of the espresso machine pages."
	finishedAnswer = "Two of the four steps landed; the third failed on its run, so the fourth was skipped."
)

func misplaceAnEntity(ctx context.Context, core *app.Core, entities map[string]string) error {
	_, err := core.Graph.MoveEntity(ctx, graph.MoveEntityRequest{
		EntityID: entities[misplacedEntity], NewParentID: entities[misplacedUnder],
	})
	return err
}

func seedPlans(ctx context.Context, core *app.Core, siteID string, byPath, entities map[string]string) error {
	conversation, err := onlyConversation(ctx, core, siteID)
	if err != nil {
		return err
	}
	finished, err := finishedRunItems(ctx, core, siteID)
	if err != nil {
		return err
	}
	now := time.Now().UTC()

	done, err := finishedPlan(conversation, siteID, byPath, finished.runID, now)
	if err != nil {
		return err
	}
	if doneErr := proposeInConversation(ctx, core, done, finishedAsk, finishedAnswer); doneErr != nil {
		return doneErr
	}
	proposed, err := proposedPlan(conversation, siteID, byPath, entities, now.Add(time.Second))
	if err != nil {
		return err
	}
	return proposeInConversation(ctx, core, proposed, plannedAsk, plannedAnswer)
}

func argsOf(value map[string]any) json.RawMessage {
	encoded, err := json.Marshal(value)
	if err != nil {
		return json.RawMessage("{}")
	}
	return encoded
}

func proposedPlan(conversation, siteID string, byPath, entities map[string]string, at time.Time) (domainagent.Plan, error) {
	hub, guide := byPath["/grinders/"], byPath["/grinders/grind-size/"]
	return domainagent.NewPlan(domainagent.Plan{
		ID: id.New(), ConversationID: conversation, SiteID: siteID, CreatedAt: at, UpdatedAt: at,
		Title: "Link the grinders hub and the grind size guide",
		Steps: []domainagent.PlanStep{
			{
				ID: id.New(), Tool: "graph",
				Args: argsOf(map[string]any{
					"op": "add_edge", "fromEntityId": entities["Grinders"], "toEntityId": entities["Grind size for espresso"],
					"kind": "related",
				}),
				Summary: "Add a related edge from Grinders to Grind size for espresso",
				Reason:  "The hub must link to the guide, and no edge of the entity graph joins the two yet.",
			},
			{
				ID: id.New(), Tool: "runs",
				Args: argsOf(map[string]any{
					"op": "start", "pageIds": []string{hub},
					"brief": map[string]any{
						"goal": "Help a first buyer choose a grinder before a machine",
						"tone": "Plain and practical, no brand talk",
					},
					"directives": []map[string]any{{"pageId": hub, "mustLink": []map[string]any{{"pageId": guide}}}},
				}),
				Summary:   "Write /grinders/ again with a brief, linking it to /grinders/grind-size/",
				Reason:    "The hub reads like a price list; a brief turns it into a buying guide that sends readers on.",
				DependsOn: []int{0},
				Editable: []domainagent.EditField{
					{Name: "goal", Path: "brief.goal"}, {Name: "tone", Path: "brief.tone"},
				},
				EstimateUSD: 0.42, EstimateTokens: 18400,
			},
			{
				ID: id.New(), Tool: "runs",
				Args:      argsOf(map[string]any{"op": "start", "kind": "relink", "pageIds": []string{guide}}),
				Summary:   "Place the owed links on /grinders/grind-size/",
				Reason:    "Once the hub is written, the guide owes it a link back.",
				DependsOn: []int{1},
			},
		},
	})
}

func finishedPlan(conversation, siteID string, byPath map[string]string, runID string, at time.Time) (domainagent.Plan, error) {
	plan, err := domainagent.NewPlan(domainagent.Plan{
		ID: id.New(), ConversationID: conversation, SiteID: siteID, CreatedAt: at, UpdatedAt: at,
		Title: "Tidy the espresso machine titles",
		Steps: []domainagent.PlanStep{
			{
				ID: id.New(), Tool: "pages", Args: argsOf(map[string]any{"op": "set", "pageId": byPath["/espresso-machines/"]}),
				Summary: "Retitle /espresso-machines/ so it leads with the types",
				Reason:  "The hub's title buries the word readers search for.",
			},
			{
				ID: id.New(), Tool: "runs", Args: argsOf(map[string]any{"op": "start", "pageIds": []string{byPath[completedRun[0]]}}),
				Summary: "Write the meta of the pre-infusion guide again",
				Reason:  "Its description repeats the title word for word.",
			},
			{
				ID: id.New(), Tool: "runs", Args: argsOf(map[string]any{"op": "start", "pageIds": []string{byPath[failingPath]}}),
				Summary: "Write the single dosing guide's title again",
				Reason:  "The planned title is longer than a search result shows.",
			},
			{
				ID: id.New(), Tool: "runs", Args: argsOf(map[string]any{"op": "start", "kind": "relink", "pageIds": []string{byPath[failingPath]}}),
				Summary: "Place the owed links on the single dosing guide", Reason: "Its new title changes the anchor others use.",
				DependsOn: []int{2},
			},
			{
				ID: id.New(), Tool: "pages", Args: argsOf(map[string]any{"op": "set", "pageId": byPath["/espresso-machines/dual-boiler/"]}),
				Summary: "Retitle /espresso-machines/dual-boiler/", Reason: "The title says boilers twice.",
			},
		},
	})
	if err != nil {
		return domainagent.Plan{}, err
	}
	plan.Status = domainagent.PlanDone
	settled := []struct {
		status domainagent.StepStatus
		runID  string
		why    string
	}{
		{domainagent.StepDone, "", ""},
		{domainagent.StepDone, runID, ""},
		{domainagent.StepFailed, "", "its run did not land: 1 of its pages failed"},
		{domainagent.StepSkipped, "", "step 3 it depends on did not land: 1 of its pages failed"},
		{domainagent.StepDeclined, "", ""},
	}
	for i := range plan.Steps {
		plan.Steps[i].Status, plan.Steps[i].RunID, plan.Steps[i].Error = settled[i].status, settled[i].runID, settled[i].why
	}
	return plan, nil
}

func proposeInConversation(ctx context.Context, core *app.Core, plan domainagent.Plan, ask, answer string) error {
	if err := sqlite.NewPlanRepo(core.Store).Insert(ctx, plan); err != nil {
		return err
	}
	view := tools.ViewOfPlan(plan)
	proposal := tools.PlansResponse{Status: tools.PlanProposed, PlanID: plan.ID, Plan: &view}
	payload, err := json.Marshal(proposal)
	if err != nil {
		return errors.Wrap(err, errors.Internal, "encode the seeded plan")
	}
	callID := id.New()
	call, err := domainagent.NewToolCall(domainagent.ToolCall{
		ID: id.New(), ConversationID: plan.ConversationID, CallID: callID, Tool: "plans",
		Args: argsOf(map[string]any{"op": "propose", "title": plan.Title}), Status: domainagent.CallOK,
		DurationMS: 40, CreatedAt: plan.CreatedAt,
	})
	if err != nil {
		return err
	}
	if callErr := sqlite.NewToolCallRepo(core.Store).Insert(ctx, call); callErr != nil {
		return callErr
	}
	messages := sqlite.NewMessageRepo(core.Store)
	exchange := []domainagent.Message{
		{Role: domainagent.RoleUser, Text: ask},
		{Role: domainagent.RoleTool, Tool: "plans", CallID: callID, Payload: payload},
		{Role: domainagent.RoleAssistant, Text: answer},
	}
	for i := range exchange {
		held := &exchange[i]
		held.ID, held.ConversationID, held.Seq, held.CreatedAt = id.New(), plan.ConversationID, 1, plan.CreatedAt
		message, validErr := domainagent.NewMessage(*held)
		if validErr != nil {
			return validErr
		}
		if _, appendErr := messages.Append(ctx, message); appendErr != nil {
			return appendErr
		}
	}
	return nil
}
