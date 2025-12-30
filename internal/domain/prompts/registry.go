package prompts

import (
	"sync"

	"github.com/davidmovas/postulator/internal/domain/entities"
)

// ContextFieldRegistry holds all available context field definitions
type ContextFieldRegistry struct {
	mu     sync.RWMutex
	fields map[string]*entities.ContextFieldDefinition
}

var (
	globalRegistry *ContextFieldRegistry
	registryOnce   sync.Once
)

// GetRegistry returns the global context field registry
func GetRegistry() *ContextFieldRegistry {
	registryOnce.Do(func() {
		globalRegistry = newContextFieldRegistry()
	})
	return globalRegistry
}

func newContextFieldRegistry() *ContextFieldRegistry {
	r := &ContextFieldRegistry{
		fields: make(map[string]*entities.ContextFieldDefinition),
	}
	r.registerAllFields()
	return r
}

func (r *ContextFieldRegistry) registerAllFields() {
	// =========================================================================
	// POST_GEN Fields
	// =========================================================================
	r.register(&entities.ContextFieldDefinition{
		Key:          "title",
		Label:        "Topic Title",
		Description:  "Include the topic/post title",
		Type:         entities.ContextFieldTypeCheckbox,
		DefaultValue: "true",
		Required:     true,
		Categories:   []entities.PromptCategory{entities.PromptCategoryPostGen},
		Group:        "content",
	})
	r.register(&entities.ContextFieldDefinition{
		Key:          "siteName",
		Label:        "Site Name",
		Description:  "Include the website name",
		Type:         entities.ContextFieldTypeCheckbox,
		DefaultValue: "true",
		Categories: []entities.PromptCategory{
			entities.PromptCategoryPostGen,
			entities.PromptCategoryPageGen,
		},
		Group: "site",
	})
	r.register(&entities.ContextFieldDefinition{
		Key:          "siteUrl",
		Label:        "Site URL",
		Description:  "Include the website URL",
		Type:         entities.ContextFieldTypeCheckbox,
		DefaultValue: "false",
		Categories: []entities.PromptCategory{
			entities.PromptCategoryPostGen,
			entities.PromptCategoryPageGen,
		},
		Group: "site",
	})
	r.register(&entities.ContextFieldDefinition{
		Key:          "categories",
		Label:        "Categories",
		Description:  "Include post categories",
		Type:         entities.ContextFieldTypeCheckbox,
		DefaultValue: "true",
		Categories:   []entities.PromptCategory{entities.PromptCategoryPostGen},
		Group:        "content",
	})
	r.register(&entities.ContextFieldDefinition{
		Key:          "language",
		Label:        "Language",
		Description:  "Content language",
		Type:         entities.ContextFieldTypeInput,
		DefaultValue: "English",
		Categories: []entities.PromptCategory{
			entities.PromptCategoryPostGen,
			entities.PromptCategoryPageGen,
			entities.PromptCategoryLinkApply,
		},
		Group: "settings",
	})
	r.register(&entities.ContextFieldDefinition{
		Key:          "words",
		Label:        "Word Count",
		Description:  "Target word count (e.g., 800-1200)",
		Type:         entities.ContextFieldTypeInput,
		DefaultValue: "800-1200",
		Categories:   []entities.PromptCategory{entities.PromptCategoryPostGen},
		Group:        "settings",
	})

	// =========================================================================
	// PAGE_GEN Fields
	// =========================================================================
	r.register(&entities.ContextFieldDefinition{
		Key:          "pageTitle",
		Label:        "Page Title",
		Description:  "Include the page title",
		Type:         entities.ContextFieldTypeCheckbox,
		DefaultValue: "true",
		Required:     true,
		Categories:   []entities.PromptCategory{entities.PromptCategoryPageGen},
		Group:        "content",
	})
	r.register(&entities.ContextFieldDefinition{
		Key:          "path",
		Label:        "Page Path",
		Description:  "Include the URL path",
		Type:         entities.ContextFieldTypeCheckbox,
		DefaultValue: "true",
		Required:     true,
		Categories:   []entities.PromptCategory{entities.PromptCategoryPageGen},
		Group:        "content",
	})
	r.register(&entities.ContextFieldDefinition{
		Key:          "keywords",
		Label:        "Keywords",
		Description:  "Include target keywords",
		Type:         entities.ContextFieldTypeCheckbox,
		DefaultValue: "true",
		Categories:   []entities.PromptCategory{entities.PromptCategoryPageGen},
		Group:        "content",
	})
	r.register(&entities.ContextFieldDefinition{
		Key:          "hierarchy",
		Label:        "Site Hierarchy",
		Description:  "Include site structure context",
		Type:         entities.ContextFieldTypeCheckbox,
		DefaultValue: "true",
		Categories:   []entities.PromptCategory{entities.PromptCategoryPageGen},
		Group:        "content",
	})
	r.register(&entities.ContextFieldDefinition{
		Key:          "wordCount",
		Label:        "Word Count",
		Description:  "Target word count",
		Type:         entities.ContextFieldTypeInput,
		DefaultValue: "800-1200",
		Categories:   []entities.PromptCategory{entities.PromptCategoryPageGen},
		Group:        "settings",
	})
	r.register(&entities.ContextFieldDefinition{
		Key:          "writingStyle",
		Label:        "Writing Style",
		Description:  "Content writing style",
		Type:         entities.ContextFieldTypeSelect,
		DefaultValue: "professional",
		Options: []entities.SelectOption{
			{Value: "professional", Label: "Professional"},
			{Value: "casual", Label: "Casual"},
			{Value: "formal", Label: "Formal"},
			{Value: "friendly", Label: "Friendly"},
			{Value: "technical", Label: "Technical"},
		},
		Categories: []entities.PromptCategory{entities.PromptCategoryPageGen},
		Group:      "style",
	})
	r.register(&entities.ContextFieldDefinition{
		Key:          "contentTone",
		Label:        "Content Tone",
		Description:  "Content tone and voice",
		Type:         entities.ContextFieldTypeSelect,
		DefaultValue: "informative",
		Options: []entities.SelectOption{
			{Value: "informative", Label: "Informative"},
			{Value: "persuasive", Label: "Persuasive"},
			{Value: "educational", Label: "Educational"},
			{Value: "engaging", Label: "Engaging"},
			{Value: "authoritative", Label: "Authoritative"},
		},
		Categories: []entities.PromptCategory{entities.PromptCategoryPageGen},
		Group:      "style",
	})
	// NOTE: customInstructions is a runtime-only field, not stored in prompt config
	// It's passed at usage time via overrides
	r.register(&entities.ContextFieldDefinition{
		Key:          "internalLinks",
		Label:        "Internal Links",
		Description:  "Include internal links to embed",
		Type:         entities.ContextFieldTypeCheckbox,
		DefaultValue: "true",
		Categories:   []entities.PromptCategory{entities.PromptCategoryPageGen},
		Group:        "content",
	})

	// =========================================================================
	// LINK_SUGGEST Fields
	// =========================================================================
	// NOTE: nodesInfo was removed - hierarchyTree now includes all node info
	// (ID, title, path, keywords, link counts)
	r.register(&entities.ContextFieldDefinition{
		Key:          "hierarchyTree",
		Label:        "Hierarchy Tree",
		Description:  "Include visual site hierarchy with node details",
		Type:         entities.ContextFieldTypeCheckbox,
		DefaultValue: "true",
		Required:     true,
		Categories:   []entities.PromptCategory{entities.PromptCategoryLinkSuggest},
		Group:        "content",
	})
	r.register(&entities.ContextFieldDefinition{
		Key:          "constraints",
		Label:        "Link Constraints",
		Description:  "Include max link limits (passed via runtime data)",
		Type:         entities.ContextFieldTypeCheckbox,
		DefaultValue: "true",
		Categories:   []entities.PromptCategory{entities.PromptCategoryLinkSuggest},
		Group:        "content",
	})
	// NOTE: maxIncoming/maxOutgoing are now included in the constraints placeholder
	// and passed via runtime data, not as separate registry fields
	// NOTE: feedback is a runtime-only field, not stored in prompt config
	// It's passed at usage time via overrides

	// =========================================================================
	// LINK_APPLY Fields
	// =========================================================================
	r.register(&entities.ContextFieldDefinition{
		Key:          "applyPageTitle",
		Label:        "Page Title",
		Description:  "Include the page title",
		Type:         entities.ContextFieldTypeCheckbox,
		DefaultValue: "true",
		Required:     true,
		Categories:   []entities.PromptCategory{entities.PromptCategoryLinkApply},
		Group:        "content",
	})
	r.register(&entities.ContextFieldDefinition{
		Key:          "pagePath",
		Label:        "Page Path",
		Description:  "Include the page URL path",
		Type:         entities.ContextFieldTypeCheckbox,
		DefaultValue: "true",
		Required:     true,
		Categories:   []entities.PromptCategory{entities.PromptCategoryLinkApply},
		Group:        "content",
	})
	r.register(&entities.ContextFieldDefinition{
		Key:          "content",
		Label:        "Page Content",
		Description:  "Include the page HTML content",
		Type:         entities.ContextFieldTypeCheckbox,
		DefaultValue: "true",
		Required:     true,
		Categories:   []entities.PromptCategory{entities.PromptCategoryLinkApply},
		Group:        "content",
	})
	r.register(&entities.ContextFieldDefinition{
		Key:          "linksList",
		Label:        "Links List",
		Description:  "Include the list of links to insert",
		Type:         entities.ContextFieldTypeCheckbox,
		DefaultValue: "true",
		Required:     true,
		Categories:   []entities.PromptCategory{entities.PromptCategoryLinkApply},
		Group:        "content",
	})
}

