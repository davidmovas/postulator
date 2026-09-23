-- +goose Up
CREATE TABLE model_catalog (
    provider TEXT NOT NULL,
    model TEXT NOT NULL,
    context_tokens INTEGER NOT NULL,
    max_output_tokens INTEGER NOT NULL,
    input_usd_per_m REAL NOT NULL,
    output_usd_per_m REAL NOT NULL,
    rpm INTEGER NOT NULL,
    tpm INTEGER NOT NULL,
    supports_structured INTEGER NOT NULL,
    supports_images INTEGER NOT NULL,
    reasoning INTEGER NOT NULL,
    enabled INTEGER NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    PRIMARY KEY (provider, model)
) STRICT;

-- +goose Down
DROP TABLE model_catalog;
