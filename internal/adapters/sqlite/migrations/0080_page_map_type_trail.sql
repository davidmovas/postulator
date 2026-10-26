-- +goose Up
CREATE INDEX page_map_type_trail ON page_map (site_id, type_id, trail);

-- +goose Down
DROP INDEX page_map_type_trail;
