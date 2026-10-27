-- +goose NO TRANSACTION
-- +goose Up
PRAGMA foreign_keys = OFF;

BEGIN;

ALTER TABLE pages
    ADD COLUMN page_type_reason_widened TEXT NOT NULL DEFAULT '' CHECK (page_type_reason_widened IN (
        '', 'product', 'store_category', 'shop', 'technical', 'listing', 'default', 'kind', 'user', 'ai', 'import', 'rule', 'unmatched'
    ));

UPDATE pages SET page_type_reason_widened = page_type_reason;

ALTER TABLE pages
    DROP COLUMN page_type_reason;

ALTER TABLE pages
    RENAME COLUMN page_type_reason_widened TO page_type_reason;

ALTER TABLE pages
    ADD COLUMN page_type_rule_id TEXT NOT NULL DEFAULT '';

COMMIT;

PRAGMA foreign_keys = ON;

-- +goose Down
PRAGMA foreign_keys = OFF;

BEGIN;

ALTER TABLE pages
    DROP COLUMN page_type_rule_id;

ALTER TABLE pages
    ADD COLUMN page_type_reason_narrowed TEXT NOT NULL DEFAULT '' CHECK (page_type_reason_narrowed IN (
        '', 'product', 'store_category', 'shop', 'technical', 'listing', 'default', 'kind', 'user', 'ai', 'import'
    ));

UPDATE pages
SET page_type_reason_narrowed = CASE WHEN page_type_reason IN ('rule', 'unmatched') THEN 'default' ELSE page_type_reason END;

ALTER TABLE pages
    DROP COLUMN page_type_reason;

ALTER TABLE pages
    RENAME COLUMN page_type_reason_narrowed TO page_type_reason;

COMMIT;

PRAGMA foreign_keys = ON;
