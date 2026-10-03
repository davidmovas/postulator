-- +goose Up
ALTER TABLE model_catalog
    ADD COLUMN cache_write_usd_per_m REAL NOT NULL DEFAULT 0
    CHECK (cache_write_usd_per_m >= 0);

ALTER TABLE model_catalog
    ADD COLUMN flex_input_usd_per_m REAL NOT NULL DEFAULT 0
    CHECK (flex_input_usd_per_m >= 0);

ALTER TABLE model_catalog
    ADD COLUMN flex_cached_input_usd_per_m REAL NOT NULL DEFAULT 0
    CHECK (flex_cached_input_usd_per_m >= 0);

ALTER TABLE model_catalog
    ADD COLUMN flex_cache_write_usd_per_m REAL NOT NULL DEFAULT 0
    CHECK (flex_cache_write_usd_per_m >= 0);

ALTER TABLE model_catalog
    ADD COLUMN flex_output_usd_per_m REAL NOT NULL DEFAULT 0
    CHECK (flex_output_usd_per_m >= 0);

-- +goose Down
ALTER TABLE model_catalog
    DROP COLUMN flex_output_usd_per_m;

ALTER TABLE model_catalog
    DROP COLUMN flex_cache_write_usd_per_m;

ALTER TABLE model_catalog
    DROP COLUMN flex_cached_input_usd_per_m;

ALTER TABLE model_catalog
    DROP COLUMN flex_input_usd_per_m;

ALTER TABLE model_catalog
    DROP COLUMN cache_write_usd_per_m;
