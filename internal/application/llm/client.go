package llm

import (
	"context"

	domain "github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

func (r Role) Valid() bool {
	return r == RoleUser || r == RoleAssistant
}

type Message struct {
	Role Role   `json:"role"`
	Text string `json:"text"`
}

type CallMeta struct {
	RunID          string `json:"runId"`
	ItemID         string `json:"itemId"`
	Step           string `json:"step"`
	ConversationID string `json:"conversationId"`
}

type Request struct {
	Temperature *float64        `json:"temperature,omitempty"`
	Schema      *Schema         `json:"schema,omitempty"`
	Ref         domain.ModelRef `json:"ref"`
	System      string          `json:"system"`
	Messages    []Message       `json:"messages"`
	Meta        CallMeta        `json:"meta"`
	MaxTokens   int             `json:"maxTokens"`
}

const (
	maxTemperature = 2.0
	minTemperature = 0.0
)

func (r Request) Validate() error {
	if !r.Ref.Valid() {
		return errors.New(errors.Invalid, "the model reference must name a provider and a model")
	}
	if len(r.Messages) == 0 {
		return errors.New(errors.Invalid, "the request must carry at least one message")
	}
	for i, message := range r.Messages {
		if !message.Role.Valid() {
			return errors.New(errors.Invalid, "a message role must be user or assistant").WithDetail("index", i)
		}
		if message.Text == "" {
			return errors.New(errors.Invalid, "a message must not be empty").WithDetail("index", i)
		}
	}
	if r.Messages[len(r.Messages)-1].Role != RoleUser {
		return errors.New(errors.Invalid, "the last message must come from the user")
	}
	if r.MaxTokens < 0 {
		return errors.New(errors.Invalid, "the token ceiling must not be negative")
	}
	if r.Temperature != nil && (*r.Temperature < minTemperature || *r.Temperature > maxTemperature) {
		return errors.New(errors.Invalid, "the temperature must be between 0 and 2")
	}
	return nil
}

type FinishReason string

const (
	FinishStop          FinishReason = "stop"
	FinishLength        FinishReason = "length"
	FinishContentFilter FinishReason = "content_filter"
)

type Response struct {
	Text         string       `json:"text"`
	Usage        domain.Usage `json:"usage"`
	FinishReason FinishReason `json:"finishReason"`
}

type Delta struct {
	Usage *domain.Usage `json:"usage,omitempty"`
	Err   error         `json:"-"`
	Text  string        `json:"text"`
	Done  bool          `json:"done"`
}

type Client interface {
	Complete(ctx context.Context, req Request) (Response, error)
	Stream(ctx context.Context, req Request) (<-chan Delta, error)
}