// register adds a field definition to the registry
func (r *ContextFieldRegistry) register(def *entities.ContextFieldDefinition) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.fields[def.Key] = def
}

// Get returns a field definition by key
func (r *ContextFieldRegistry) Get(key string) *entities.ContextFieldDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.fields[key]
}

// GetByCategory returns all field definitions for a specific category
func (r *ContextFieldRegistry) GetByCategory(category entities.PromptCategory) []*entities.ContextFieldDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*entities.ContextFieldDefinition
	for _, def := range r.fields {
		for _, cat := range def.Categories {
			if cat == category {
				result = append(result, def)
				break
			}
		}
	}
	return result
}

// GetAll returns all field definitions
func (r *ContextFieldRegistry) GetAll() []*entities.ContextFieldDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*entities.ContextFieldDefinition, 0, len(r.fields))
	for _, def := range r.fields {
		result = append(result, def)
	}
	return result
}

// GetDefaultContextConfig returns the default context config for a category
func (r *ContextFieldRegistry) GetDefaultContextConfig(category entities.PromptCategory) entities.ContextConfig {
	fields := r.GetByCategory(category)
	config := make(entities.ContextConfig)

	for _, field := range fields {
		enabled := field.Required || field.DefaultValue == "true"
		config[field.Key] = entities.ContextFieldValue{
			Enabled: enabled,
			Value:   field.DefaultValue,
		}
	}

	return config
}
