-- +goose Up
CREATE INDEX pages_site_title ON pages (site_id, title, id);

-- +goose Down
DROP INDEX pages_site_title;
