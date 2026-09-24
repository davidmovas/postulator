package events

import "encoding/json"

type GraphChangedPayload struct {
	SiteID string `json:"siteId"`
}

type PagesChangedPayload struct {
	SiteID string `json:"siteId"`
}

type TemplatesChangedPayload struct{}

type SitesChangedPayload struct {
	SiteID string `json:"siteId"`
}

type SchedulesChangedPayload struct {
	SiteID string `json:"siteId"`
}

type SettingsChangedPayload struct{}

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

type AgentUsagePayload struct {
	ConversationID    string  `json:"conversationId"`
	MessageID         string  `json:"messageId"`
	Provider          string  `json:"provider"`
	Model             string  `json:"model"`
	Round             int     `json:"round"`
	InputTokens       int     `json:"inputTokens"`
	CachedInputTokens int     `json:"cachedInputTokens"`
	OutputTokens      int     `json:"outputTokens"`
	USD               float64 `json:"usd"`
}

type AgentWaitingPayload struct {
	ConversationID string `json:"conversationId"`
	MessageID      string `json:"messageId"`
	Reason         string `json:"reason"`
	Attempt        int    `json:"attempt"`
	AfterMs        int64  `json:"afterMs"`
}

type AgentDonePayload struct {
	ConversationID    string  `json:"conversationId"`
	MessageID         string  `json:"messageId"`
	Text              string  `json:"text"`
	Code              string  `json:"code"`
	Error             string  `json:"error"`
	InputTokens       int     `json:"inputTokens"`
	CachedInputTokens int     `json:"cachedInputTokens"`
	OutputTokens      int     `json:"outputTokens"`
	Calls             int     `json:"calls"`
	USD               float64 `json:"usd"`
}

type AgentTitledPayload struct {
	ConversationID string `json:"conversationId"`
	Title          string `json:"title"`
}

type AgentToolFinishedPayload struct {
	ConversationID string          `json:"conversationId"`
	CallID         string          `json:"callId"`
	Tool           string          `json:"tool"`
	Result         json.RawMessage `json:"result"`
	Status         string          `json:"status"`
	Error          string          `json:"error"`
	DurationMs     int64           `json:"durationMs"`
}

type AgentConfirmRequestedPayload struct {
	ConversationID string          `json:"conversationId"`
	ConfirmationID string          `json:"confirmationId"`
	Tool           string          `json:"tool"`
	Args           json.RawMessage `json:"args"`
	Risk           string          `json:"risk"`
	Summary        string          `json:"summary"`
}

type AgentConfirmResolvedPayload struct {
	ConversationID string          `json:"conversationId"`
	ConfirmationID string          `json:"confirmationId"`
	Tool           string          `json:"tool"`
	Status         string          `json:"status"`
	Result         json.RawMessage `json:"result"`
	Error          string          `json:"error"`
}

type AppLockedPayload struct{}

type FilesDroppedPayload struct {
	Paths []string `json:"paths"`
}

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
	RunID        string  `json:"runId"`
	SpentUSD     float64 `json:"spentUsd"`
	BudgetUSD    float64 `json:"budgetUsd"`
	SpentTokens  int     `json:"spentTokens"`
	BudgetTokens int     `json:"budgetTokens"`
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
	RunID   string `json:"runId"`
	ItemID  string `json:"itemId"`
	Reason  string `json:"reason"`
	Message string `json:"message"`
}

type ItemRestartedPayload struct {
	RunID  string `json:"runId"`
	ItemID string `json:"itemId"`
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
	Message    string `json:"message"`
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
	Code    string `json:"code"`
	Message string `json:"message"`
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
