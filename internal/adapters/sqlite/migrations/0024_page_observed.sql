-- +goose Up
ALTER TABLE pages
    ADD COLUMN wp_link TEXT NOT NULL DEFAULT '';

ALTER TABLE pages
    ADD COLUMN wp_slug TEXT NOT NULL DEFAULT '';

ALTER TABLE pages
    ADD COLUMN wp_status TEXT NOT NULL DEFAULT '';

ALTER TABLE pages
    ADD COLUMN wp_title TEXT NOT NULL DEFAULT '';

ALTER TABLE pages
    ADD COLUMN wp_h1 TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE pages
    DROP COLUMN wp_h1;

ALTER TABLE pages
    DROP COLUMN wp_title;

ALTER TABLE pages
    DROP COLUMN wp_status;

ALTER TABLE pages
    DROP COLUMN wp_slug;

ALTER TABLE pages
    DROP COLUMN wp_link;
