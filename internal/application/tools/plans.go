package tools

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/domain/agent"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

const (
	plansName    = "plans"
	opPropose    = "propose"
	PlanProposed = "planProposed"
)

type StepCheck struct {
	Summary  string
	Editable []agent.EditField
	Price    Price
}

func (r *Registry) Validate(ctx context.Context, b Binding, name string, args json.RawMessage) (StepCheck, error) {
	build, known := r.byName[name]
	switch {
	case !known:
		return StepCheck{}, errors.New(errors.NotFound, "no tool is registered under this name").WithDetail("tool", name)
	case name == plansName:
		return StepCheck{}, errors.New(errors.Invalid, "a plan cannot hold a plan").WithDetail("tool", name)
	}
	tool := build(r.deps)
	if !tool.risk(args).NeedsConfirmation() {
		return StepCheck{}, errors.New(errors.Invalid, "the tool "+name+" only reads: a plan holds the writes, and a read is called directly").
			WithDetail("tool", name)
	}
	if tool.Authorize != nil {
		if err := tool.Authorize(ctx, b); err != nil {
			return StepCheck{}, err
		}
	}
	if tool.Check != nil {
		if err := tool.Check(args); err != nil {
			return StepCheck{}, err
		}
	}
	check := StepCheck{Editable: tool.Edits}
	if tool.Confirm != nil {
		said, err := tool.Confirm(ctx, b, args)
		if err != nil {
			return StepCheck{}, err
		}
		check.Summary = said
	}
	if check.Summary == "" {
		said, err := sayOf(ctx, tool, b, args)
		if err != nil {
			return StepCheck{}, err
		}
		check.Summary = said
	}
	if check.Summary == "" {
		check.Summary = Summary(tool.Def, args)
	}
	if tool.Price != nil {
		price, err := tool.Price(ctx, b, args)
		if err != nil {
			return StepCheck{}, err
		}
		check.Price = price
	}
	return check, nil
}

type planStepArgs struct {
	Tool      string `json:"tool" description:"The write tool the step calls"`
	Args      string `json:"args" description:"Its arguments as one JSON object, as that tool takes them"`
	Reason    string `json:"reason" description:"Why the step is in the plan, one sentence for the person"`
	DependsOn []int  `json:"dependsOn,omitempty" description:"Earlier steps, numbered from 1, that must land first"`
}

type plansArgs struct {
	Op     string         `json:"op,omitempty" enum:"propose,get,list" description:"Default list; propose puts a plan in front of the person"`
	Title  string         `json:"title,omitempty" description:"propose: what the plan does, in a few words"`
	Steps  []planStepArgs `json:"steps,omitempty" description:"propose: the writes in the order they run"`
	PlanID string         `json:"planId,omitempty" description:"get: a plan's id"`
}

type PlanStepView struct {
	Number         int              `json:"number"`
	ID             string           `json:"id"`
	Tool           string           `json:"tool"`
	Summary        string           `json:"summary"`
	Reason         string           `json:"reason"`
	DependsOn      []int            `json:"dependsOn"`
	Status         agent.StepStatus `json:"status"`
	EstimateUSD    float64          `json:"estimateUsd,omitempty"`
	EstimateTokens int              `json:"estimateTokens,omitempty"`
	Error          string           `json:"error,omitempty"`
	RunID          string           `json:"runId,omitempty"`
	Edits          []EditView       `json:"edits"`
}

type EditView struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type PlanView struct {
	ID     string           `json:"id"`
	Title  string           `json:"title"`
	Status agent.PlanStatus `json:"status"`
	Steps  []PlanStepView   `json:"steps"`
}

type PlansResponse struct {
	Status string     `json:"status,omitempty"`
	PlanID string     `json:"planId,omitempty"`
	Plan   *PlanView  `json:"plan,omitempty"`
	Plans  []PlanView `json:"plans,omitempty"`
}

func ViewOfPlan(plan agent.Plan) PlanView {
	view := PlanView{ID: plan.ID, Title: plan.Title, Status: plan.Status, Steps: make([]PlanStepView, 0, len(plan.Steps))}
	for i := range plan.Steps {
		held := &plan.Steps[i]
		dependsOn := make([]int, 0, len(held.DependsOn))
		for _, at := range held.DependsOn {
			dependsOn = append(dependsOn, at+1)
		}
		view.Steps = append(view.Steps, PlanStepView{
			Number: i + 1, ID: held.ID, Tool: held.Tool, Summary: held.Summary, Reason: held.Reason, DependsOn: dependsOn,
			Status: held.Status, EstimateUSD: held.EstimateUSD, EstimateTokens: held.EstimateTokens, Error: held.Error, RunID: held.RunID,
			Edits: editsOf(held),
		})
	}
	return view
}

