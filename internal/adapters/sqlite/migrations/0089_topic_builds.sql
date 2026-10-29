-- +goose Up
CREATE TABLE topic_builds (
    site_id TEXT NOT NULL REFERENCES sites (id) ON DELETE CASCADE,
    subject TEXT NOT NULL CHECK (subject <> ''),
    built_at TEXT NOT NULL,
    PRIMARY KEY (site_id, subject)
) STRICT;

-- +goose Down
DROP TABLE topic_builds;
