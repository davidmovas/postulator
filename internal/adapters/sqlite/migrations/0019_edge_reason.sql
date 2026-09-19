-- +goose Up
ALTER TABLE edges ADD COLUMN reason TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE edges DROP COLUMN reason;
