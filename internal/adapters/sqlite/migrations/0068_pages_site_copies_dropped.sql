-- +goose Up
INSERT INTO page_fields (page_id, field_key, current_value, current_hash, base_value, base_hash, observed_at)
SELECT id, 'title', wp_title, lower(hex(sha256(wp_title))), wp_title, lower(hex(sha256(wp_title))), coalesce(last_synced_at, updated_at)
FROM pages
WHERE wp_title <> ''
ON CONFLICT (page_id, field_key) DO NOTHING;

INSERT INTO page_fields (page_id, field_key, current_value, current_hash, base_value, base_hash, observed_at)
SELECT id, 'h1', wp_h1, lower(hex(sha256(wp_h1))), wp_h1, lower(hex(sha256(wp_h1))), coalesce(last_synced_at, updated_at)
FROM pages
WHERE wp_h1 <> '' AND wp_type <> 'product_cat'
ON CONFLICT (page_id, field_key) DO NOTHING;

ALTER TABLE pages
    DROP COLUMN drift;

ALTER TABLE pages
    DROP COLUMN content_hash;

ALTER TABLE pages
    DROP COLUMN wp_link;

ALTER TABLE pages
    DROP COLUMN wp_slug;

ALTER TABLE pages
    DROP COLUMN wp_status;

ALTER TABLE pages
    DROP COLUMN wp_title;

ALTER TABLE pages
    DROP COLUMN wp_h1;

ALTER TABLE pages
    DROP COLUMN canonical;

-- +goose Down
ALTER TABLE pages
    ADD COLUMN canonical TEXT NOT NULL DEFAULT '';

ALTER TABLE pages
    ADD COLUMN content_hash TEXT NOT NULL DEFAULT '';

ALTER TABLE pages
    ADD COLUMN drift INTEGER NOT NULL DEFAULT 0 CHECK (drift IN (0, 1));

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

UPDATE pages
SET wp_title = coalesce((SELECT current_value FROM page_fields WHERE page_id = pages.id AND field_key = 'title'), ''),
    wp_h1 = coalesce((SELECT current_value FROM page_fields WHERE page_id = pages.id AND field_key = 'h1'), ''),
    wp_status = coalesce((SELECT status FROM page_identity WHERE page_id = pages.id), ''),
    wp_link = coalesce((SELECT url FROM page_identity WHERE page_id = pages.id), '');

UPDATE pages
SET content_hash = (SELECT hash FROM page_bodies WHERE page_id = pages.id AND side = 'base'),
    drift = (
        SELECT CASE WHEN current.hash <> base.hash THEN 1 ELSE 0 END
        FROM page_bodies AS current
        JOIN page_bodies AS base ON base.page_id = current.page_id AND base.side = 'base'
        WHERE current.page_id = pages.id AND current.side = 'current'
    )
WHERE EXISTS (SELECT 1 FROM page_bodies AS current
        JOIN page_bodies AS base ON base.page_id = current.page_id AND base.side = 'base'
        WHERE current.page_id = pages.id AND current.side = 'current')
    AND (EXISTS (SELECT 1 FROM page_fields WHERE page_id = pages.id AND written_run_id IS NOT NULL)
        OR EXISTS (SELECT 1 FROM page_blocks_written WHERE page_id = pages.id));
