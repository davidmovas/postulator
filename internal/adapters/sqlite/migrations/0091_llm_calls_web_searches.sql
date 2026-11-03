-- +goose Up
ALTER TABLE llm_calls
    ADD COLUMN web_searches INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE llm_calls
    DROP COLUMN web_searches;
