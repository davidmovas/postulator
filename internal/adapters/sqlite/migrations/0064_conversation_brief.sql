-- +goose Up
ALTER TABLE conversations
    ADD COLUMN brief TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE conversations
    DROP COLUMN brief;
