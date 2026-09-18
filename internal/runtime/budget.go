package runtime

import (
	"context"

	"github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	tokensPerWord       = 1.4
	promptOverhead      = 800
	fallbackTargetWords = 800
	inputShareOfOutput  = 0.5
)

func (e *Engine) EstimateRun(ctx context.Context, record run.Run, spec template.TemplateSpec) (run.Estimate, error) {
	output := int(float64(targetWords(spec)) * tokensPerWord)
	input := promptOverhead + int(float64(output)*inputShareOfOutput)
	usage := llm.Usage{Input: input, Output: output, Total: input + output}

	var estimate run.Estimate
	for _, step := range run.Enabled(record.Recipe) {
		def, known := e.registry.Lookup(step.Name)
		if !known {
			return run.Estimate{}, errors.New(errors.NotFound, "the recipe names the unknown step "+step.Name).
				WithDetail("step", step.Name)
		}
		if def.Role == "" {
			continue
		}

		ref, err := e.deps.Profiles.Resolve(ctx, record.SiteID, def.Role, spec.ModelProfiles)
		if err != nil {
			return run.Estimate{}, err
		}
		info, err := e.deps.Catalog.Lookup(ctx, ref)
		if err != nil {
			return run.Estimate{}, err
		}

		estimate.Tokens += usage.Total
		estimate.USD += llm.Cost(usage, info)
	}

	targets := max(len(record.Targets), 1)
	estimate.Tokens *= targets
	estimate.USD *= float64(targets)
	return estimate, nil
}

func targetWords(spec template.TemplateSpec) int {
	words := 0
	for i := range spec.Sections {
		words += spec.Sections[i].TargetWords
	}
	if words == 0 {
		words = spec.Length.Max
	}
	if words == 0 {
		words = fallbackTargetWords
	}
	return words
}
