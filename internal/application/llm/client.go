package llm

import (
	"context"
	"encoding/json"

	domain "github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleDeveloper Role = "developer"
	RoleTool      Role = "tool"
)

func (r Role) Valid() bool {
	switch r {
	case RoleUser, RoleAssistant, RoleDeveloper, RoleTool:
		return true
	default:
		return false
	}
}

type ToolCall struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Args      json.RawMessage `json:"args"`
	Namespace string          `json:"namespace,omitempty"`
}

type ToolResult struct {
	CallID string          `json:"callId"`
	Output json.RawMessage `json:"output"`
}

type SearchKind string

const (
	SearchCall   SearchKind = "call"
	SearchOutput SearchKind = "output"
)

func (k SearchKind) Valid() bool {
	return k == SearchCall || k == SearchOutput
}

type ToolSearch struct {
	Kind      SearchKind      `json:"kind"`
	CallID    string          `json:"callId,omitempty"`
	Execution string          `json:"execution,omitempty"`
	Payload   json.RawMessage `json:"payload"`
}

type Message struct {
	Role   Role        `json:"role"`
	Text   string      `json:"text"`
	Call   *ToolCall   `json:"call,omitempty"`
	Result *ToolResult `json:"result,omitempty"`
	Search *ToolSearch `json:"search,omitempty"`
}

type ToolGroup struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type Tool struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Schema      *Schema    `json:"schema,omitempty"`
	Deferred    *ToolGroup `json:"deferred,omitempty"`
}

type CallMeta struct {
	RunID          string      `json:"runId"`
	ItemID         string      `json:"itemId"`
	Step           string      `json:"step"`
	ConversationID string      `json:"conversationId"`
	Role           domain.Role `json:"role,omitempty"`
}

type Request struct {
	Temperature *float64               `json:"temperature,omitempty"`
	Schema      *Schema                `json:"schema,omitempty"`
	Ref         domain.ModelRef        `json:"ref"`
	System      string                 `json:"system"`
	Messages    []Message              `json:"messages"`
	Meta        CallMeta               `json:"meta"`
	MaxTokens   int                    `json:"maxTokens"`
	Tools       []Tool                 `json:"tools,omitempty"`
	Effort      domain.ReasoningEffort `json:"effort,omitempty"`
	Tier        domain.ServiceTier     `json:"tier,omitempty"`
	CacheKey    string                 `json:"cacheKey,omitempty"`
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
	called := make(map[string]struct{})
	for i, message := range r.Messages {
		if problem := message.check(called); problem != nil {
			return problem.WithDetail("index", i)
		}
		if message.Call != nil {
			called[message.Call.ID] = struct{}{}
		}
	}
	if r.Messages[len(r.Messages)-1].Role == RoleAssistant {
		return errors.New(errors.Invalid, "the last message must come from the user, the developer or a tool")
	}
	if r.MaxTokens < 0 {
		return errors.New(errors.Invalid, "the token ceiling must not be negative")
	}
	if r.Temperature != nil && (*r.Temperature < minTemperature || *r.Temperature > maxTemperature) {
		return errors.New(errors.Invalid, "the temperature must be between 0 and 2")
	}
	if r.Effort != "" && !r.Effort.Valid() {
		return errors.New(errors.Invalid, "the reasoning effort must be none, low, medium, high or xhigh")
	}
	if r.Tier != "" && !r.Tier.Valid() {
		return errors.New(errors.Invalid, "the service tier must be default or flex")
	}
	return checkTools(r.Tools)
}

func (m Message) check(called map[string]struct{}) *errors.Error {
	if !m.Role.Valid() {
		return errors.New(errors.Invalid, "a message role must be user, assistant, developer or tool")
	}
	carried := m.carried()
	if carried == 0 {
		return errors.New(errors.Invalid, "a message must not be empty")
	}
	if carried > 1 {
		return errors.New(errors.Invalid, "a message carries only one of text, a tool call, a tool search or a tool result")
	}
	if m.Call != nil && m.Role != RoleAssistant {
		return errors.New(errors.Invalid, "a tool call must come from the assistant")
	}
	if m.Search != nil && m.Role != RoleAssistant {
		return errors.New(errors.Invalid, "a tool search must come from the assistant")
	}
	if m.Result != nil && m.Role != RoleTool {
		return errors.New(errors.Invalid, "a tool result must travel in a tool message")
	}
	if m.Role == RoleTool && m.Result == nil {
		return errors.New(errors.Invalid, "a tool message must carry a tool result")
	}
	switch {
	case m.Call != nil:
		return m.Call.check()
	case m.Result != nil:
		return m.Result.check(called)
	case m.Search != nil:
		return m.Search.check()
	default:
		return nil
	}
}

