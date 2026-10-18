-- +goose Up
ALTER TABLE pages
    ADD COLUMN parent_method TEXT NOT NULL DEFAULT '' CHECK (parent_method IN ('', 'href', 'redirect', 'name'));

UPDATE pages
SET parent_method = 'href'
WHERE parent_reason IN ('jsonld_breadcrumb', 'aria_breadcrumb');

-- +goose Down
ALTER TABLE pages
    DROP COLUMN parent_method;
