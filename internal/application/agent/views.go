package agent

import (
	"encoding/json"

	domainagent "github.com/davidmovas/postulator/internal/domain/agent"
	"github.com/davidmovas/postulator/internal/kernel/dto"
)

type Conversation struct {
	ID        string   `json:"id"`
	SiteID    *string  `json:"siteId"`
	Title     string   `json:"title"`
	Mode      string   `json:"mode"`
	CreatedAt dto.Time `json:"createdAt"`
	UpdatedAt dto.Time `json:"updatedAt"`
}

type Message struct {
	ID             string          `json:"id"`
	ConversationID string          `json:"conversationId"`
	Seq            int64           `json:"seq"`
	Role           string          `json:"role"`
	Text           string          `json:"text"`
	Tool           string          `json:"tool,omitempty"`
	CallID         string          `json:"callId,omitempty"`
	Payload        json.RawMessage `json:"payload,omitempty"`
	CreatedAt      dto.Time        `json:"createdAt"`
}

type PendingAction struct {
	ID             string          `json:"id"`
	ConversationID string          `json:"conversationId"`
	Tool           string          `json:"tool"`
	Args           json.RawMessage `json:"args"`
	Summary        string          `json:"summary"`
	Status         string          `json:"status"`
	Result         json.RawMessage `json:"result,omitempty"`
	Error          string          `json:"error,omitempty"`
	CreatedAt      dto.Time        `json:"createdAt"`
	UpdatedAt      dto.Time        `json:"updatedAt"`
}

func conversationView(c domainagent.Conversation) Conversation {
	return Conversation{
		ID: c.ID, SiteID: c.SiteID, Title: c.Title, Mode: string(c.Mode),
		CreatedAt: dto.NewTime(c.CreatedAt), UpdatedAt: dto.NewTime(c.UpdatedAt),
	}
}

func messageView(m domainagent.Message) Message {
	return Message{
		ID: m.ID, ConversationID: m.ConversationID, Seq: m.Seq, Role: string(m.Role), Text: m.Text,
		Tool: m.Tool, CallID: m.CallID, Payload: m.Payload, CreatedAt: dto.NewTime(m.CreatedAt),
	}
}

func actionView(a domainagent.PendingAction) PendingAction {
	return PendingAction{
		ID: a.ID, ConversationID: a.ConversationID, Tool: a.Tool, Args: a.Args, Summary: a.Summary,
		Status: string(a.Status), Result: a.Result, Error: a.Error,
		CreatedAt: dto.NewTime(a.CreatedAt), UpdatedAt: dto.NewTime(a.UpdatedAt),
	}
}
