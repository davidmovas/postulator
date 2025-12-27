package prompts

import (
	"strings"

	"github.com/davidmovas/postulator/internal/domain/entities"
)

// PromptBuilder builds final prompts from instructions and context
type PromptBuilder struct {
	registry *ContextFieldRegistry
}

// BuildRequest contains the data needed to build a prompt
type BuildRequest struct {
	Prompt      *entities.Prompt
	RuntimeData map[string]string // Actual data values to inject
	Overrides   entities.ContextConfig
}

// BuildResult contains the built system and user prompts
type BuildResult struct {
	SystemPrompt string
	UserPrompt   string
}

// NewPromptBuilder creates a new PromptBuilder
func NewPromptBuilder() *PromptBuilder {
	return &PromptBuilder{
		registry: GetRegistry(),
	}
}

// Build creates system and user prompts from a prompt entity with runtime data
func (b *PromptBuilder) Build(req *BuildRequest) *BuildResult {
	prompt := req.Prompt

	// Handle legacy prompts (version 1)
	if !prompt.IsV2() {
		return b.buildLegacy(prompt, req.RuntimeData)
	}

	// Merge prompt's context config with overrides
	effectiveConfig := b.mergeConfigs(prompt.ContextConfig, req.Overrides)

	// Build system prompt from instructions
	systemPrompt := prompt.Instructions

	// Build user prompt from enabled context fields
	userPrompt := b.buildUserPrompt(prompt.Category, effectiveConfig, req.RuntimeData)

	return &BuildResult{
		SystemPrompt: systemPrompt,
		UserPrompt:   userPrompt,
	}
}

// Runtime-only fields that are not in registry but can be passed via RuntimeData
// Supports both snake_case and camelCase keys for compatibility
var runtimeOnlyFields = map[string]string{
	"customInstructions":  "Additional Instructions",
	"custom_instructions": "Additional Instructions",
	"feedback":            "Feedback",
}

// buildUserPrompt builds the user prompt from enabled context fields
func (b *PromptBuilder) buildUserPrompt(
	category entities.PromptCategory,
	config entities.ContextConfig,
	data map[string]string,
) string {
	fields := b.registry.GetByCategory(category)

	// Group fields for organized output
	groups := map[string][]*entities.ContextFieldDefinition{
		"content":  {},
		"site":     {},
		"settings": {},
		"style":    {},
		"advanced": {},
	}

	for _, field := range fields {
		group := field.Group
		if group == "" {
			group = "content"
		}
		groups[group] = append(groups[group], field)
	}

	// Output in logical order
	groupOrder := []string{"content", "site", "settings", "style", "advanced"}

	var sb strings.Builder
	for _, groupName := range groupOrder {
		groupFields := groups[groupName]
		for _, field := range groupFields {
			fieldConfig, exists := config[field.Key]

			// Skip if not enabled (unless required)
			if !field.Required && (!exists || !fieldConfig.Enabled) {
				continue
			}

			// Get the value from runtime data first, then from config, then default
			var value string
			if runtimeVal, ok := data[field.Key]; ok && runtimeVal != "" {
				value = runtimeVal
			} else if fieldConfig.Value != "" {
				value = fieldConfig.Value
			} else if field.DefaultValue != "" {
				value = field.DefaultValue
			}

			// Skip empty values and checkbox-only fields that are just enabled flags
			if value == "" || value == "true" || value == "false" {
				// For checkbox fields, we only include them if there's actual runtime data
				if field.Type == entities.ContextFieldTypeCheckbox {
					if runtimeVal, ok := data[field.Key]; ok && runtimeVal != "" && runtimeVal != "true" && runtimeVal != "false" {
						value = runtimeVal
					} else {
						continue
					}
				} else {
					continue
				}
			}

			// Format and append to output
			sb.WriteString(field.Label)
			sb.WriteString(": ")
			sb.WriteString(value)
			sb.WriteString("\n")
		}
	}

	// Add runtime-only fields if present in data
	for key, label := range runtimeOnlyFields {
		if value, ok := data[key]; ok && strings.TrimSpace(value) != "" {
			sb.WriteString("\n")
			sb.WriteString(label)
			sb.WriteString(":\n")
			sb.WriteString(value)
			sb.WriteString("\n")
		}
	}

	return strings.TrimSpace(sb.String())
}

// mergeConfigs merges base config with overrides
func (b *PromptBuilder) mergeConfigs(base, override entities.ContextConfig) entities.ContextConfig {
	result := make(entities.ContextConfig)

	// Copy base config
	for k, v := range base {
		result[k] = v
	}

	// Apply overrides
	for k, v := range override {
		result[k] = v
	}

	return result
}

// buildLegacy handles old format prompts with {{placeholder}} syntax
func (b *PromptBuilder) buildLegacy(prompt *entities.Prompt, data map[string]string) *BuildResult {
	systemPrompt := prompt.SystemPrompt
	userPrompt := prompt.UserPrompt

	// Replace placeholders
	for key, value := range data {
		placeholder := "{{" + key + "}}"
		systemPrompt = strings.ReplaceAll(systemPrompt, placeholder, value)
		userPrompt = strings.ReplaceAll(userPrompt, placeholder, value)
	}

	// Remove any unreplaced placeholders (clean output)
	systemPrompt = b.removeUnreplacedPlaceholders(systemPrompt)
	userPrompt = b.removeUnreplacedPlaceholders(userPrompt)

	return &BuildResult{
		SystemPrompt: systemPrompt,
		UserPrompt:   userPrompt,
	}
}

// removeUnreplacedPlaceholders removes any {{placeholder}} that wasn't replaced
func (b *PromptBuilder) removeUnreplacedPlaceholders(text string) string {
	result := text
	start := 0
	for {
		openIdx := strings.Index(result[start:], "{{")
		if openIdx == -1 {
			break
		}
		openIdx += start

		closeIdx := strings.Index(result[openIdx:], "}}")
		if closeIdx == -1 {
			break
		}
		closeIdx += openIdx + 2

		// Remove the placeholder
		result = result[:openIdx] + result[closeIdx:]
		start = openIdx
	}

	// Clean up multiple newlines and spaces
	for strings.Contains(result, "\n\n\n") {
		result = strings.ReplaceAll(result, "\n\n\n", "\n\n")
	}
	for strings.Contains(result, "  ") {
		result = strings.ReplaceAll(result, "  ", " ")
	}

	return strings.TrimSpace(result)
}

// GetDefaultContextConfig returns the default context config for a category
func (b *PromptBuilder) GetDefaultContextConfig(category entities.PromptCategory) entities.ContextConfig {
	return b.registry.GetDefaultContextConfig(category)
}

// GetFieldsByCategory returns all field definitions for a category
func (b *PromptBuilder) GetFieldsByCategory(category entities.PromptCategory) []*entities.ContextFieldDefinition {
	return b.registry.GetByCategory(category)
}
