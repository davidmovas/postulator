-- +goose Up
CREATE TABLE page_map_sites (
    site_id TEXT PRIMARY KEY REFERENCES sites (id) ON DELETE CASCADE,
    generation INTEGER NOT NULL DEFAULT 1,
    built INTEGER NOT NULL DEFAULT 0
) STRICT;

INSERT INTO page_map_sites (site_id, generation, built)
SELECT id, 1, 0 FROM sites;

CREATE TABLE page_map (
    page_id TEXT PRIMARY KEY REFERENCES pages (id) ON DELETE CASCADE,
    site_id TEXT NOT NULL REFERENCES sites (id) ON DELETE CASCADE,
    parent_id TEXT NOT NULL DEFAULT '',
    path TEXT NOT NULL,
    depth INTEGER NOT NULL DEFAULT 0,
    trail TEXT NOT NULL DEFAULT '',
    children INTEGER NOT NULL DEFAULT 0,
    descendants INTEGER NOT NULL DEFAULT 0,
    twin_id TEXT NOT NULL DEFAULT '',
    hub_id TEXT NOT NULL DEFAULT '',
    node INTEGER NOT NULL CHECK (node IN (0, 1)),
    type_id TEXT NOT NULL DEFAULT '',
    twin_type_id TEXT NOT NULL DEFAULT '',
    title TEXT NOT NULL DEFAULT '',
    title_key TEXT NOT NULL DEFAULT '',
    path_key TEXT NOT NULL DEFAULT '',
    twin_title_key TEXT NOT NULL DEFAULT '',
    twin_path_key TEXT NOT NULL DEFAULT ''
) STRICT;

CREATE INDEX page_map_children ON page_map (site_id, parent_id, node, path, page_id);
CREATE INDEX page_map_trail ON page_map (site_id, trail);
CREATE INDEX page_map_nodes ON page_map (site_id, node, path, page_id);

-- +goose StatementBegin
CREATE TRIGGER page_map_site_added AFTER INSERT ON sites
BEGIN
    INSERT INTO page_map_sites (site_id, generation, built) VALUES (NEW.id, 1, 0);
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER page_map_page_added AFTER INSERT ON pages
BEGIN
    UPDATE page_map_sites SET generation = generation + 1 WHERE site_id = NEW.site_id;
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER page_map_page_removed AFTER DELETE ON pages
BEGIN
    UPDATE page_map_sites SET generation = generation + 1 WHERE site_id = OLD.site_id;
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER page_map_page_moved AFTER UPDATE OF parent_page_id, status, path, title, h1, wp_type, page_type_id, technical_reason ON pages
WHEN OLD.parent_page_id IS NOT NEW.parent_page_id OR OLD.status IS NOT NEW.status OR OLD.path IS NOT NEW.path
    OR OLD.title IS NOT NEW.title OR OLD.h1 IS NOT NEW.h1 OR OLD.wp_type IS NOT NEW.wp_type
    OR OLD.page_type_id IS NOT NEW.page_type_id OR OLD.technical_reason IS NOT NEW.technical_reason
BEGIN
    UPDATE page_map_sites SET generation = generation + 1 WHERE site_id = NEW.site_id;
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER page_map_name_added AFTER INSERT ON page_fields
WHEN NEW.field_key IN ('title', 'h1')
BEGIN
    UPDATE page_map_sites SET generation = generation + 1 WHERE site_id = (SELECT site_id FROM pages WHERE id = NEW.page_id);
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER page_map_name_moved AFTER UPDATE OF current_value ON page_fields
WHEN NEW.field_key IN ('title', 'h1') AND OLD.current_value IS NOT NEW.current_value
BEGIN
    UPDATE page_map_sites SET generation = generation + 1 WHERE site_id = (SELECT site_id FROM pages WHERE id = NEW.page_id);
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER page_map_name_removed AFTER DELETE ON page_fields
WHEN OLD.field_key IN ('title', 'h1')
BEGIN
    UPDATE page_map_sites SET generation = generation + 1 WHERE site_id = (SELECT site_id FROM pages WHERE id = OLD.page_id);
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER page_map_type_added AFTER INSERT ON page_types
BEGIN
    UPDATE page_map_sites SET generation = generation + 1 WHERE site_id = NEW.site_id;
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER page_map_type_moved AFTER UPDATE OF preset, site_id ON page_types
WHEN OLD.preset IS NOT NEW.preset OR OLD.site_id IS NOT NEW.site_id
BEGIN
    UPDATE page_map_sites SET generation = generation + 1 WHERE site_id IN (OLD.site_id, NEW.site_id);
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER page_map_type_removed AFTER DELETE ON page_types
BEGIN
    UPDATE page_map_sites SET generation = generation + 1 WHERE site_id = OLD.site_id;
END;
-- +goose StatementEnd

-- +goose Down
DROP TRIGGER page_map_type_removed;
DROP TRIGGER page_map_type_moved;
DROP TRIGGER page_map_type_added;
DROP TRIGGER page_map_name_removed;
DROP TRIGGER page_map_name_moved;
DROP TRIGGER page_map_name_added;
DROP TRIGGER page_map_page_moved;
DROP TRIGGER page_map_page_removed;
DROP TRIGGER page_map_page_added;
DROP TRIGGER page_map_site_added;
DROP INDEX page_map_nodes;
DROP INDEX page_map_trail;
DROP INDEX page_map_children;
DROP TABLE page_map;
DROP TABLE page_map_sites;
