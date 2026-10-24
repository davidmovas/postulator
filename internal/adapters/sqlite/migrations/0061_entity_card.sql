-- +goose Up
ALTER TABLE entities
    ADD COLUMN definition TEXT NOT NULL DEFAULT '';

ALTER TABLE entities
    ADD COLUMN facts TEXT NOT NULL DEFAULT '[]' CHECK (json_valid(facts) AND json_type(facts) = 'array');

ALTER TABLE entities
    ADD COLUMN notes TEXT NOT NULL DEFAULT '[]' CHECK (json_valid(notes) AND json_type(notes) = 'array');

-- +goose Down
ALTER TABLE entities
    DROP COLUMN notes;

ALTER TABLE entities
    DROP COLUMN facts;

ALTER TABLE entities
    DROP COLUMN definition;
