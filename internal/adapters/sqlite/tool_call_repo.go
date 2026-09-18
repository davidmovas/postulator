package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/davidmovas/postulator/internal/domain/agent"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	toolCallColumns = `id, conversation_id, call_id, tool, args, status, duration_ms, error, created_at`
	insertToolCall  = `INSERT INTO tool_calls (` + toolCallColumns + `) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
	selectToolCalls = `SELECT ` + toolCallColumns + ` FROM tool_calls WHERE conversation_id = ?
		ORDER BY created_at, id`

	historyColumns = `conversation_id, history, version, updated_at`
	upsertHistory  = `INSERT INTO conversation_histories (` + historyColumns + `) VALUES (?, ?, ?, ?)
		ON CONFLICT (conversation_id) DO UPDATE SET history = excluded.history, version = excluded.version,
		updated_at = excluded.updated_at`
	selectHistory = `SELECT history, version FROM conversation_histories WHERE conversation_id = ?`
)

type ToolCallRepo struct {
	store *Store
}

func NewToolCallRepo(store *Store) *ToolCallRepo {
	return &ToolCallRepo{store: store}
}

func (r *ToolCallRepo) Insert(ctx context.Context, c agent.ToolCall) error {
	_, err := execWrite(ctx, r.store.writeFrom(ctx), insertToolCall, []any{
		c.ID, c.ConversationID, c.CallID, c.Tool, string(payloadOf(c.Args)), string(c.Status), c.DurationMS,
		c.Error, formatTime(c.CreatedAt),
	}, errors.New(errors.Conflict, "a tool call with this id already exists"), "record the tool call")
	return err
}

func (r *ToolCallRepo) ByConversation(ctx context.Context, conversationID string) ([]agent.ToolCall, error) {
	return selectAll(ctx, r.store.execFrom(ctx), selectToolCalls, []any{conversationID}, scanToolCall,
		"list the tool calls")
}

func scanToolCall(rows *sql.Rows) (agent.ToolCall, error) {
	var (
		c            agent.ToolCall
		args, status string
		createdAt    string
	)
	if err := rows.Scan(&c.ID, &c.ConversationID, &c.CallID, &c.Tool, &args, &status, &c.DurationMS, &c.Error,
		&createdAt); err != nil {
		return agent.ToolCall{}, err
	}

	c.Args = json.RawMessage(args)
	c.Status = agent.CallStatus(status)

	var err error
	if c.CreatedAt, err = parseTime(createdAt); err != nil {
		return agent.ToolCall{}, err
	}
	return c, nil
}

type ConversationHistoryRepo struct {
	store *Store
}

func NewConversationHistoryRepo(store *Store) *ConversationHistoryRepo {
	return &ConversationHistoryRepo{store: store}
}

type storedHistory struct {
	body    []byte
	version int
}

func (r *ConversationHistoryRepo) Load(ctx context.Context, conversationID string) (body []byte, version int, err error) {
	stored, selectErr := selectOne(ctx, r.store.execFrom(ctx), selectHistory, []any{conversationID}, scanStoredHistory,
		errors.New(errors.NotFound, "the conversation carries no history yet").
			WithDetail("conversationId", conversationID), "read the conversation history")
	if selectErr != nil {
		return nil, 0, selectErr
	}
	return stored.body, stored.version, nil
}

func (r *ConversationHistoryRepo) Save(ctx context.Context, conversationID string, body []byte, version int, at time.Time) error {
	_, err := execWrite(ctx, r.store.writeFrom(ctx), upsertHistory,
		[]any{conversationID, string(body), version, formatTime(at)}, nil,
		"save the conversation history")
	return err
}

func scanStoredHistory(rows *sql.Rows) (storedHistory, error) {
	var stored storedHistory
	var body string
	if err := rows.Scan(&body, &stored.version); err != nil {
		return storedHistory{}, err
	}
	stored.body = []byte(body)
	return stored, nil
}
