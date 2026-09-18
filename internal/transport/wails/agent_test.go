package wails_test

import (
	"context"
	"testing"

	"go.uber.org/zap"

	"github.com/davidmovas/postulator/internal/application/agent"
	"github.com/davidmovas/postulator/internal/kernel/paging"
	"github.com/davidmovas/postulator/internal/transport/wails"
)

type agentFake struct{ mode failure }

func (f agentFake) CreateConversation(context.Context, agent.CreateConversationRequest) (agent.CreateConversationResponse, error) {
	return answer[agent.CreateConversationResponse](f.mode)
}

func (f agentFake) SetMode(context.Context, agent.SetModeRequest) (agent.SetModeResponse, error) {
	return answer[agent.SetModeResponse](f.mode)
}

func (f agentFake) Send(context.Context, agent.SendRequest) (agent.SendResponse, error) {
	return answer[agent.SendResponse](f.mode)
}

func (f agentFake) Confirm(context.Context, agent.ConfirmRequest) (agent.ConfirmResponse, error) {
	return answer[agent.ConfirmResponse](f.mode)
}

func (f agentFake) Cancel(context.Context, agent.CancelRequest) (agent.CancelResponse, error) {
	return answer[agent.CancelResponse](f.mode)
}

func (f agentFake) ListConversations(context.Context, agent.ListConversationsRequest) (paging.List[agent.Conversation], error) {
	return answer[paging.List[agent.Conversation]](f.mode)
}

func (f agentFake) ListMessages(context.Context, agent.ListMessagesRequest) (paging.List[agent.Message], error) {
	return answer[paging.List[agent.Message]](f.mode)
}

func (f agentFake) ListPendingActions(context.Context, agent.ListPendingActionsRequest) (paging.List[agent.PendingAction], error) {
	return answer[paging.List[agent.PendingAction]](f.mode)
}

func TestAgentServiceConvertsEveryFailure(t *testing.T) {
	t.Parallel()

	assertMethodNames(t, wails.NewAgentService(zap.NewNop(), ready[wails.AgentUseCase](agentFake{})), []string{
		"Cancel", "Confirm", "CreateConversation", "ListConversations", "ListMessages",
		"ListPendingActions", "Send", "SetMode",
	})
	assertEveryMethodConverts(t, wails.NewAgentService(zap.NewNop(), ready[wails.AgentUseCase](agentFake{mode: missing})), missingBody)
	assertEveryMethodConverts(t, wails.NewAgentService(zap.NewNop(), ready[wails.AgentUseCase](agentFake{mode: panicking})), panicBody)
}
