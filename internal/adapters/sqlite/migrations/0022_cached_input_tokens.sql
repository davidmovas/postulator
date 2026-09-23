-- +goose Up
ALTER TABLE llm_calls
    ADD COLUMN cached_input_tokens INTEGER NOT NULL DEFAULT 0;

ALTER TABLE model_catalog
    ADD COLUMN cached_input_usd_per_m REAL NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE model_catalog
    DROP COLUMN cached_input_usd_per_m;

ALTER TABLE llm_calls
    DROP COLUMN cached_input_tokens;
