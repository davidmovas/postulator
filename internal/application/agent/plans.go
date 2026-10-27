package agent

import (
	"context"
	"encoding/json"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/application/tools"
	domainagent "github.com/davidmovas/postulator/internal/domain/agent"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

const stoppedMidStep = "the app stopped while this step ran, so nothing says whether it landed; check what it would have " +
	"changed and propose it again"

type planStore interface {
	Get(ctx context.Context, id string) (domainagent.Plan, error)
	Update(ctx context.Context, p domainagent.Plan) error
	Transition(ctx context.Context, p domainagent.Plan, from domainagent.PlanStatus) (bool, error)
	ListRunning(ctx context.Context) ([]domainagent.Plan, error)
	ListByConversation(ctx context.Context, conversationID string) ([]domainagent.Plan, error)
}

type runReader interface {
	Get(ctx context.Context, id string) (run.Run, error)
}

type StepChoice struct {
	StepID  string            `json:"stepId"`
	Approve bool              `json:"approve"`
	Edits   map[string]string `json:"edits,omitempty"`
}

type ConfirmPlanRequest struct {
	PlanID string       `json:"planId"`
	Steps  []StepChoice `json:"steps"`
}

type ConfirmPlanResponse struct {
	Plan tools.PlanView `json:"plan"`
}

type ListPlansRequest struct {
	ConversationID string `json:"conversationId"`
}

type ListPlansResponse struct {
	Plans []tools.PlanView `json:"plans"`
}

type RunOutcome struct {
	RunID   string
	Failure string
}

type planLocks struct {
	mu   sync.Mutex
	held map[string]*sync.Mutex
}

func (l *planLocks) of(planID string) *sync.Mutex {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.held == nil {
		l.held = map[string]*sync.Mutex{}
	}
	if _, found := l.held[planID]; !found {
		l.held[planID] = &sync.Mutex{}
	}
	return l.held[planID]
}

func (s *Service) ListPlans(ctx context.Context, req ListPlansRequest) (ListPlansResponse, error) {
	held, err := s.deps.Plans.ListByConversation(ctx, strings.TrimSpace(req.ConversationID))
	if err != nil {
		return ListPlansResponse{}, err
	}
	out := ListPlansResponse{Plans: make([]tools.PlanView, 0, len(held))}
	for i := range held {
		out.Plans = append(out.Plans, tools.ViewOfPlan(held[i]))
	}
	return out, nil
}

func (s *Service) ConfirmPlan(ctx context.Context, req ConfirmPlanRequest) (ConfirmPlanResponse, error) {
	lock := s.locks.of(req.PlanID)
	lock.Lock()
	defer lock.Unlock()

	plan, err := s.deps.Plans.Get(ctx, strings.TrimSpace(req.PlanID))
	if err != nil {
		return ConfirmPlanResponse{}, err
	}
	conversation, err := s.deps.Conversations.Get(ctx, plan.ConversationID)
	if err != nil {
		return ConfirmPlanResponse{}, err
	}
	binding := tools.Binding{SiteID: siteOf(conversation), ConversationID: conversation.ID, Mode: conversation.Mode}
	chosen := make(map[string]bool, len(req.Steps))
	for _, choice := range req.Steps {
		at := slices.IndexFunc(plan.Steps, func(held domainagent.PlanStep) bool { return held.ID == choice.StepID })
		if at < 0 {
			return ConfirmPlanResponse{}, invalid("the plan holds no step "+choice.StepID, "steps.stepId")
		}
		chosen[choice.StepID] = choice.Approve
		if !choice.Approve || len(choice.Edits) == 0 {
			continue
		}
		edited, editErr := plan.Steps[at].Edited(choice.Edits)
		if editErr != nil {
			return ConfirmPlanResponse{}, editErr
		}
		assuming := s.deps.Registry.Assuming(binding, plan.Steps[:at], plan.Steps[at].DependsOn)
		check, checkErr := s.deps.Registry.Validate(ctx, assuming, plan.Steps[at].Tool, edited)
		if checkErr != nil {
			return ConfirmPlanResponse{}, checkErr
		}
		plan.Steps[at].Args, plan.Steps[at].Summary = edited, check.Summary
	}
	if approveErr := plan.Approve(chosen); approveErr != nil {
		return ConfirmPlanResponse{}, approveErr
	}
	plan.UpdatedAt = s.now()
	moved, err := s.deps.Plans.Transition(ctx, plan, domainagent.PlanProposed)
	if err != nil {
		return ConfirmPlanResponse{}, err
	}
	if !moved {
		return ConfirmPlanResponse{}, errors.New(errors.Conflict, "the plan was settled already").WithDetail("planId", plan.ID)
	}
	s.planMoved(plan, "")
	if plan.Status == domainagent.PlanDeclined {
		s.tell(ctx, conversation, "The person declined the whole plan “"+plan.Title+"”; none of its steps ran.")
		return ConfirmPlanResponse{Plan: tools.ViewOfPlan(plan)}, nil
	}
	s.advance(ctx, conversation, &plan)
	return ConfirmPlanResponse{Plan: tools.ViewOfPlan(plan)}, nil
}

func (s *Service) RunEnded(ctx context.Context, outcome RunOutcome) error {
	running, err := s.deps.Plans.ListRunning(ctx)
	if err != nil {
		return err
	}
	for i := range running {
		if _, waiting := running[i].WaitingOn(outcome.RunID); !waiting {
			continue
		}
		if wakeErr := s.wake(ctx, running[i].ID, outcome); wakeErr != nil {
			return wakeErr
		}
	}
	return nil
}

func (s *Service) wake(ctx context.Context, planID string, outcome RunOutcome) error {
	lock := s.locks.of(planID)
	lock.Lock()
	defer lock.Unlock()

	plan, err := s.deps.Plans.Get(ctx, planID)
	if err != nil {
		return err
	}
	at, waiting := plan.WaitingOn(outcome.RunID)
	if !waiting {
		return nil
	}
	settleRun(&plan, at, outcome)
	conversation, err := s.deps.Conversations.Get(ctx, plan.ConversationID)
	if err != nil {
		return err
	}
	s.advance(ctx, conversation, &plan)
	return nil
}

func settleRun(plan *domainagent.Plan, at int, outcome RunOutcome) {
	if outcome.Failure != "" {
		plan.Settle(at, domainagent.StepFailed, nil, "its run did not land: "+outcome.Failure)
		return
	}
	plan.Settle(at, domainagent.StepDone, nil, "")
}

func (s *Service) ResumePlans(ctx context.Context) error {
	running, err := s.deps.Plans.ListRunning(ctx)
	if err != nil {
		return err
	}
	for i := range running {
		if resumeErr := s.resumePlan(ctx, running[i].ID); resumeErr != nil {
			return resumeErr
		}
	}
	return nil
}

func (s *Service) resumePlan(ctx context.Context, planID string) error {
	lock := s.locks.of(planID)
	lock.Lock()
	defer lock.Unlock()

	plan, err := s.deps.Plans.Get(ctx, planID)
	if err != nil {
		return err
	}
	for at := range plan.Steps {
		switch plan.Steps[at].Status {
		case domainagent.StepRunning:
			plan.Settle(at, domainagent.StepFailed, nil, stoppedMidStep)
		case domainagent.StepWaiting:
			if outcome, ended := s.ended(ctx, plan.Steps[at].RunID); ended {
				settleRun(&plan, at, outcome)
			}
		default:
		}
	}
	conversation, err := s.deps.Conversations.Get(ctx, plan.ConversationID)
	if err != nil {
		return err
	}
	s.advance(ctx, conversation, &plan)
	return nil
}

func (s *Service) ended(ctx context.Context, runID string) (RunOutcome, bool) {
	if s.deps.Runs == nil || runID == "" {
		return RunOutcome{}, false
	}
	held, err := s.deps.Runs.Get(ctx, runID)
	if err != nil {
		return RunOutcome{RunID: runID, Failure: err.Error()}, errors.IsCode(err, errors.NotFound)
	}
	if !held.Status.Terminal() {
		return RunOutcome{}, false
	}
	outcome := RunOutcome{RunID: runID}
	switch {
	case held.Status != run.StatusCompleted:
		outcome.Failure = "the run ended " + string(held.Status)
	case held.Stats.Failed > 0:
		outcome.Failure = strconv.Itoa(held.Stats.Failed) + " of its pages failed"
	}
	return outcome, true
}

func (s *Service) advance(ctx context.Context, conversation domainagent.Conversation, plan *domainagent.Plan) {
	binding := tools.Binding{SiteID: siteOf(conversation), ConversationID: conversation.ID, Mode: conversation.Mode, Approved: true}
	for {
		runnable := plan.Runnable()
		if len(runnable) == 0 {
			break
		}
		for _, at := range runnable {
			s.runStep(ctx, binding, plan, at)
		}
	}
	plan.Finish(s.now())
	if err := s.deps.Plans.Update(ctx, *plan); err != nil {
		s.drop(err)
	}
	s.planMoved(*plan, "")
	s.tell(ctx, conversation, planReport(*plan))
}

func (s *Service) runStep(ctx context.Context, b tools.Binding, plan *domainagent.Plan, at int) {
	held := &plan.Steps[at]
	held.Status = domainagent.StepRunning
	plan.UpdatedAt = s.now()
	if err := s.deps.Plans.Update(ctx, *plan); err != nil {
		s.drop(err)
	}
	s.planMoved(*plan, held.ID)

	started := time.Now()
	outcome, callErr := s.deps.Registry.Call(ctx, s.deps.Registry.Assuming(b, plan.Steps[:at], held.DependsOn), held.Tool, held.Args)
	_, encoded, failure := settleCall(outcome, callErr)
	encoded = capped(encoded, s.maxToolResult())
	s.auditStep(ctx, plan.ConversationID, *held, failure, errors.IsCode(callErr, errors.Unauthorized), time.Since(started).Milliseconds())

	switch runID := runIDOf(encoded); {
	case failure != "":
		plan.Settle(at, domainagent.StepFailed, nil, failure)
	case runID != "" && dependedOn(*plan, at):
		held.RunID = runID
		plan.Settle(at, domainagent.StepWaiting, encoded, "")
		if ended, done := s.ended(ctx, runID); done {
			settleRun(plan, at, ended)
		}
	default:
		held.RunID = runID
		plan.Settle(at, domainagent.StepDone, encoded, "")
	}
	plan.UpdatedAt = s.now()
	if err := s.deps.Plans.Update(ctx, *plan); err != nil {
		s.drop(err)
	}
	s.planMoved(*plan, held.ID)
}

func (s *Service) auditStep(ctx context.Context, conversationID string, step domainagent.PlanStep, failure string, denied bool, elapsed int64) {
	call, err := domainagent.NewToolCall(domainagent.ToolCall{
		ID: id.New(), ConversationID: conversationID, CallID: step.ID, Tool: step.Tool, Args: tools.Redact(step.Args),
		Status: callStatus(failure, denied), DurationMS: elapsed, Error: failure, CreatedAt: s.now(),
	})
	if err == nil {
		err = s.deps.Calls.Insert(ctx, call)
	}
	s.drop(err)
}

func (s *Service) planMoved(plan domainagent.Plan, stepID string) {
	stepStatus := ""
	if at := slices.IndexFunc(plan.Steps, func(held domainagent.PlanStep) bool { return held.ID == stepID }); at >= 0 {
		stepStatus = string(plan.Steps[at].Status)
	}
	s.emit(events.AgentPlanUpdated, events.AgentPlanUpdatedPayload{
		ConversationID: plan.ConversationID, PlanID: plan.ID, Status: string(plan.Status), StepID: stepID, StepStatus: stepStatus,
	})
}

func (s *Service) tell(ctx context.Context, conversation domainagent.Conversation, text string) {
	if text == "" {
		return
	}
	s.drop(s.resume(ctx, conversation, text))
}

func dependedOn(plan domainagent.Plan, at int) bool {
	return slices.ContainsFunc(plan.Steps[at+1:], func(held domainagent.PlanStep) bool {
		return held.Status == domainagent.StepPending && slices.Contains(held.DependsOn, at)
	})
}

func runIDOf(result json.RawMessage) string {
	var held struct {
		RunID string `json:"runId"`
		Edit  *struct {
			RunID string `json:"runId"`
		} `json:"edit"`
	}
	if json.Unmarshal(result, &held) != nil {
		return ""
	}
	if held.RunID == "" && held.Edit != nil {
		return held.Edit.RunID
	}
	return held.RunID
}

func planReport(plan domainagent.Plan) string {
	var report strings.Builder
	report.WriteString("The plan “" + plan.Title + "” ")
	if plan.Status == domainagent.PlanDone {
		report.WriteString("has run:")
	} else {
		report.WriteString("is under way:")
	}
	for i := range plan.Steps {
		held := &plan.Steps[i]
		report.WriteString("\n- step " + strconv.Itoa(i+1) + " (" + held.Summary + "): " + string(held.Status))
		if held.RunID != "" {
			report.WriteString(", run " + held.RunID)
		}
		if held.Error != "" {
			report.WriteString(": " + held.Error)
		}
	}
	return report.String()
}
