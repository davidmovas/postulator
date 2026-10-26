package agent

import (
	"encoding/json"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type PlanStatus string

const (
	PlanProposed PlanStatus = "proposed"
	PlanRunning  PlanStatus = "running"
	PlanDone     PlanStatus = "done"
	PlanDeclined PlanStatus = "declined"
)

type StepStatus string

const (
	StepProposed StepStatus = "proposed"
	StepDeclined StepStatus = "declined"
	StepPending  StepStatus = "pending"
	StepRunning  StepStatus = "running"
	StepWaiting  StepStatus = "waiting"
	StepDone     StepStatus = "done"
	StepFailed   StepStatus = "failed"
	StepSkipped  StepStatus = "skipped"
)

const (
	MaxPlanSteps = 50
	maxPlanTitle = 160
)

type EditField struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type PlanStep struct {
	ID             string          `json:"id"`
	Tool           string          `json:"tool"`
	Args           json.RawMessage `json:"args"`
	Reason         string          `json:"reason"`
	Summary        string          `json:"summary"`
	DependsOn      []int           `json:"dependsOn"`
	Editable       []EditField     `json:"editable"`
	EstimateUSD    float64         `json:"estimateUsd,omitempty"`
	EstimateTokens int             `json:"estimateTokens,omitempty"`
	Status         StepStatus      `json:"status"`
	Result         json.RawMessage `json:"result,omitempty"`
	Error          string          `json:"error,omitempty"`
	RunID          string          `json:"runId,omitempty"`
}

type Plan struct {
	ID             string
	ConversationID string
	SiteID         string
	Title          string
	Status         PlanStatus
	Steps          []PlanStep
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func NewPlan(p Plan) (Plan, error) {
	p.Title = strings.TrimSpace(p.Title)
	switch {
	case p.ID == "" || p.ConversationID == "":
		return Plan{}, invalid("a plan belongs to a conversation", "conversationId")
	case p.Title == "" || len([]rune(p.Title)) > maxPlanTitle:
		return Plan{}, invalid("a plan is titled in at most "+strconv.Itoa(maxPlanTitle)+" characters", "title")
	case len(p.Steps) == 0 || len(p.Steps) > MaxPlanSteps:
		return Plan{}, invalid("a plan holds between one and "+strconv.Itoa(MaxPlanSteps)+" steps", "steps")
	}
	if p.Status == "" {
		p.Status = PlanProposed
	}
	seen := make(map[string]struct{}, len(p.Steps))
	for i := range p.Steps {
		held := &p.Steps[i]
		if err := held.valid(i, seen); err != nil {
			return Plan{}, err
		}
		if held.Status == "" {
			held.Status = StepProposed
		}
		if held.DependsOn == nil {
			held.DependsOn = []int{}
		}
		if held.Editable == nil {
			held.Editable = []EditField{}
		}
	}
	return p, nil
}

func (s *PlanStep) valid(at int, seen map[string]struct{}) error {
	where := "step " + strconv.Itoa(at+1)
	if _, twice := seen[s.ID]; twice || s.ID == "" {
		return invalid(where+" needs an id of its own", "steps.id")
	}
	seen[s.ID] = struct{}{}
	s.Reason = strings.TrimSpace(s.Reason)
	switch {
	case strings.TrimSpace(s.Tool) == "":
		return invalid(where+" names the tool it calls", "steps.tool")
	case s.Reason == "":
		return invalid(where+" says why it is in the plan", "steps.reason")
	}
	var object map[string]any
	if json.Unmarshal(s.Args, &object) != nil {
		return invalid(where+"'s arguments are one JSON object", "steps.args")
	}
	for _, dependency := range s.DependsOn {
		if dependency < 0 || dependency >= at {
			return invalid(where+" can depend only on a step before it", "steps.dependsOn")
		}
	}
	return nil
}

func (p *Plan) Approve(chosen map[string]bool) error {
	if p.Status != PlanProposed {
		return errors.New(errors.Conflict, "the plan was settled already").WithDetail("planId", p.ID).WithDetail("status", string(p.Status))
	}
	pending := 0
	for i := range p.Steps {
		held := &p.Steps[i]
		held.Status = StepDeclined
		if !chosen[held.ID] {
			continue
		}
		if slices.ContainsFunc(held.DependsOn, func(at int) bool { return p.Steps[at].Status == StepDeclined }) {
			continue
		}
		held.Status = StepPending
		pending++
	}
	p.Status = PlanRunning
	if pending == 0 {
		p.Status = PlanDeclined
	}
	return nil
}

func (p Plan) Runnable() []int {
	out := make([]int, 0)
	for i := range p.Steps {
		if p.Steps[i].Status != StepPending {
			continue
		}
		if slices.ContainsFunc(p.Steps[i].DependsOn, func(at int) bool { return p.Steps[at].Status != StepDone }) {
			continue
		}
		out = append(out, i)
	}
	return out
}

func (p Plan) WaitingOn(runID string) (int, bool) {
	at := slices.IndexFunc(p.Steps, func(held PlanStep) bool { return held.Status == StepWaiting && held.RunID == runID })
	return at, at >= 0
}

func (p *Plan) Settle(at int, status StepStatus, result json.RawMessage, failure string) {
	held := &p.Steps[at]
	held.Status, held.Error = status, failure
	if result != nil {
		held.Result = result
	}
	if status == StepFailed || status == StepSkipped {
		p.skipAfter(at, failure)
	}
}

func (p *Plan) skipAfter(at int, failure string) {
	why := "step " + strconv.Itoa(at+1) + " it depends on did not land: " + failure
	for i := at + 1; i < len(p.Steps); i++ {
		held := &p.Steps[i]
		if held.Status != StepPending || !slices.Contains(held.DependsOn, at) {
			continue
		}
		held.Status, held.Error = StepSkipped, why
		p.skipAfter(i, failure)
	}
}

func (p Plan) Finished() bool {
	return !slices.ContainsFunc(p.Steps, func(held PlanStep) bool {
		return held.Status == StepPending || held.Status == StepRunning || held.Status == StepWaiting
	})
}

func (p *Plan) Finish(at time.Time) {
	if p.Finished() && p.Status == PlanRunning {
		p.Status = PlanDone
	}
	p.UpdatedAt = at
}

func (s PlanStep) Edited(edits map[string]string) (json.RawMessage, error) {
	if len(edits) == 0 {
		return s.Args, nil
	}
	var args map[string]any
	if err := json.Unmarshal(s.Args, &args); err != nil {
		return nil, invalid("the step's arguments are not readable", "steps.args")
	}
	for name, value := range edits {
		at := slices.IndexFunc(s.Editable, func(field EditField) bool { return field.Name == name })
		if at < 0 {
			return nil, invalid("the step offers no edit of "+name, "steps.edits").WithDetail("stepId", s.ID).WithDetail("edit", name)
		}
		set(args, strings.Split(s.Editable[at].Path, "."), value)
	}
	encoded, err := json.Marshal(args)
	if err != nil {
		return nil, errors.Wrap(err, errors.Internal, "encode the edited step")
	}
	return encoded, nil
}

func set(into map[string]any, path []string, value string) {
	if len(path) == 1 {
		into[path[0]] = value
		return
	}
	next, held := into[path[0]].(map[string]any)
	if !held {
		next = map[string]any{}
		into[path[0]] = next
	}
	set(next, path[1:], value)
}
