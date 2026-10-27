-- +goose Up
ALTER TABLE page_renders
    ADD COLUMN h1 TEXT NOT NULL DEFAULT '';

ALTER TABLE page_renders
    ADD COLUMN h1_source TEXT NOT NULL DEFAULT '' CHECK (h1_source IN ('', 'content', 'theme', 'none'));

UPDATE page_renders
SET h1 = coalesce((
        SELECT trim(json_extract(heading.value, '$.text'))
        FROM json_each(page_renders.headings) AS heading
        WHERE json_extract(heading.value, '$.level') = 1
        ORDER BY heading.key
        LIMIT 1
    ), ''),
    h1_source = CASE
        WHEN EXISTS (
            SELECT 1 FROM json_each(page_renders.headings) AS heading WHERE json_extract(heading.value, '$.level') = 1
        ) THEN 'content'
        ELSE 'none'
    END
WHERE theme_h1 = 'absent' AND final_path = '';

-- +goose Down
ALTER TABLE page_renders
    DROP COLUMN h1_source;

ALTER TABLE page_renders
    DROP COLUMN h1;