func (m Message) carried() int {
	carried := 0
	if m.Text != "" {
		carried++
	}
	if m.Call != nil {
		carried++
	}
	if m.Result != nil {
		carried++
	}
	if m.Search != nil {
		carried++
	}
	return carried
}

func (s ToolSearch) check() *errors.Error {
	if !s.Kind.Valid() {
		return errors.New(errors.Invalid, "a tool search is a call or its output").WithDetail("kind", string(s.Kind))
	}
	if !json.Valid(s.Payload) {
		return errors.New(errors.Invalid, "a tool search must carry JSON").WithDetail("kind", string(s.Kind))
	}
	return nil
}

func (c ToolCall) check() *errors.Error {
	if c.ID == "" {
		return errors.New(errors.Invalid, "a tool call needs an identifier")
	}
	if c.Name == "" {
		return errors.New(errors.Invalid, "a tool call must name its tool")
	}
	if !json.Valid(c.Args) {
		return errors.New(errors.Invalid, "a tool call's arguments must be JSON").WithDetail("tool", c.Name)
	}
	return nil
}

func (r ToolResult) check(called map[string]struct{}) *errors.Error {
	if r.CallID == "" {
		return errors.New(errors.Invalid, "a tool result must name the call it answers")
	}
	if !json.Valid(r.Output) {
		return errors.New(errors.Invalid, "a tool result must be JSON").WithDetail("callId", r.CallID)
	}
	if _, ok := called[r.CallID]; !ok {
		return errors.New(errors.Invalid, "a tool result must answer a call made before it").WithDetail("callId", r.CallID)
	}
	return nil
}

func checkTools(tools []Tool) error {
	named := make(map[string]struct{}, len(tools))
	described := map[string]string{}
	for i, tool := range tools {
		if tool.Name == "" {
			return errors.New(errors.Invalid, "a tool needs a name").WithDetail("index", i)
		}
		if _, taken := named[tool.Name]; taken {
			return errors.New(errors.Invalid, "two tools share one name").WithDetail("tool", tool.Name)
		}
		named[tool.Name] = struct{}{}
		if tool.Schema != nil && tool.Schema.Type != SchemaObject {
			return errors.New(errors.Invalid, "a tool's parameters must be an object").WithDetail("tool", tool.Name)
		}
		if tool.Deferred != nil {
			if problem := tool.Deferred.check(described); problem != nil {
				return problem.WithDetail("tool", tool.Name)
			}
		}
	}
	return nil
}

func (g ToolGroup) check(described map[string]string) *errors.Error {
	if g.Name == "" || g.Description == "" {
		return errors.New(errors.Invalid, "a deferred tool needs a group with a name and a description")
	}
	if held, seen := described[g.Name]; seen && held != g.Description {
		return errors.New(errors.Invalid, "a tool group is described two ways").WithDetail("group", g.Name)
	}
	described[g.Name] = g.Description
	return nil
}

type FinishReason string

const (
	FinishStop          FinishReason = "stop"
	FinishLength        FinishReason = "length"
	FinishContentFilter FinishReason = "content_filter"
)

type Response struct {
	Text         string             `json:"text"`
	Usage        domain.Usage       `json:"usage"`
	FinishReason FinishReason       `json:"finishReason"`
	Calls        []ToolCall         `json:"calls,omitempty"`
	Tier         domain.ServiceTier `json:"tier,omitempty"`
	Searches     []ToolSearch       `json:"searches,omitempty"`
}

type Delta struct {
	Usage  *domain.Usage      `json:"usage,omitempty"`
	Err    error              `json:"-"`
	Text   string             `json:"text"`
	Done   bool               `json:"done"`
	Call   *ToolCall          `json:"call,omitempty"`
	Finish FinishReason       `json:"finish,omitempty"`
	Tier   domain.ServiceTier `json:"tier,omitempty"`
	Search *ToolSearch        `json:"search,omitempty"`
}

type Client interface {
	Complete(ctx context.Context, req Request) (Response, error)
	Stream(ctx context.Context, req Request) (<-chan Delta, error)
}
