package generation

import (
	"fmt"
	"strings"

	"github.com/davidmovas/postulator/internal/domain/jobs/execution/commands"
	"github.com/davidmovas/postulator/internal/domain/jobs/execution/fault"
	"github.com/davidmovas/postulator/internal/domain/jobs/execution/pipeline"
	"github.com/davidmovas/postulator/internal/domain/prompts"
)

var _ pipeline.Command = (*RenderPromptCommand)(nil)

type RenderPromptCommand struct {
	*commands.BaseCommand
	promptService prompts.Service
}

func NewRenderPromptCommand(promptService prompts.Service) *RenderPromptCommand {
	return &RenderPromptCommand{
		BaseCommand: commands.NewBaseCommand(
			"render_prompt",
			pipeline.StateExecutionCreated,
			pipeline.StatePromptRendered,
		),
		promptService: promptService,
	}
}

func (c *RenderPromptCommand) Execute(ctx *pipeline.Context) error {
	if !ctx.HasExecution() {
		return fault.NewFatalError(fault.ErrCodeInvalidJob, c.Name(), "execution not created")
	}

	if !ctx.HasSelection() {
		return fault.NewFatalError(fault.ErrCodeInvalidJob, c.Name(), "topic or category not selected")
	}

	placeholders := c.buildPlaceholders(ctx)

	// Strict validation for jobs - user must provide all required placeholder values
	if err := c.validatePlaceholdersStrict(ctx, placeholders); err != nil {
		return fault.NewFatalError(fault.ErrCodePromptRenderFailed, c.Name(), err.Error())
	}

	systemPrompt, userPrompt, err := c.promptService.RenderPrompt(
		ctx.Context(),
		ctx.Execution.Prompt.ID,
		placeholders,
	)
	if err != nil {
		return fault.WrapError(err, fault.ErrCodePromptRenderFailed, c.Name(), "failed to render prompt")
	}

	if !ctx.HasGeneration() {
		ctx.InitGenerationPhase()
	}

	ctx.Generation.SystemPrompt = systemPrompt
	ctx.Generation.UserPrompt = userPrompt

	return nil
}

func (c *RenderPromptCommand) buildPlaceholders(ctx *pipeline.Context) map[string]string {
	placeholders := make(map[string]string)

	for _, placeholder := range ctx.Execution.Prompt.Placeholders {
		placeholders[placeholder] = ""
	}

	placeholders["title"] = ctx.Selection.VariationTopic.Title
	placeholders["siteName"] = ctx.Validated.Site.Name
	placeholders["siteUrl"] = ctx.Validated.Site.URL

	var categoryNames []string
	for _, cat := range ctx.Selection.Categories {
		categoryNames = append(categoryNames, cat.Name)
	}

	placeholders["category"] = strings.Join(categoryNames, ", ")

	if ctx.Job.PlaceholdersValues != nil {
		for placeholder, value := range ctx.Job.PlaceholdersValues {
			placeholders[placeholder] = value
		}
	}

	return placeholders
}

// validatePlaceholdersStrict checks that all required placeholders have non-empty values.
// For jobs, users must explicitly provide values for all placeholders in the prompt.
func (c *RenderPromptCommand) validatePlaceholdersStrict(ctx *pipeline.Context, placeholders map[string]string) error {
	var missing []string

	for _, placeholder := range ctx.Execution.Prompt.Placeholders {
		value, exists := placeholders[placeholder]
		if !exists || strings.TrimSpace(value) == "" {
			missing = append(missing, placeholder)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required placeholders: %s", strings.Join(missing, ", "))
	}

	return nil
}
