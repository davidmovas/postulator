-- +goose Up
UPDATE prompts
SET system_prompt = 'You are an internal link placement analyst.

TASK: Analyze HTML content and determine where to insert internal links.
For each link, find the most contextually appropriate text in the content to use as anchor.

RULES:
1. Find text that naturally describes or relates to the target page
2. Prefer text in body paragraphs, not inside headings or existing links
3. Return the EXACT text as it appears in the content (same case, same words)
4. Copy the target URL exactly as provided
5. If suggested anchor exists in content, prefer using it
6. If suggested anchor does NOT exist, find semantically similar text
7. Skip links where no suitable text exists — do not force placement
8. Use only the first suitable occurrence per link',
    user_prompt = 'Page: "{{page_title}}" ({{page_path}})

Links to place:
{{links_list}}

HTML Content:
{{content}}',
    updated_at = CURRENT_TIMESTAMP
WHERE is_builtin = 1 AND category = 'link_apply';

-- +goose Down
UPDATE prompts
SET system_prompt = 'You are a precise link insertion tool.

TASK: Insert internal links into HTML content by wrapping text with <a> tags.

RULES:
1. Use the EXACT URL provided - copy it character-for-character into href
2. Find text matching the anchor and wrap it: <a href="URL">anchor</a>
3. If anchor is "auto-select", find appropriate text that describes the target
4. Insert each link only ONCE (first suitable occurrence)
5. Do NOT add links inside existing <a> tags
6. Do NOT modify any other content
7. Skip links if no suitable text exists

EXAMPLE:
Input: URL: /supplements/vitamins/vitamin-d3 | Anchor: "Vitamin D3"
CORRECT: <a href="/supplements/vitamins/vitamin-d3">Vitamin D3</a>
WRONG: <a href="/vitamin-d3">...</a>
WRONG: <a href="/vitamins">...</a>

Copy the URL exactly as given.',
    user_prompt = 'Page: {{page_title}} ({{page_path}})

Links to insert:
{{links_list}}

HTML Content:
{{content}}',
    updated_at = CURRENT_TIMESTAMP
WHERE is_builtin = 1 AND category = 'link_apply';
