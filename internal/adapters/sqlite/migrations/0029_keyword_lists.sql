-- +goose Up
ALTER TABLE entities
    ADD COLUMN keywords TEXT NOT NULL DEFAULT '[]';

UPDATE entities
SET keywords = (
    SELECT json_group_array(json_object('text', trim(item.value)) ORDER BY part.key, item.key)
    FROM json_each(json_array(json_array(entities.primary_keyword), json(entities.secondary_keywords))) AS part,
        json_each(part.value) AS item
    WHERE trim(item.value) <> ''
        AND (part.key = 0 OR lower(trim(item.value)) <> lower(trim(entities.primary_keyword)))
);

ALTER TABLE entities
    DROP COLUMN secondary_keywords;

ALTER TABLE entities
    DROP COLUMN primary_keyword;

UPDATE pages
SET keywords = (
    SELECT json_group_array(json_object('text', trim(item.value)) ORDER BY part.key, item.key)
    FROM json_each(json_array(json_array(pages.primary_keyword), json(pages.keywords))) AS part,
        json_each(part.value) AS item
    WHERE trim(item.value) <> ''
        AND (part.key = 0 OR lower(trim(item.value)) <> lower(trim(pages.primary_keyword)))
);

ALTER TABLE pages
    DROP COLUMN primary_keyword;

ALTER TABLE pages
    ADD COLUMN notes TEXT NOT NULL DEFAULT '[]';

-- +goose Down
ALTER TABLE pages
    DROP COLUMN notes;

ALTER TABLE pages
    ADD COLUMN primary_keyword TEXT NOT NULL DEFAULT '';

UPDATE pages
SET primary_keyword = coalesce(json_extract(keywords, '$[0].text'), ''),
    keywords = (
        SELECT json_group_array(json_extract(item.value, '$.text') ORDER BY item.key)
        FROM json_each(pages.keywords) AS item
        WHERE item.key > 0
    );

ALTER TABLE entities
    ADD COLUMN primary_keyword TEXT NOT NULL DEFAULT '';

ALTER TABLE entities
    ADD COLUMN secondary_keywords TEXT NOT NULL DEFAULT '[]';

UPDATE entities
SET primary_keyword = coalesce(json_extract(keywords, '$[0].text'), ''),
    secondary_keywords = (
        SELECT json_group_array(json_extract(item.value, '$.text') ORDER BY item.key)
        FROM json_each(entities.keywords) AS item
        WHERE item.key > 0
    );

ALTER TABLE entities
    DROP COLUMN keywords;
