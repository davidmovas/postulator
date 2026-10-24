-- +goose Up
ALTER TABLE pages
    ADD COLUMN planned_trail TEXT NOT NULL DEFAULT '[]';

ALTER TABLE pages
    ADD COLUMN row_key TEXT NOT NULL DEFAULT '';

ALTER TABLE pages
    ADD COLUMN parent_source TEXT NOT NULL DEFAULT 'site' CHECK (parent_source IN ('site', 'workbook'));

-- +goose Down
ALTER TABLE pages
    DROP COLUMN parent_source;

ALTER TABLE pages
    DROP COLUMN row_key;

ALTER TABLE pages
    DROP COLUMN planned_trail;
