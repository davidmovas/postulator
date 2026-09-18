package agent

import (
	"context"
	"encoding/json"

	"github.com/davidmovas/postulator/internal/application/tools"
	domainllm "github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/kernel/dto"
)

const (
	UntrustedMarker = "untrustedContent"
	UntrustedData   = "data"
)

type SiteContext struct {
	SiteName  string
	Mode      string
	Templates []string
	Tools     []string
	Entities  int
	Pages     int
	Published int
	Unmapped  int
}

type Stream interface {
	Delta(ctx context.Context, seq int64, text string) error
	ToolStarted(ctx context.Context, callID, tool string, args json.RawMessage) error
	ToolFinished(ctx context.Context, outcome ToolOutcome) error
}

type ToolOutcome struct {
	CallID     string
	Tool       string
	Args       json.RawMessage
	Result     json.RawMessage
	Status     string
	Error      string
	DurationMS int64
}

type RunSpec struct {
	Binding       tools.Binding
	Ref           domainllm.ModelRef
	Context       SiteContext
	Input         string
	MessageID     string
	Allowed       []string
	Stream        Stream
	LoopLimit     int
	HistoryBudget int
}

type RunResult struct {
	Text      string
	ToolCalls int
	Usage     domainllm.Usage
	USD       float64
}

type CreateConversationRequest struct {
	SiteID string `json:"siteId,omitempty"`
	Title  string `json:"title,omitempty"`
	Mode   string `json:"mode,omitempty" enum:"confirm,autonomous"`
}

type CreateConversationResponse struct {
	Conversation Conversation `json:"conversation"`
}

type SendRequest struct {
	ConversationID string `json:"conversationId"`
	Text           string `json:"text"`
}

type SendResponse struct {
	MessageID string `json:"messageId"`
}

type ConfirmRequest struct {
	ActionID string `json:"actionId"`
	Approve  bool   `json:"approve"`
}

type ConfirmResponse struct {
	Action PendingAction `json:"action"`
}

type ListConversationsRequest struct {
	dto.ListRequest
	SiteID string `json:"siteId,omitempty"`
}

type ListMessagesRequest struct {
	dto.ListRequest
	ConversationID string `json:"conversationId"`
}

type SetModeRequest struct {
	ConversationID string `json:"conversationId"`
	Mode           string `json:"mode" enum:"confirm,autonomous"`
}

type SetModeResponse struct {
	Conversation Conversation `json:"conversation"`
}

type CancelRequest struct {
	ConversationID string `json:"conversationId"`
}

type CancelResponse struct {
	Cancelled bool `json:"cancelled"`
}

type ListPendingActionsRequest struct {
	dto.ListRequest
	ConversationID string `json:"conversationId,omitempty"`
	Status         string `json:"status,omitempty" enum:"pending,approved,rejected,executed,failed"`
}
