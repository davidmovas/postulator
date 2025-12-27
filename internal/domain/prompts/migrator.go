package prompts

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/davidmovas/postulator/internal/domain/entities"
	"github.com/davidmovas/postulator/pkg/logger"
)

// Migrator handles migration of prompts from v1 to v2 format
type Migrator struct {
	repo     Repository
	registry *ContextFieldRegistry
	logger   *logger.Logger
}

// NewMigrator creates a new prompt migrator
func NewMigrator(repo Repository, logger *logger.Logger) *Migrator {
	return &Migrator{
		repo:     repo,
		registry: GetRegistry(),
		logger:   logger.WithScope("migrator").WithScope("prompts"),
	}
}

// MigrateAllToV2 migrates all v1 prompts to v2 format
func (m *Migrator) MigrateAllToV2(ctx context.Context) error {
	prompts, err := m.repo.GetAll(ctx)
	if err != nil {
		return err
	}

	migrated := 0
	for _, prompt := range prompts {
		if prompt.Version >= 2 {
			continue // Already v2
		}

		if err := m.migratePrompt(ctx, prompt); err != nil {
			m.logger.Warnf("Failed to migrate prompt %d (%s): %v", prompt.ID, prompt.Name, err)
			continue
		}
		migrated++
	}

	if migrated > 0 {
		m.logger.Infof("Migrated %d prompts to v2 format", migrated)
	}

	return nil
}

// migratePrompt converts a single v1 prompt to v2 format
func (m *Migrator) migratePrompt(ctx context.Context, prompt *entities.Prompt) error {
	// Extract instructions from system prompt
	prompt.Instructions = m.extractInstructions(prompt.SystemPrompt)

	// Build context config from placeholders and category
	prompt.ContextConfig = m.buildContextConfig(prompt.Category, prompt.Placeholders)

	// Mark as v2
	prompt.Version = 2

	return m.repo.Update(ctx, prompt)
}

// extractInstructions extracts the core instructions from a system prompt
// This removes placeholder references like {{language}} and cleans up the text
func (m *Migrator) extractInstructions(systemPrompt string) string {
	result := systemPrompt

	// Remove placeholder patterns like {{placeholder_name}}
	for {
		start := strings.Index(result, "{{")
		if start == -1 {
			break
		}
		end := strings.Index(result[start:], "}}")
		if end == -1 {
			break
		}
		end += start + 2
		result = result[:start] + result[end:]
	}

	// Clean up multiple spaces and newlines
	for strings.Contains(result, "  ") {
		result = strings.ReplaceAll(result, "  ", " ")
	}
	for strings.Contains(result, "\n\n\n") {
		result = strings.ReplaceAll(result, "\n\n\n", "\n\n")
	}

	// Remove lines that are now empty or just contain punctuation/labels
	lines := strings.Split(result, "\n")
	var cleanLines []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Skip lines that are just "- Language:" or similar after placeholder removal
		if trimmed == "" || trimmed == "-" || strings.HasSuffix(trimmed, ":") && len(trimmed) < 20 {
			continue
		}
		cleanLines = append(cleanLines, line)
	}

	return strings.TrimSpace(strings.Join(cleanLines, "\n"))
}

// buildContextConfig creates a context config based on category and placeholders
func (m *Migrator) buildContextConfig(category entities.PromptCategory, placeholders []string) entities.ContextConfig {
	// Get default config for this category
	config := m.registry.GetDefaultContextConfig(category)

	// Map legacy placeholder names to new field keys
	placeholderMap := map[string]string{
		"title":               "title",
		"siteName":            "siteName",
		"siteUrl":             "siteUrl",
		"categories":          "categories",
		"language":            "language",
		"words":               "words",
		"path":                "path",
		"keywords":            "keywords",
		"hierarchy":           "hierarchy",
		"word_count":          "wordCount",
		"writing_style":       "writingStyle",
		"content_tone":        "contentTone",
		"custom_instructions": "customInstructions",
		"internal_links":      "internalLinks",
		"nodes_info":          "nodesInfo",
		"constraints":         "constraints",
		"feedback":            "feedback",
		"maxIncoming":         "maxIncoming",
		"maxOutgoing":         "maxOutgoing",
		"page_title":          "applyPageTitle",
		"page_path":           "pagePath",
		"content":             "content",
		"links_list":          "linksList",
	}

	// Enable fields based on placeholders used
	placeholderSet := make(map[string]bool)
	for _, p := range placeholders {
		placeholderSet[p] = true
	}

	for oldKey, newKey := range placeholderMap {
		if placeholderSet[oldKey] {
			if fieldConfig, exists := config[newKey]; exists {
				fieldConfig.Enabled = true
				config[newKey] = fieldConfig
			}
		}
	}

	return config
}

// MarshalContextConfig serializes context config to JSON string
func MarshalContextConfig(config entities.ContextConfig) (string, error) {
	if config == nil {
		return "", nil
	}
	bytes, err := json.Marshal(config)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}
