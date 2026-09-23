-- +goose Up
ALTER TABLE conversations
    ADD COLUMN title_settled INTEGER NOT NULL DEFAULT 0 CHECK (title_settled IN (0, 1));

UPDATE conversations SET title_settled = 1 WHERE title <> '';

-- +goose Down
ALTER TABLE conversations
    DROP COLUMN title_settled;
