-- +goose Up
CREATE TABLE conversations (
    id TEXT PRIMARY KEY,
    site_id TEXT REFERENCES sites (id) ON DELETE CASCADE,
    title TEXT NOT NULL DEFAULT '',
    mode TEXT NOT NULL CHECK (mode IN ('confirm', 'autonomous')),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
) STRICT;

CREATE INDEX conversations_created_at ON conversations (created_at, id);
CREATE INDEX conversations_site ON conversations (site_id, created_at, id);

CREATE TABLE messages (
    id TEXT PRIMARY KEY,
    conversation_id TEXT NOT NULL REFERENCES conversations (id) ON DELETE CASCADE,
    seq INTEGER NOT NULL,
    role TEXT NOT NULL CHECK (role IN ('user', 'assistant', 'tool')),
    text TEXT NOT NULL DEFAULT '',
    tool TEXT NOT NULL DEFAULT '',
    call_id TEXT NOT NULL DEFAULT '',
    payload TEXT NOT NULL DEFAULT '{}',
    created_at TEXT NOT NULL,
    UNIQUE (conversation_id, seq)
) STRICT;

CREATE INDEX messages_conversation_seq ON messages (conversation_id, seq);

CREATE TABLE conversation_histories (
    conversation_id TEXT PRIMARY KEY REFERENCES conversations (id) ON DELETE CASCADE,
    history TEXT NOT NULL,
    version INTEGER NOT NULL,
    updated_at TEXT NOT NULL
) STRICT;

CREATE TABLE pending_actions (
    id TEXT PRIMARY KEY,
    conversation_id TEXT NOT NULL REFERENCES conversations (id) ON DELETE CASCADE,
    tool TEXT NOT NULL,
    args TEXT NOT NULL DEFAULT '{}',
    summary TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('pending', 'approved', 'rejected', 'executed', 'failed')),
    result TEXT NOT NULL DEFAULT '',
    error TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
) STRICT;

CREATE INDEX pending_actions_conversation ON pending_actions (conversation_id, created_at, id);
CREATE INDEX pending_actions_status ON pending_actions (status, created_at, id);

CREATE TABLE tool_calls (
    id TEXT PRIMARY KEY,
    conversation_id TEXT NOT NULL REFERENCES conversations (id) ON DELETE CASCADE,
    call_id TEXT NOT NULL DEFAULT '',
    tool TEXT NOT NULL,
    args TEXT NOT NULL DEFAULT '{}',
    status TEXT NOT NULL CHECK (status IN ('ok', 'denied', 'error')),
    duration_ms INTEGER NOT NULL DEFAULT 0,
    error TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL
) STRICT;

CREATE INDEX tool_calls_conversation ON tool_calls (conversation_id, created_at, id);

-- +goose Down
DROP INDEX tool_calls_conversation;
DROP TABLE tool_calls;
DROP INDEX pending_actions_status;
DROP INDEX pending_actions_conversation;
DROP TABLE pending_actions;
DROP TABLE conversation_histories;
DROP INDEX messages_conversation_seq;
DROP TABLE messages;
DROP INDEX conversations_site;
DROP INDEX conversations_created_at;
DROP TABLE conversations;
