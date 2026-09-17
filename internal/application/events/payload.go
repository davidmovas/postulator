package events

import "encoding/json"

type GraphChangedPayload struct {
	SiteID string `json:"siteId"`
}

type PagesChangedPayload struct {
	SiteID string `json:"siteId"`
}

type TemplatesChangedPayload struct{}

type AgentDeltaPayload struct {
	ConversationID string `json:"conversationId"`
	MessageID      string `json:"messageId"`
	Seq            int64  `json:"seq"`
	Text           string `json:"text"`
}

type AgentToolStartedPayload struct {
	ConversationID string          `json:"conversationId"`
	CallID         string          `json:"callId"`
	Tool           string          `json:"tool"`
	Args           json.RawMessage `json:"args"`
}

type AgentToolFinishedPayload struct {
	ConversationID string          `json:"conversationId"`
	CallID         string          `json:"callId"`
	Tool           string          `json:"tool"`
	Result         json.RawMessage `json:"result"`
}

type AgentConfirmRequestedPayload struct {
	ConfirmationID string          `json:"confirmationId"`
	Tool           string          `json:"tool"`
	Args           json.RawMessage `json:"args"`
	Risk           string          `json:"risk"`
}

type AppLockedPayload struct{}

type AppUnlockedPayload struct{}

type RunQueuedPayload struct {
	RunID string `json:"runId"`
	Kind  string `json:"kind"`
	Items int    `json:"items"`
}

type RunStartedPayload struct {
	RunID string `json:"runId"`
}

type RunPausedPayload struct {
	RunID  string `json:"runId"`
	Reason string `json:"reason"`
}

type RunResumedPayload struct {
	RunID string `json:"runId"`
}

type RunCancelledPayload struct {
	RunID string `json:"runId"`
}

type RunCompletedPayload struct {
	RunID     string `json:"runId"`
	Succeeded int    `json:"succeeded"`
	Failed    int    `json:"failed"`
}

type RunFailedPayload struct {
	RunID   string `json:"runId"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type RunBudgetExceededPayload struct {
	RunID     string  `json:"runId"`
	SpentUSD  float64 `json:"spentUsd"`
	BudgetUSD float64 `json:"budgetUsd"`
}

type ItemStartedPayload struct {
	RunID  string `json:"runId"`
	ItemID string `json:"itemId"`
}

type ItemDonePayload struct {
	RunID  string `json:"runId"`
	ItemID string `json:"itemId"`
}

type ItemFailedPayload struct {
	RunID   string `json:"runId"`
	ItemID  string `json:"itemId"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ItemNeedsHumanPayload struct {
	RunID  string `json:"runId"`
	ItemID string `json:"itemId"`
	Reason string `json:"reason"`
}

type StepStartedPayload struct {
	RunID  string `json:"runId"`
	ItemID string `json:"itemId"`
	Step   string `json:"step"`
}

type StepDonePayload struct {
	RunID      string `json:"runId"`
	ItemID     string `json:"itemId"`
	Step       string `json:"step"`
	DurationMs int64  `json:"durationMs"`
}

type StepFailedPayload struct {
	RunID   string `json:"runId"`
	ItemID  string `json:"itemId"`
	Step    string `json:"step"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type StepRetryingPayload struct {
	RunID   string `json:"runId"`
	ItemID  string `json:"itemId"`
	Step    string `json:"step"`
	Attempt int    `json:"attempt"`
	AfterMs int64  `json:"afterMs"`
}

type LLMUsagePayload struct {
	RunID            string  `json:"runId"`
	ItemID           string  `json:"itemId"`
	Provider         string  `json:"provider"`
	Model            string  `json:"model"`
	PromptTokens     int     `json:"promptTokens"`
	CompletionTokens int     `json:"completionTokens"`
	USD              float64 `json:"usd"`
}
