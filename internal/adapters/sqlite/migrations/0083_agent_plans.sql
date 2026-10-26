-- +goose Up
CREATE TABLE agent_plans (
    id TEXT PRIMARY KEY,
    conversation_id TEXT NOT NULL REFERENCES conversations (id) ON DELETE CASCADE,
    site_id TEXT NOT NULL DEFAULT '',
    title TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('proposed', 'running', 'done', 'declined')),
    steps TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
) STRICT;

CREATE INDEX agent_plans_conversation ON agent_plans (conversation_id, created_at, id);
CREATE INDEX agent_plans_status ON agent_plans (status, updated_at, id);

-- +goose Down
DROP INDEX agent_plans_status;
DROP INDEX agent_plans_conversation;
DROP TABLE agent_plans;
