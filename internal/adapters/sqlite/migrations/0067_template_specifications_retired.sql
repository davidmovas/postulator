-- +goose Up
UPDATE templates
SET spec = json_remove(spec, '$.product.specifications')
WHERE json_type(spec, '$.product.specifications') IS NOT NULL;

UPDATE template_overrides
SET patch = json_remove(patch, '$.product.specifications')
WHERE json_type(patch, '$.product.specifications') IS NOT NULL;

-- +goose Down
SELECT 1;
