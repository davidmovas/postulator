package generation

import (
	"fmt"
	"strings"

	"github.com/davidmovas/postulator/internal/domain/entities"
)

// Default prompts use system_instructions and user_instructions placeholders
// which are built dynamically from ContentSettings
const DefaultPageSystemPrompt = `{{system_instructions}}`
const DefaultPageUserPrompt = `{{user_instructions}}`

var DefaultPagePlaceholders = []string{
	"title",
	"path",
	"keywords",
	"hierarchy",
	"context",
	"language",
	"word_count",
	"writing_style",
	"content_tone",
	"custom_instructions",
	"internal_links",
	"system_instructions",
	"user_instructions",
}

type PromptRenderer struct {
	systemPrompt string
	userPrompt   string
}

func NewPromptRenderer(systemPrompt, userPrompt string) *PromptRenderer {
	return &PromptRenderer{
		systemPrompt: systemPrompt,
		userPrompt:   userPrompt,
	}
}

func NewDefaultPromptRenderer() *PromptRenderer {
	return &PromptRenderer{
		systemPrompt: DefaultPageSystemPrompt,
		userPrompt:   DefaultPageUserPrompt,
	}
}

func (r *PromptRenderer) Render(placeholders map[string]string) (system, user string) {
	system = r.replacePlaceholders(r.systemPrompt, placeholders)
	user = r.replacePlaceholders(r.userPrompt, placeholders)
	return
}

func (r *PromptRenderer) replacePlaceholders(template string, placeholders map[string]string) string {
	result := template
	for key, value := range placeholders {
		result = strings.ReplaceAll(result, "{{"+key+"}}", value)
	}
	return result
}

func GetDefaultPromptEntity() *entities.Prompt {
	return &entities.Prompt{
		Name:         "Default Page Generation",
		SystemPrompt: DefaultPageSystemPrompt,
		UserPrompt:   DefaultPageUserPrompt,
		Placeholders: DefaultPagePlaceholders,
	}
}

type NodeContext struct {
	Title              string
	Path               string
	Keywords           []string
	Hierarchy          []HierarchyNode
	Context            string
	Language           string
	WordCount          string
	WritingStyle       string
	ContentTone        string
	CustomInstructions string
	LinkTargets        []LinkTarget // Internal links to include in content
}

type HierarchyNode struct {
	Title string
	Path  string
	Depth int
}

func BuildPlaceholders(ctx NodeContext) map[string]string {
	// Build basic placeholders
	keywords := ""
	if len(ctx.Keywords) > 0 {
		keywords = strings.Join(ctx.Keywords, ", ")
	}

	hierarchy := ""
	if len(ctx.Hierarchy) > 0 {
		var parts []string
		for _, h := range ctx.Hierarchy {
			indent := strings.Repeat("  ", h.Depth)
			parts = append(parts, indent+"- "+h.Title+" ("+h.Path+")")
		}
		hierarchy = strings.Join(parts, "\n")
	}

	// Apply defaults for empty values
	language := ctx.Language
	if language == "" {
		language = "English"
	}

	wordCount := ctx.WordCount
	if wordCount == "" {
		wordCount = "500"
	}

	writingStyle := ctx.WritingStyle
	if writingStyle == "" {
		writingStyle = "professional"
	}

	contentTone := ctx.ContentTone
	if contentTone == "" {
		contentTone = "informative"
	}

	// Build internal links section
	internalLinks := buildInternalLinksSection(ctx.LinkTargets)

	// Build system instructions from settings
	systemInstructions := buildSystemInstructions(language, wordCount, writingStyle, contentTone, ctx.CustomInstructions, len(ctx.LinkTargets) > 0)

	// Build user instructions from node data
	userInstructions := buildUserInstructions(ctx.Title, ctx.Path, keywords, hierarchy, ctx.Context, internalLinks)

	return map[string]string{
		"title":               ctx.Title,
		"path":                ctx.Path,
		"keywords":            keywords,
		"hierarchy":           hierarchy,
		"context":             ctx.Context,
		"language":            language,
		"word_count":          wordCount,
		"writing_style":       writingStyle,
		"content_tone":        contentTone,
		"custom_instructions": ctx.CustomInstructions,
		"internal_links":      internalLinks,
		"system_instructions": systemInstructions,
		"user_instructions":   userInstructions,
	}
}

func buildSystemInstructions(language, wordCount, writingStyle, contentTone, customInstructions string, hasLinks bool) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Generate WordPress page content in %s.\n", language))
	sb.WriteString(fmt.Sprintf("Word count: %s words maximum.\n", wordCount))
	sb.WriteString(fmt.Sprintf("Writing style: %s.\n", writingStyle))
	sb.WriteString(fmt.Sprintf("Tone: %s.\n", contentTone))
	sb.WriteString("Use HTML tags: p, h2, h3, ul, li.\n")
	sb.WriteString("Be concise and direct.")

	if hasLinks {
		sb.WriteString("\n\nIMPORTANT: You MUST include ALL specified internal links in the content. ")
		sb.WriteString("Use <a href=\"PATH\">ANCHOR TEXT</a> format. ")
		sb.WriteString("Place links naturally within the text where they fit contextually. ")
		sb.WriteString("If anchor text is not specified, choose appropriate anchor text based on context.")
	}

	if customInstructions != "" {
		sb.WriteString("\n\nAdditional instructions: ")
		sb.WriteString(customInstructions)
	}

	return sb.String()
}

func buildUserInstructions(title, path, keywords, hierarchy, context, internalLinks string) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Create content for page: %s\n", title))
	sb.WriteString(fmt.Sprintf("URL path: %s\n", path))

	if keywords != "" {
		sb.WriteString(fmt.Sprintf("Target keywords: %s\n", keywords))
	}

	if hierarchy != "" {
		sb.WriteString(fmt.Sprintf("\nSite hierarchy:\n%s\n", hierarchy))
	}

	if context != "" {
		sb.WriteString(fmt.Sprintf("\nContext: %s\n", context))
	}

	if internalLinks != "" {
		sb.WriteString(fmt.Sprintf("\n%s", internalLinks))
	}

	return sb.String()
}

// buildInternalLinksSection creates a formatted section describing required internal links
func buildInternalLinksSection(links []LinkTarget) string {
	if len(links) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("REQUIRED INTERNAL LINKS (must be included in content):\n")

	for i, link := range links {
		sb.WriteString(fmt.Sprintf("%d. Link to: %s\n", i+1, link.TargetTitle))
		sb.WriteString(fmt.Sprintf("   Path: %s\n", link.TargetPath))
		if link.AnchorText != nil && *link.AnchorText != "" {
			sb.WriteString(fmt.Sprintf("   Suggested anchor: \"%s\"\n", *link.AnchorText))
		} else {
			sb.WriteString("   Anchor: Choose appropriate text based on context\n")
		}
	}

	return sb.String()
}
