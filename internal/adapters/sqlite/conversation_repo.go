package sqlite

import (
	"context"
	"database/sql"

	"github.com/Masterminds/squirrel"

	"github.com/davidmovas/postulator/internal/domain/agent"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

const (
	conversationColumns = `id, site_id, title, mode, created_at, updated_at`
	insertConversation  = `INSERT INTO conversations (` + conversationColumns + `) VALUES (?, ?, ?, ?, ?, ?)`
	updateConversation  = `UPDATE conversations SET title = ?, mode = ?, updated_at = ? WHERE id = ?`
	selectConversation  = `SELECT ` + conversationColumns + ` FROM conversations WHERE id = ?`
	deleteConversation  = `DELETE FROM conversations WHERE id = ?`
)

type ConversationRepo struct {
	store *Store
}

func NewConversationRepo(store *Store) *ConversationRepo {
	return &ConversationRepo{store: store}
}

func conversationNotFound(id string) *errors.Error {
	return errors.New(errors.NotFound, "conversation not found").WithDetail("conversationId", id)
}

func (r *ConversationRepo) Insert(ctx context.Context, c agent.Conversation) error {
	_, err := execWrite(ctx, r.store.writeFrom(ctx), insertConversation, []any{
		c.ID, nullString(c.SiteID), c.Title, string(c.Mode), formatTime(c.CreatedAt), formatTime(c.UpdatedAt),
	}, errors.New(errors.Conflict, "a conversation with this id already exists"), "insert the conversation")
	return err
}

func (r *ConversationRepo) Update(ctx context.Context, c agent.Conversation) error {
	affected, err := execWrite(ctx, r.store.writeFrom(ctx), updateConversation,
		[]any{c.Title, string(c.Mode), formatTime(c.UpdatedAt), c.ID}, nil, "update the conversation")
	return requireAffected(affected, err, conversationNotFound(c.ID))
}

func (r *ConversationRepo) Delete(ctx context.Context, id string) error {
	affected, err := execWrite(ctx, r.store.writeFrom(ctx), deleteConversation, []any{id}, nil, "delete the conversation")
	return requireAffected(affected, err, conversationNotFound(id))
}

func (r *ConversationRepo) Get(ctx context.Context, id string) (agent.Conversation, error) {
	return selectOne(ctx, r.store.execFrom(ctx), selectConversation, []any{id}, scanConversation,
		conversationNotFound(id), "read the conversation")
}

func (r *ConversationRepo) List(ctx context.Context, q agent.ConversationQuery, page paging.Request) (paging.List[agent.Conversation], error) {
	builder := squirrel.Select(conversationColumns).From("conversations")
	if q.SiteID != "" {
		builder = builder.Where(squirrel.Eq{"site_id": q.SiteID})
	}

	keyset := conversationKeyset(q.Desc)
	keyed, err := keyset.Apply(builder, page)
	if err != nil {
		return paging.List[agent.Conversation]{}, err
	}
	query, args, err := buildQuery(keyed, "conversations")
	if err != nil {
		return paging.List[agent.Conversation]{}, err
	}
	rows, err := selectAll(ctx, r.store.execFrom(ctx), query, args, scanConversation, "list the conversations")
	if err != nil {
		return paging.List[agent.Conversation]{}, err
	}
	return keyset.Cut(rows, page)
}

func conversationKeyset(desc bool) paging.Keyset[agent.Conversation] {
	return paging.Keyset[agent.Conversation]{
		IDColumn: "id",
		ID:       func(c agent.Conversation) string { return c.ID },
		Keys: []paging.SortKey[agent.Conversation]{
			paging.TimeKey[agent.Conversation]("createdAt", "created_at", func(c agent.Conversation) any { return c.CreatedAt }),
		},
		Desc: desc,
	}
}

func scanConversation(rows *sql.Rows) (agent.Conversation, error) {
	var (
		c                    agent.Conversation
		siteID               sql.NullString
		mode                 string
		createdAt, updatedAt string
	)
	if err := rows.Scan(&c.ID, &siteID, &c.Title, &mode, &createdAt, &updatedAt); err != nil {
		return agent.Conversation{}, err
	}

	c.SiteID = optString(siteID)
	c.Mode = agent.Mode(mode)

	var err error
	if c.CreatedAt, err = parseTime(createdAt); err != nil {
		return agent.Conversation{}, err
	}
	if c.UpdatedAt, err = parseTime(updatedAt); err != nil {
		return agent.Conversation{}, err
	}
	return c, nil
}
