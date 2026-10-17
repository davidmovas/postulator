-- +goose Up
ALTER TABLE pages
    ADD COLUMN page_type_id TEXT;

ALTER TABLE pages
    ADD COLUMN parent_reason TEXT NOT NULL DEFAULT 'none' CHECK (parent_reason IN ('jsonld_breadcrumb', 'aria_breadcrumb', 'wp_parent', 'url', 'workbook', 'none'));

ALTER TABLE pages
    ADD COLUMN technical_reason TEXT NOT NULL DEFAULT '' CHECK (technical_reason IN ('', 'front_page', 'posts_page', 'privacy', 'shop', 'cart', 'checkout', 'account', 'terms', 'shortcode', 'noindex', 'workbook', 'manual'));

ALTER TABLE pages
    ADD COLUMN read_only_reason TEXT NOT NULL DEFAULT '';

CREATE INDEX pages_page_type ON pages (page_type_id);

-- +goose Down
DROP INDEX pages_page_type;

ALTER TABLE pages
    DROP COLUMN read_only_reason;

ALTER TABLE pages
    DROP COLUMN technical_reason;

ALTER TABLE pages
    DROP COLUMN parent_reason;

ALTER TABLE pages
    DROP COLUMN page_type_id;
