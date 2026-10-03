-- +goose Up
ALTER TABLE llm_calls
    ADD COLUMN reasoning_tokens INTEGER NOT NULL DEFAULT 0;

ALTER TABLE llm_calls
    ADD COLUMN cache_write_tokens INTEGER NOT NULL DEFAULT 0;

ALTER TABLE llm_calls
    ADD COLUMN service_tier TEXT NOT NULL DEFAULT '';

CREATE INDEX llm_calls_created ON llm_calls (created_at, id);

-- +goose Down
DROP INDEX llm_calls_created;

ALTER TABLE llm_calls
    DROP COLUMN service_tier;

ALTER TABLE llm_calls
    DROP COLUMN cache_write_tokens;

ALTER TABLE llm_calls
    DROP COLUMN reasoning_tokens;