func editsOf(step *agent.PlanStep) []EditView {
	var args map[string]any
	if json.Unmarshal(step.Args, &args) != nil {
		args = map[string]any{}
	}
	out := make([]EditView, 0, len(step.Editable))
	for _, field := range step.Editable {
		out = append(out, EditView{Name: field.Name, Value: valueAt(args, strings.Split(field.Path, "."))})
	}
	return out
}

func valueAt(args map[string]any, path []string) string {
	held, found := args[path[0]]
	if !found {
		return ""
	}
	if len(path) == 1 {
		if text, isText := held.(string); isText {
			return text
		}
		return ""
	}
	nested, object := held.(map[string]any)
	if !object {
		return ""
	}
	return valueAt(nested, path[1:])
}

func plansTool(deps Deps) Tool {
	return NewTool(Def{
		Name: plansName,
		Description: "Put a plan of writes in front of the person as one card, each step with its reason; nothing runs until " +
			"they approve it, whole or in part. get and list read the plans of this conversation.",
		Risk: RiskRead,
	}, func(ctx context.Context, b Binding, in plansArgs) (PlansResponse, error) {
		if b.ConversationID == "" {
			return PlansResponse{}, errors.New(errors.Invalid, "a plan belongs to a conversation, and this call names none")
		}
		switch in.Op {
		case opPropose:
			return proposePlan(ctx, deps, b, in)
		case opGet:
			plan, err := deps.Plans.Get(ctx, strings.TrimSpace(in.PlanID))
			if err != nil {
				return PlansResponse{}, err
			}
			if plan.ConversationID != b.ConversationID {
				return PlansResponse{}, errors.New(errors.NotFound, "plan not found").WithDetail("planId", in.PlanID)
			}
			view := ViewOfPlan(plan)
			return PlansResponse{Plan: &view}, nil
		default:
			held, err := deps.Plans.ListByConversation(ctx, b.ConversationID)
			if err != nil {
				return PlansResponse{}, err
			}
			views := make([]PlanView, 0, len(held))
			for i := range held {
				views = append(views, ViewOfPlan(held[i]))
			}
			return PlansResponse{Plans: views}, nil
		}
	})
}

func proposePlan(ctx context.Context, deps Deps, b Binding, in plansArgs) (PlansResponse, error) {
	now := deps.Clock.Now().UTC().Truncate(1e9)
	steps := make([]agent.PlanStep, 0, len(in.Steps))
	for i, raw := range in.Steps {
		where := "step " + strconv.Itoa(i+1)
		args := json.RawMessage(strings.TrimSpace(raw.Args))
		if !json.Valid(args) {
			return PlansResponse{}, errors.New(errors.Invalid, where+"'s args are not one JSON object").WithDetail("field", "steps.args")
		}
		check, err := deps.steps.Validate(ctx, b, strings.TrimSpace(raw.Tool), args)
		if err != nil {
			return PlansResponse{}, errors.Wrap(err, errors.CodeOf(err), where+" ("+raw.Tool+") would be refused: "+err.Error())
		}
		dependsOn := make([]int, 0, len(raw.DependsOn))
		for _, number := range raw.DependsOn {
			dependsOn = append(dependsOn, number-1)
		}
		steps = append(steps, agent.PlanStep{
			ID: id.New(), Tool: strings.TrimSpace(raw.Tool), Args: args, Reason: raw.Reason, Summary: check.Summary,
			DependsOn: dependsOn, Editable: check.Editable, EstimateUSD: check.Price.USD, EstimateTokens: check.Price.Tokens,
		})
	}
	plan, err := agent.NewPlan(agent.Plan{
		ID: id.New(), ConversationID: b.ConversationID, SiteID: b.SiteID, Title: in.Title, Steps: steps, CreatedAt: now, UpdatedAt: now,
	})
	if err != nil {
		return PlansResponse{}, err
	}
	if insertErr := deps.Plans.Insert(ctx, plan); insertErr != nil {
		return PlansResponse{}, insertErr
	}
	if publishErr := deps.Publisher.Publish(events.AgentPlanProposed, events.AgentPlanProposedPayload{
		ConversationID: b.ConversationID, PlanID: plan.ID, Title: plan.Title, Steps: len(plan.Steps),
	}); publishErr != nil {
		return PlansResponse{}, publishErr
	}
	view := ViewOfPlan(plan)
	return PlansResponse{Status: PlanProposed, PlanID: plan.ID, Plan: &view}, nil
}
