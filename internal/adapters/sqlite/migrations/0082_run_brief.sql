-- +goose Up
ALTER TABLE runs ADD COLUMN brief TEXT NOT NULL DEFAULT '';
ALTER TABLE run_items ADD COLUMN directive TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE run_items DROP COLUMN directive;
ALTER TABLE runs DROP COLUMN brief;
