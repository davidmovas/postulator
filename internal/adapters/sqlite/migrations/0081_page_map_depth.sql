-- +goose Up
CREATE INDEX page_map_depth ON page_map (site_id, node, depth, path, page_id);

-- +goose Down
DROP INDEX page_map_depth;
