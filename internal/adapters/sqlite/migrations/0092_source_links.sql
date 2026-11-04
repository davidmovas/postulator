-- +goose Up
UPDATE field_scopes
SET link_policy = json_remove(link_policy, '$.forbidExternal')
WHERE json_type(link_policy, '$.forbidExternal') IS NOT NULL;

UPDATE pending_actions
SET args = json_remove(args, '$.links.forbidExternal')
WHERE tool = 'pagetypes'
    AND status IN ('pending', 'approved')
    AND json_valid(args)
    AND json_type(args, '$.links.forbidExternal') IS NOT NULL;

UPDATE agent_plans
SET steps = (
    SELECT json_group_array(
        CASE
            WHEN json_extract(step.value, '$.tool') = 'pagetypes'
                THEN json_remove(step.value, '$.args.links.forbidExternal')
            ELSE json(step.value)
        END
        ORDER BY step.key)
    FROM json_each(agent_plans.steps) AS step
)
WHERE status IN ('proposed', 'running')
    AND json_valid(steps)
    AND instr(steps, '"forbidExternal"') > 0;

UPDATE templates
SET spec = json_set(spec, '$.sourceLinks', json('{"max":3,"perDomain":1,"min":0}'))
WHERE json_type(spec, '$.sourceLinks') IS NULL
    AND EXISTS (
        SELECT 1
        FROM json_each(templates.spec, '$.recipe') AS step
        WHERE json_extract(step.value, '$.name') = 'research'
            AND json_extract(step.value, '$.enabled') = 1
    );

-- +goose Down
UPDATE field_scopes
SET link_policy = json_set(link_policy, '$.forbidExternal',
    json(iif(json_extract(link_policy, '$.links') = 1, 'true', 'false')));

UPDATE templates
SET spec = json_remove(spec, '$.sourceLinks')
WHERE json_type(spec, '$.sourceLinks') IS NOT NULL;

UPDATE template_overrides
SET patch = json_remove(patch, '$.sourceLinks')
WHERE json_valid(patch) AND json_type(patch, '$.sourceLinks') IS NOT NULL;
