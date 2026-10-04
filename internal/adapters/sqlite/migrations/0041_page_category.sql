-- +goose Up
ALTER TABLE pages
    ADD COLUMN category_id TEXT NOT NULL DEFAULT '';

CREATE INDEX pages_category ON pages (category_id);

-- +goose Down
DROP INDEX pages_category;

ALTER TABLE pages
    DROP COLUMN category_id;
