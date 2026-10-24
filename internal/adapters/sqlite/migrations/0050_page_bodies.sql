-- +goose Up
CREATE TABLE page_bodies (
    page_id TEXT NOT NULL REFERENCES pages (id) ON DELETE CASCADE,
    side TEXT NOT NULL CHECK (side IN ('current', 'base')),
    raw TEXT NOT NULL,
    hash TEXT NOT NULL,
    kind TEXT NOT NULL CHECK (kind IN ('classic', 'blocks')),
    observed_at TEXT NOT NULL,
    PRIMARY KEY (page_id, side)
) STRICT;

CREATE TABLE page_blocks_written (
    page_id TEXT NOT NULL REFERENCES pages (id) ON DELETE CASCADE,
    block_hash TEXT NOT NULL,
    run_id TEXT NOT NULL,
    written_at TEXT NOT NULL,
    PRIMARY KEY (page_id, block_hash)
) STRICT;

-- +goose Down
DROP TABLE page_blocks_written;
DROP TABLE page_bodies;
