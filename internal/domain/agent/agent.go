package agent

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type Mode string

const (
	ModeConfirm    Mode = "confirm"
	ModeAutonomous Mode = "autonomous"
)

func (m Mode) Valid() bool {
	switch m {
	case ModeConfirm, ModeAutonomous:
		return true
	default:
		return false
	}
}

type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

func (r Role) Valid() bool {
	switch r {
	case RoleUser, RoleAssistant, RoleTool:
		return true
	default:
		return false
	}
}

const MaxTitle = 120

type Conversation struct {
	ID           string
	SiteID       *string
	Title        string
	Mode         Mode
	TitleSettled bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func invalid(message, field string) *errors.Error {
	return errors.New(errors.Invalid, message).WithDetail("field", field)
}

func NewConversation(c Conversation) (Conversation, error) {
	if c.Mode == "" {
		c.Mode = ModeConfirm
	}
	c.Title = Title(c.Title)

	switch {
	case c.ID == "":
		return Conversation{}, invalid("conversation id must not be empty", "id")
	case !c.Mode.Valid():
		return Conversation{}, invalid("conversation mode must be confirm or autonomous", "mode")
	case c.SiteID != nil && *c.SiteID == "":
		return Conversation{}, invalid("conversation site id must not be empty when set", "siteId")
	case c.CreatedAt.IsZero():
		return Conversation{}, invalid("a conversation needs a creation timestamp", "createdAt")
	}
	return c, nil
}

const MaxSuggestedTitle = 60

var titleQuotes = "\"'“”‘’«»„"

var titleLabels = []string{"title:", "chat title:", "conversation title:"}

func SuggestedTitle(raw string) (string, bool) {
	cleaned := strings.Join(strings.Fields(raw), " ")
	for _, label := range titleLabels {
		if len(cleaned) >= len(label) && strings.EqualFold(cleaned[:len(label)], label) {
			cleaned = strings.TrimSpace(cleaned[len(label):])
			break
		}
	}

	cleaned = strings.Trim(cleaned, titleQuotes)
	cleaned = strings.TrimRight(cleaned, ".")
	cleaned = strings.Trim(cleaned, titleQuotes)
	cleaned = strings.TrimSpace(cleaned)

	if cleaned == "" || len([]rune(cleaned)) > MaxSuggestedTitle {
		return "", false
	}
	return cleaned, true
}

func Title(raw string) string {
	trimmed := strings.Join(strings.Fields(raw), " ")
	if len([]rune(trimmed)) <= MaxTitle {
		return trimmed
	}
	return strings.TrimSpace(string([]rune(trimmed)[:MaxTitle]))
}

type Message struct {
	ID             string
	ConversationID string
	Seq            int64
	Role           Role
	Text           string
	Tool           string
	CallID         string
	Payload        json.RawMessage
	CreatedAt      time.Time
}

func NewMessage(m Message) (Message, error) {
	if len(m.Payload) == 0 {
		m.Payload = json.RawMessage("{}")
	}

	switch {
	case m.ID == "":
		return Message{}, invalid("message id must not be empty", "id")
	case m.ConversationID == "":
		return Message{}, invalid("message conversation id must not be empty", "conversationId")
	case m.Seq <= 0:
		return Message{}, invalid("message sequence must be positive", "seq")
	case !m.Role.Valid():
		return Message{}, invalid("message role is not recognized", "role")
	case m.Role == RoleTool && m.Tool == "":
		return Message{}, invalid("a tool message must name its tool", "tool")
	case m.CreatedAt.IsZero():
		return Message{}, invalid("a message needs a creation timestamp", "createdAt")
	}
	return m, nil
}

type ActionStatus string

const (
	ActionPending  ActionStatus = "pending"
	ActionApproved ActionStatus = "approved"
	ActionRejected ActionStatus = "rejected"
	ActionExecuted ActionStatus = "executed"
	ActionFailed   ActionStatus = "failed"
)

func (s ActionStatus) Valid() bool {
	switch s {
	case ActionPending, ActionApproved, ActionRejected, ActionExecuted, ActionFailed:
		return true
	default:
		return false
	}
}

func (s ActionStatus) Settled() bool {
	return s == ActionRejected || s == ActionExecuted || s == ActionFailed
}

const ConfirmationRequired = "confirmationRequired"

type PendingAction struct {
	ID             string
	ConversationID string
	Tool           string
	Args           json.RawMessage
	Summary        string
	Status         ActionStatus
	Result         json.RawMessage
	Error          string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func NewPendingAction(a PendingAction) (PendingAction, error) {
	if a.Status == "" {
		a.Status = ActionPending
	}
	if len(a.Args) == 0 {
		a.Args = json.RawMessage("{}")
	}

	switch {
	case a.ID == "":
		return PendingAction{}, invalid("pending action id must not be empty", "id")
	case a.ConversationID == "":
		return PendingAction{}, invalid("pending action conversation id must not be empty", "conversationId")
	case a.Tool == "":
		return PendingAction{}, invalid("a pending action must name its tool", "tool")
	case a.Summary == "":
		return PendingAction{}, invalid("a pending action must carry a summary a human can read", "summary")
	case !a.Status.Valid():
		return PendingAction{}, invalid("pending action status is not recognized", "status")
	case a.CreatedAt.IsZero():
		return PendingAction{}, invalid("a pending action needs a creation timestamp", "createdAt")
	}
	return a, nil
}

type CallStatus string

const (
	CallOK     CallStatus = "ok"
	CallDenied CallStatus = "denied"
	CallError  CallStatus = "error"
)

func (s CallStatus) Valid() bool {
	switch s {
	case CallOK, CallDenied, CallError:
		return true
	default:
		return false
	}
}

type ToolCall struct {
	ID             string
	ConversationID string
	CallID         string
	Tool           string
	Args           json.RawMessage
	Status         CallStatus
	DurationMS     int64
	Error          string
	CreatedAt      time.Time
}

func NewToolCall(c ToolCall) (ToolCall, error) {
	if len(c.Args) == 0 {
		c.Args = json.RawMessage("{}")
	}

	switch {
	case c.ID == "":
		return ToolCall{}, invalid("tool call id must not be empty", "id")
	case c.ConversationID == "":
		return ToolCall{}, invalid("tool call conversation id must not be empty", "conversationId")
	case c.Tool == "":
		return ToolCall{}, invalid("a tool call must name its tool", "tool")
	case !c.Status.Valid():
		return ToolCall{}, invalid("tool call status is not recognized", "status")
	case c.DurationMS < 0:
		return ToolCall{}, invalid("a tool call duration must not be negative", "durationMs")
	case c.CreatedAt.IsZero():
		return ToolCall{}, invalid("a tool call needs a creation timestamp", "createdAt")
	}
	return c, nil
}

type ConversationQuery struct {
	SiteID string
	Desc   bool
}

type MessageQuery struct {
	ConversationID string
	Desc           bool
}

type ActionQuery struct {
	ConversationID string
	Status         *ActionStatus
	Desc           bool
}
