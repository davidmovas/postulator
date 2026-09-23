-- +goose Up
ALTER TABLE model_catalog
    ADD COLUMN reasoning_effort TEXT NOT NULL DEFAULT ''
    CHECK (reasoning_effort IN ('', 'none', 'low', 'medium', 'high', 'xhigh'));

-- +goose Down
ALTER TABLE model_catalog
    DROP COLUMN reasoning_effort;
