-- +goose Up
CREATE TABLE template_overrides (
    id TEXT PRIMARY KEY,
    template_id TEXT NOT NULL REFERENCES templates (id) ON DELETE CASCADE,
    scope TEXT NOT NULL CHECK (scope IN ('site', 'page')),
    site_id TEXT REFERENCES sites (id) ON DELETE CASCADE,
    page_id TEXT REFERENCES pages (id) ON DELETE CASCADE,
    patch TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    CHECK ((scope = 'site' AND site_id IS NOT NULL AND page_id IS NULL) OR (scope = 'page' AND page_id IS NOT NULL AND site_id IS NULL))
) STRICT;

CREATE UNIQUE INDEX template_overrides_target ON template_overrides (template_id, scope, coalesce(site_id, page_id));
CREATE INDEX template_overrides_site ON template_overrides (site_id);
CREATE INDEX template_overrides_page ON template_overrides (page_id);

-- +goose Down
DROP INDEX template_overrides_page;
DROP INDEX template_overrides_site;
DROP INDEX template_overrides_target;
DROP TABLE template_overrides;
