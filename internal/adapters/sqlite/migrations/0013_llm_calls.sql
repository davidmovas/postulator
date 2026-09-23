-- +goose Up
CREATE TABLE llm_calls (
    id TEXT PRIMARY KEY,
    run_id TEXT NOT NULL,
    item_id TEXT NOT NULL,
    step TEXT NOT NULL,
    conversation_id TEXT NOT NULL,
    provider TEXT NOT NULL,
    model TEXT NOT NULL,
    input_tokens INTEGER NOT NULL,
    output_tokens INTEGER NOT NULL,
    usd REAL NOT NULL,
    latency_ms INTEGER NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('ok', 'error')),
    error_code TEXT NOT NULL,
    created_at TEXT NOT NULL
) STRICT;

CREATE INDEX llm_calls_run ON llm_calls (run_id, created_at, id);
CREATE INDEX llm_calls_conversation ON llm_calls (conversation_id, created_at, id);

-- +goose Down
DROP INDEX llm_calls_conversation;
DROP INDEX llm_calls_run;
DROP TABLE llm_calls;
