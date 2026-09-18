package wails

import (
	"context"

	"go.uber.org/zap"

	"github.com/davidmovas/postulator/internal/application/agent"
	"github.com/davidmovas/postulator/internal/kernel/middleware"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

type agentUseCase interface {
	CreateConversation(ctx context.Context, req agent.CreateConversationRequest) (agent.CreateConversationResponse, error)
	SetMode(ctx context.Context, req agent.SetModeRequest) (agent.SetModeResponse, error)
	Send(ctx context.Context, req agent.SendRequest) (agent.SendResponse, error)
	Confirm(ctx context.Context, req agent.ConfirmRequest) (agent.ConfirmResponse, error)
	Cancel(ctx context.Context, req agent.CancelRequest) (agent.CancelResponse, error)
	ListConversations(ctx context.Context, req agent.ListConversationsRequest) (paging.List[agent.Conversation], error)
	ListMessages(ctx context.Context, req agent.ListMessagesRequest) (paging.List[agent.Message], error)
	ListPendingActions(ctx context.Context, req agent.ListPendingActionsRequest) (paging.List[agent.PendingAction], error)
}

type AgentService struct {
	createConversation middleware.Handler[agent.CreateConversationRequest, agent.CreateConversationResponse]
	setMode            middleware.Handler[agent.SetModeRequest, agent.SetModeResponse]
	send               middleware.Handler[agent.SendRequest, agent.SendResponse]
	confirm            middleware.Handler[agent.ConfirmRequest, agent.ConfirmResponse]
	cancel             middleware.Handler[agent.CancelRequest, agent.CancelResponse]
	listConversations  middleware.Handler[agent.ListConversationsRequest, paging.List[agent.Conversation]]
	listMessages       middleware.Handler[agent.ListMessagesRequest, paging.List[agent.Message]]
	listPendingActions middleware.Handler[agent.ListPendingActionsRequest, paging.List[agent.PendingAction]]
}

func NewAgentService(logger *zap.Logger, useCase agentUseCase) *AgentService {
	return &AgentService{
		createConversation: Wrap(logger, "agent.createConversation", useCase.CreateConversation),
		setMode:            Wrap(logger, "agent.setMode", useCase.SetMode),
		send:               Wrap(logger, "agent.send", useCase.Send),
		confirm:            Wrap(logger, "agent.confirm", useCase.Confirm),
		cancel:             Wrap(logger, "agent.cancel", useCase.Cancel),
		listConversations:  Wrap(logger, "agent.listConversations", useCase.ListConversations),
		listMessages:       Wrap(logger, "agent.listMessages", useCase.ListMessages),
		listPendingActions: Wrap(logger, "agent.listPendingActions", useCase.ListPendingActions),
	}
}

func (s *AgentService) CreateConversation(c context.Context, req agent.CreateConversationRequest) (agent.CreateConversationResponse, error) {
	return s.createConversation(c, req)
}

func (s *AgentService) SetMode(c context.Context, req agent.SetModeRequest) (agent.SetModeResponse, error) {
	return s.setMode(c, req)
}

func (s *AgentService) Send(c context.Context, req agent.SendRequest) (agent.SendResponse, error) {
	return s.send(c, req)
}

func (s *AgentService) Confirm(c context.Context, req agent.ConfirmRequest) (agent.ConfirmResponse, error) {
	return s.confirm(c, req)
}

func (s *AgentService) Cancel(c context.Context, req agent.CancelRequest) (agent.CancelResponse, error) {
	return s.cancel(c, req)
}

func (s *AgentService) ListConversations(c context.Context, req agent.ListConversationsRequest) (paging.List[agent.Conversation], error) {
	return s.listConversations(c, req)
}

func (s *AgentService) ListMessages(c context.Context, req agent.ListMessagesRequest) (paging.List[agent.Message], error) {
	return s.listMessages(c, req)
}

func (s *AgentService) ListPendingActions(c context.Context, req agent.ListPendingActionsRequest) (paging.List[agent.PendingAction], error) {
	return s.listPendingActions(c, req)
}
