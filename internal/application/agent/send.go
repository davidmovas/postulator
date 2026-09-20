package agent

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/application/tools"
	domainagent "github.com/davidmovas/postulator/internal/domain/agent"
	domainllm "github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/kernel/dto"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

func (s *Service) Send(ctx context.Context, req SendRequest) (SendResponse, error) {
	text := strings.TrimSpace(req.Text)
	if text == "" {
		return SendResponse{}, invalid("a message must carry something to answer", "text")
	}

	conversation, err := s.conversation(ctx, req.ConversationID)
	if err != nil {
		return SendResponse{}, err
	}
	if s.deps.Turns.Running(conversation.ID) {
		return SendResponse{}, errors.New(errors.Conflict, "this conversation is already answering").
			WithDetail("conversationId", conversation.ID)
	}

	asked, err := s.append(ctx, conversation.ID, domainagent.Message{
		Role: domainagent.RoleUser, Text: text,
	})
	if err != nil {
		return SendResponse{}, err
	}

	if conversation.Title == "" {
		conversation.Title = domainagent.Title(text)
		conversation.UpdatedAt = s.now()
		if titleErr := s.deps.Conversations.Update(ctx, conversation); titleErr != nil {
			return SendResponse{}, titleErr
		}
	}

	assistantID, turnErr := s.turn(ctx, conversation, text)
	if turnErr != nil {
		return SendResponse{}, turnErr
	}
	return SendResponse{MessageID: asked.ID, AssistantMessageID: assistantID}, nil
}

func (s *Service) Status(_ context.Context, req StatusRequest) (StatusResponse, error) {
	conversationID := strings.TrimSpace(req.ConversationID)
	if conversationID == "" {
		return StatusResponse{}, invalid("a status needs a conversation", "conversationId")
	}

	held, running := s.deps.Turns.Status(conversationID)
	if !running {
		return StatusResponse{}, nil
	}
	return StatusResponse{
		Running: true, MessageID: held.MessageID,
		StartedAt: dto.NewTime(held.StartedAt), LastSeq: held.LastSeq,
	}, nil
}

func (s *Service) turn(ctx context.Context, conversation domainagent.Conversation, input string) (string, error) {
	ref, err := s.deps.Profiles.Resolve(ctx, siteOf(conversation), domainllm.RoleChat, nil)
	if err != nil {
		return "", err
	}
	built, err := s.siteContext(ctx, conversation)
	if err != nil {
		return "", err
	}

	spec := RunSpec{
		Binding: tools.Binding{
			SiteID: siteOf(conversation), ConversationID: conversation.ID, Mode: conversation.Mode,
		},
		Ref:           ref,
		Context:       built,
		Input:         input,
		MessageID:     id.New(),
		Allowed:       s.allowed(),
		LoopLimit:     s.deps.LoopLimit,
		HistoryBudget: s.deps.HistoryBudget,
	}
	spec.Stream = &stream{service: s, conversationID: conversation.ID, messageID: spec.MessageID}

	if startErr := s.deps.Turns.Start(ctx, conversation.ID, spec.MessageID, s.now(),
		func(turnCtx context.Context) { s.answer(turnCtx, conversation.ID, spec) }); startErr != nil {
		return "", startErr
	}
	return spec.MessageID, nil
}

func (s *Service) answer(ctx context.Context, conversationID string, spec RunSpec) {
	defer func() {
		if recovered := recover(); recovered != nil {
			s.deps.Turns.Note(errors.New(errors.Internal, "the agent turn panicked"))
			s.emit(events.AgentDone, events.AgentDonePayload{
				ConversationID: conversationID, MessageID: spec.MessageID,
				Code: string(errors.Internal), Error: "the agent turn failed unexpectedly",
			})
		}
	}()

	result, err := s.deps.Runner.Run(ctx, spec)

	done := events.AgentDonePayload{ConversationID: conversationID, MessageID: spec.MessageID}
	if err != nil {
		done.Code, done.Error = describe(ctx, err)
		s.emit(events.AgentDone, done)
		return
	}

	if text := strings.TrimSpace(result.Text); text != "" {
		if _, appendErr := s.append(ctx, conversationID, domainagent.Message{
			ID: spec.MessageID, Role: domainagent.RoleAssistant, Text: text,
		}); appendErr != nil {
			done.Code, done.Error = describe(ctx, appendErr)
			s.emit(events.AgentDone, done)
			return
		}
		done.Text = text
	}

	done.InputTokens = result.Usage.Input
	done.OutputTokens = result.Usage.Output
	done.USD = result.USD
	s.emit(events.AgentDone, done)

	s.settle(ctx, conversationID, spec.Input, done.Text)
}

func (s *Service) append(ctx context.Context, conversationID string, message domainagent.Message) (domainagent.Message, error) {
	message.ConversationID = conversationID
	message.Seq = 1
	message.CreatedAt = s.now()
	if message.ID == "" {
		message.ID = id.New()
	}

	built, err := domainagent.NewMessage(message)
	if err != nil {
		return domainagent.Message{}, err
	}
	return s.deps.Messages.Append(ctx, built)
}

func (s *Service) emit(eventType events.Type, payload any) {
	if err := s.deps.Publisher.Publish(eventType, payload); err != nil {
		s.deps.Turns.Note(err)
	}
}

func describe(ctx context.Context, err error) (code, message string) {
	kernelCode, described := errors.Describe(err)
	if kernelCode == errors.Cancelled && expired(ctx) {
		described = deadlineMessage
	}
	return string(kernelCode), described
}

func siteOf(conversation domainagent.Conversation) string {
	if conversation.SiteID == nil {
		return ""
	}
	return *conversation.SiteID
}

type stream struct {
	service        *Service
	conversationID string
	messageID      string
}

func (s *stream) Delta(_ context.Context, seq int64, text string) error {
	s.service.deps.Turns.Observe(s.conversationID, seq)
	return s.service.deps.Publisher.Publish(events.AgentDelta, events.AgentDeltaPayload{
		ConversationID: s.conversationID, MessageID: s.messageID, Seq: seq, Text: text,
	})
}

func (s *stream) ToolStarted(_ context.Context, callID, tool string, args json.RawMessage) error {
	return s.service.deps.Publisher.Publish(events.AgentToolStarted, events.AgentToolStartedPayload{
		ConversationID: s.conversationID, CallID: callID, Tool: tool, Args: args,
	})
}

func (s *stream) ToolFinished(ctx context.Context, outcome ToolOutcome) error {
	call, err := domainagent.NewToolCall(domainagent.ToolCall{
		ID: id.New(), ConversationID: s.conversationID, CallID: outcome.CallID, Tool: outcome.Tool,
		Args: outcome.Args, Status: domainagent.CallStatus(outcome.Status), DurationMS: outcome.DurationMS,
		Error: outcome.Error, CreatedAt: s.service.now(),
	})
	if err != nil {
		return err
	}
	if insertErr := s.service.deps.Calls.Insert(ctx, call); insertErr != nil {
		return insertErr
	}
	if _, appendErr := s.service.append(ctx, s.conversationID, domainagent.Message{
		Role: domainagent.RoleTool, Tool: outcome.Tool, CallID: outcome.CallID, Payload: outcome.Result,
		Text: outcome.Error,
	}); appendErr != nil {
		return appendErr
	}

	return s.service.deps.Publisher.Publish(events.AgentToolFinished, events.AgentToolFinishedPayload{
		ConversationID: s.conversationID, CallID: outcome.CallID, Tool: outcome.Tool, Result: outcome.Result,
		Status: outcome.Status, Error: outcome.Error, DurationMs: outcome.DurationMS,
	})
}
