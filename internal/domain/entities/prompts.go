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

// ContextFieldType defines the UI control type for a context field
type ContextFieldType string

const (
	ContextFieldTypeCheckbox ContextFieldType = "checkbox"
	ContextFieldTypeSelect   ContextFieldType = "select"
	ContextFieldTypeInput    ContextFieldType = "input"
	ContextFieldTypeNumber   ContextFieldType = "number"
	ContextFieldTypeTextarea ContextFieldType = "textarea"
)

// SelectOption represents an option for select-type context fields
type SelectOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// ContextFieldValue represents the configuration for a single context field
type ContextFieldValue struct {
	Enabled bool   `json:"enabled"`
	Value   string `json:"value,omitempty"`
}

// ContextConfig stores the configuration for context fields
type ContextConfig map[string]ContextFieldValue

// ContextFieldDefinition defines metadata for a context field
type ContextFieldDefinition struct {
	Key          string           `json:"key"`
	Label        string           `json:"label"`
	Description  string           `json:"description"`
	Type         ContextFieldType `json:"type"`
	Options      []SelectOption   `json:"options,omitempty"`
	DefaultValue string           `json:"defaultValue"`
	Required     bool             `json:"required"`
	Categories   []PromptCategory `json:"categories"`
	Group        string           `json:"group,omitempty"`
}

// Prompt represents an AI prompt with instructions and context configuration
type Prompt struct {
	ID            int64
	Name          string
	Category      PromptCategory
	IsBuiltin     bool
	Instructions  string        // User's instructions (becomes System Prompt for AI)
	ContextConfig ContextConfig // JSON config for context fields
	Version       int           // 2 = new format, 1 = legacy format
	// Legacy fields (kept for backward compatibility and migration)
	SystemPrompt string
	UserPrompt   string
	Placeholders []string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// IsV2 returns true if this prompt uses the new v2 format
func (p *Prompt) IsV2() bool {
	return p.Version >= 2
}
