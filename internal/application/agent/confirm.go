package agent

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"time"

	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/application/tools"
	domainagent "github.com/davidmovas/postulator/internal/domain/agent"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
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

	started := time.Now()
	outcome, callErr := s.replay(ctx, conversation, action)
	elapsed := time.Since(started).Milliseconds()

	status, encoded, failure := settleCall(outcome, callErr)
	encoded = capped(encoded, s.maxToolResult())
	if _, err = s.deps.Actions.Transition(ctx, action.ID, domainagent.ActionApproved, status, encoded,
		failure, s.now()); err != nil {
		return ConfirmResponse{}, err
	}
	if auditErr := s.audit(ctx, action, failure, errors.IsCode(callErr, errors.Unauthorized), elapsed); auditErr != nil {
		return ConfirmResponse{}, auditErr
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
	s.deps.Turns.Note(s.resume(ctx, conversation, resumeText(action.Tool, encoded, failure)))
	return ConfirmResponse{Action: actionView(settled)}, nil
}

func (s *Service) resume(ctx context.Context, conversation domainagent.Conversation, text string) error {
	for range 2 {
		if s.deps.Turns.Queue(conversation.ID, text) {
			return nil
		}
		_, err := s.turn(ctx, conversation, text)
		if err == nil || !errors.IsCode(err, errors.Conflict) {
			return err
		}
	}
	return errors.New(errors.Conflict, "the conversation could not be told that the tool ran").
		WithDetail("conversationId", conversation.ID)
}

func (s *Service) allowed() []string {
	if len(s.deps.Allowed) > 0 {
		return slices.Clone(s.deps.Allowed)
	}
	return s.deps.Registry.Names()
}

func (s *Service) replay(ctx context.Context, conversation domainagent.Conversation,
	action domainagent.PendingAction) (any, error) {
	if err := Permit(s.deps.Allowed, action.Tool); err != nil {
		return nil, err
	}

	return s.deps.Registry.Call(ctx, tools.Binding{
		SiteID: siteOf(conversation), ConversationID: conversation.ID,
		Mode: conversation.Mode, Approved: true,
	}, action.Tool, action.Args)
}

func (s *Service) audit(ctx context.Context, action domainagent.PendingAction, failure string,
	denied bool, elapsed int64) error {
	call, err := domainagent.NewToolCall(domainagent.ToolCall{
		ID: id.New(), ConversationID: action.ConversationID, CallID: action.ID, Tool: action.Tool,
		Args: tools.Redact(action.Args), Status: callStatus(failure, denied), DurationMS: elapsed,
		Error: failure, CreatedAt: s.now(),
	})
	if err != nil {
		return err
	}
	return s.deps.Calls.Insert(ctx, call)
}

func callStatus(failure string, denied bool) domainagent.CallStatus {
	switch {
	case denied:
		return domainagent.CallDenied
	case failure != "":
		return domainagent.CallError
	default:
		return domainagent.CallOK
	}
}

func capped(encoded json.RawMessage, limit int) json.RawMessage {
	if limit <= 0 || len(encoded) <= limit {
		return encoded
	}

	var document map[string]any
	if json.Unmarshal(encoded, &document) != nil {
		held, err := json.Marshal(preview(string(encoded), len(encoded), limit))
		if err != nil {
			return encoded
		}
		return held
	}

	capped, cut := Cap(document, limit)
	if !cut {
		return encoded
	}

	shortened, err := json.Marshal(capped)
	if err != nil {
		return encoded
	}
	return shortened
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
	fenced, err := json.Marshal(Fence(result))
	if err != nil {
		return "The confirmed tool " + tool + " ran and answered something that cannot be read back."
	}
	return "The confirmed tool " + tool + " ran. Its result is " + string(fenced)
}
