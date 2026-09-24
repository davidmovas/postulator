-- +goose Up
ALTER TABLE pages
    ADD COLUMN primary_keyword TEXT NOT NULL DEFAULT '';

ALTER TABLE pages
    ADD COLUMN keywords TEXT NOT NULL DEFAULT '[]';

-- +goose Down
ALTER TABLE pages
    DROP COLUMN keywords;

ALTER TABLE pages
    DROP COLUMN primary_keyword;
