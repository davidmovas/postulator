package agent

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/application/tools"
	domainagent "github.com/davidmovas/postulator/internal/domain/agent"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func (s *Service) Confirm(ctx context.Context, req ConfirmRequest) (ConfirmResponse, error) {
	actionID := strings.TrimSpace(req.ActionID)
	if actionID == "" {
		return ConfirmResponse{}, invalid("a confirmation needs an action", "actionId")
	}

	action, err := s.deps.Actions.Get(ctx, actionID)
	if err != nil {
		return ConfirmResponse{}, err
	}
	if action.Status.Settled() {
		return ConfirmResponse{}, errors.New(errors.Conflict, "this action was already resolved").
			WithDetail("actionId", actionID).WithDetail("status", string(action.Status))
	}

	if !req.Approve {
		return s.reject(ctx, action)
	}
	return s.execute(ctx, action)
}

func (s *Service) reject(ctx context.Context, action domainagent.PendingAction) (ConfirmResponse, error) {
	taken, err := s.deps.Actions.Transition(ctx, action.ID, domainagent.ActionPending, domainagent.ActionRejected,
		nil, "", s.now())
	if err != nil {
		return ConfirmResponse{}, err
	}
	if !taken {
		return s.raced(ctx, action.ID)
	}

	settled, err := s.deps.Actions.Get(ctx, action.ID)
	if err != nil {
		return ConfirmResponse{}, err
	}
	s.emit(events.AgentConfirmResolved, events.AgentConfirmResolvedPayload{
		ConversationID: action.ConversationID, ConfirmationID: action.ID, Tool: action.Tool,
		Status: string(domainagent.ActionRejected),
	})
	return ConfirmResponse{Action: actionView(settled)}, nil
}

func (s *Service) execute(ctx context.Context, action domainagent.PendingAction) (ConfirmResponse, error) {
	claimed, err := s.deps.Actions.Transition(ctx, action.ID, domainagent.ActionPending, domainagent.ActionApproved,
		nil, "", s.now())
	if err != nil {
		return ConfirmResponse{}, err
	}
	if !claimed {
		return s.raced(ctx, action.ID)
	}

	conversation, err := s.deps.Conversations.Get(ctx, action.ConversationID)
	if err != nil {
		return ConfirmResponse{}, err
	}

	binding := tools.Binding{
		SiteID: siteOf(conversation), ConversationID: conversation.ID, Mode: domainagent.ModeAutonomous,
	}
	outcome, callErr := s.deps.Registry.Call(ctx, binding, action.Tool, action.Args)

	status, encoded, failure := settleCall(outcome, callErr)
	if _, err = s.deps.Actions.Transition(ctx, action.ID, domainagent.ActionApproved, status, encoded,
		failure, s.now()); err != nil {
		return ConfirmResponse{}, err
	}

	settled, err := s.deps.Actions.Get(ctx, action.ID)
	if err != nil {
		return ConfirmResponse{}, err
	}
	s.emit(events.AgentConfirmResolved, events.AgentConfirmResolvedPayload{
		ConversationID: action.ConversationID, ConfirmationID: action.ID, Tool: action.Tool,
		Status: string(status), Result: encoded, Error: failure,
	})

	if _, err = s.append(ctx, conversation.ID, domainagent.Message{
		Role: domainagent.RoleTool, Tool: action.Tool, CallID: action.ID, Payload: encoded, Text: failure,
	}); err != nil {
		return ConfirmResponse{}, err
	}
	if err = s.turn(ctx, conversation, resumeText(action.Tool, encoded, failure)); err != nil &&
		!errors.IsCode(err, errors.Conflict) {
		return ConfirmResponse{}, err
	}
	return ConfirmResponse{Action: actionView(settled)}, nil
}

func (s *Service) raced(ctx context.Context, actionID string) (ConfirmResponse, error) {
	latest, err := s.deps.Actions.Get(ctx, actionID)
	if err != nil {
		return ConfirmResponse{}, err
	}
	return ConfirmResponse{}, errors.New(errors.Conflict, "this action is already being resolved").
		WithDetail("actionId", actionID).WithDetail("status", string(latest.Status))
}

func settleCall(outcome any, callErr error) (status domainagent.ActionStatus, result json.RawMessage, failure string) {
	if callErr != nil {
		return domainagent.ActionFailed, nil, callErr.Error()
	}

	encoded, err := json.Marshal(outcome)
	if err != nil {
		return domainagent.ActionFailed, nil, "the tool answered something that cannot be encoded: " + err.Error()
	}
	return domainagent.ActionExecuted, encoded, ""
}

func resumeText(tool string, result json.RawMessage, failure string) string {
	if failure != "" {
		return "The confirmed tool " + tool + " failed: " + failure
	}
	fenced, err := json.Marshal(map[string]any{UntrustedMarker: true, UntrustedData: result})
	if err != nil {
		return "The confirmed tool " + tool + " ran and answered something that cannot be read back."
	}
	return "The confirmed tool " + tool + " ran. Its result is " + string(fenced)
}
