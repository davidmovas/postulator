-- +goose Up
UPDATE pages
SET technical_reason = ''
WHERE technical_reason = 'front_page';

-- +goose Down
