-- +goose Up
ALTER TABLE page_renders
    ADD COLUMN theme_h1 TEXT NOT NULL DEFAULT '' CHECK (theme_h1 IN ('', 'printed', 'absent'));

-- +goose Down
ALTER TABLE page_renders
    DROP COLUMN theme_h1;
