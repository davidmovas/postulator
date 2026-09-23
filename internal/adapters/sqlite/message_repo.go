package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/Masterminds/squirrel"

	"github.com/davidmovas/postulator/internal/domain/agent"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

const (
	messageColumns = `id, conversation_id, seq, role, text, tool, call_id, payload, created_at`
	insertMessage  = `INSERT INTO messages (` + messageColumns + `)
		SELECT ?, ?, coalesce(max(seq), 0) + 1, ?, ?, ?, ?, ?, ? FROM messages WHERE conversation_id = ?`
	selectMessage     = `SELECT ` + messageColumns + ` FROM messages WHERE id = ?`
	selectLatestSeq   = `SELECT coalesce(max(seq), 0) FROM messages WHERE conversation_id = ?`
	selectMessagesAll = `SELECT ` + messageColumns + ` FROM messages WHERE conversation_id = ? ORDER BY seq`
)

type MessageRepo struct {
	store *Store
}

func NewMessageRepo(store *Store) *MessageRepo {
	return &MessageRepo{store: store}
}

func messageNotFound(id string) *errors.Error {
	return errors.New(errors.NotFound, "message not found").WithDetail("messageId", id)
}

func (r *MessageRepo) Append(ctx context.Context, m agent.Message) (agent.Message, error) {
	_, err := execWrite(ctx, r.store.writeFrom(ctx), insertMessage, []any{
		m.ID, m.ConversationID, string(m.Role), m.Text, m.Tool, m.CallID, string(payloadOf(m.Payload)),
		formatTime(m.CreatedAt), m.ConversationID,
	}, errors.New(errors.Conflict, "a message with this id already exists"), "append the message")
	if err != nil {
		return agent.Message{}, err
	}
	return r.Get(ctx, m.ID)
}

func (r *MessageRepo) Get(ctx context.Context, id string) (agent.Message, error) {
	return selectOne(ctx, r.store.execFrom(ctx), selectMessage, []any{id}, scanMessage, messageNotFound(id),
		"read the message")
}

func (r *MessageRepo) ByConversation(ctx context.Context, conversationID string) ([]agent.Message, error) {
	return selectAll(ctx, r.store.execFrom(ctx), selectMessagesAll, []any{conversationID}, scanMessage,
		"read the conversation messages")
}

func (r *MessageRepo) LatestSeq(ctx context.Context, conversationID string) (int64, error) {
	return selectOne(ctx, r.store.execFrom(ctx), selectLatestSeq, []any{conversationID}, scanSeq, nil,
		"read the latest message sequence")
}

func (r *MessageRepo) List(ctx context.Context, q agent.MessageQuery, page paging.Request) (paging.List[agent.Message], error) {
	builder := squirrel.Select(messageColumns).From("messages").
		Where(squirrel.Eq{"conversation_id": q.ConversationID})

	keyset := messageKeyset(q.Desc)
	keyed, err := keyset.Apply(builder, page)
	if err != nil {
		return paging.List[agent.Message]{}, err
	}
	query, args, err := buildQuery(keyed, "messages")
	if err != nil {
		return paging.List[agent.Message]{}, err
	}
	rows, err := selectAll(ctx, r.store.execFrom(ctx), query, args, scanMessage, "list the messages")
	if err != nil {
		return paging.List[agent.Message]{}, err
	}
	return keyset.Cut(rows, page)
}

func messageKeyset(desc bool) paging.Keyset[agent.Message] {
	return paging.Keyset[agent.Message]{
		IDColumn: "id",
		ID:       func(m agent.Message) string { return m.ID },
		Keys: []paging.SortKey[agent.Message]{
			paging.IntKey[agent.Message]("seq", "seq", func(m agent.Message) any { return m.Seq }),
		},
		Desc: desc,
	}
}

func payloadOf(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return json.RawMessage("{}")
	}
	return raw
}

func scanSeq(rows *sql.Rows) (int64, error) {
	var seq int64
	if err := rows.Scan(&seq); err != nil {
		return 0, err
	}
	return seq, nil
}

func scanMessage(rows *sql.Rows) (agent.Message, error) {
	var (
		m         agent.Message
		role      string
		payload   string
		createdAt string
	)
	if err := rows.Scan(&m.ID, &m.ConversationID, &m.Seq, &role, &m.Text, &m.Tool, &m.CallID, &payload,
		&createdAt); err != nil {
		return agent.Message{}, err
	}

	m.Role = agent.Role(role)
	m.Payload = json.RawMessage(payload)

	var err error
	if m.CreatedAt, err = parseTime(createdAt); err != nil {
		return agent.Message{}, err
	}
	return m, nil
}
