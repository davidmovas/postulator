-- +goose Up
DELETE FROM model_profiles
WHERE provider <> 'openai';

-- +goose Down
