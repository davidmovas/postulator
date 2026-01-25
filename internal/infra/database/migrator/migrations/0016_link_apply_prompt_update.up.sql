-- +goose Up
UPDATE prompts
SET system_prompt = REPLACE(
    system_prompt,
    '- If no suitable text exists in content: skip that link (do not force it)' || char(10),
    ''
),
updated_at = CURRENT_TIMESTAMP
WHERE is_builtin = 1 AND category = 'link_apply';

-- +goose Down
UPDATE prompts
SET system_prompt = REPLACE(
    system_prompt,
    'Language for anchor text selection:',
    '- If no suitable text exists in content: skip that link (do not force it)' || char(10) || char(10) || 'Language for anchor text selection:'
),
updated_at = CURRENT_TIMESTAMP
WHERE is_builtin = 1 AND category = 'link_apply';
