package entities

import "time"

type PromptCategory string

const (
	PromptCategoryPostGen     PromptCategory = "post_gen"
	PromptCategoryPageGen     PromptCategory = "page_gen"
	PromptCategoryLinkSuggest PromptCategory = "link_suggest"
	PromptCategoryLinkApply   PromptCategory = "link_apply"
	PromptCategorySitemapGen  PromptCategory = "sitemap_gen"
)

type Prompt struct {
	ID           int64
	Name         string
	Category     PromptCategory
	IsBuiltin    bool
	SystemPrompt string
	UserPrompt   string
	Placeholders []string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
