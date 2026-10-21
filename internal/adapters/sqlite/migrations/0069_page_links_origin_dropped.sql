-- +goose Up
UPDATE page_links
SET ours = 1
WHERE origin = 'generated';

ALTER TABLE page_links
    DROP COLUMN origin;

-- +goose Down
ALTER TABLE page_links
    ADD COLUMN origin TEXT NOT NULL DEFAULT 'observed' CHECK (origin IN ('generated', 'observed'));

UPDATE page_links
SET origin = CASE WHEN ours = 1 THEN 'generated' ELSE 'observed' END;
