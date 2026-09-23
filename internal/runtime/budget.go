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

	CodeUnpricedStep = "unpriced_step"
)

func (e *Engine) EstimateRun(ctx context.Context, record run.Run, spec template.TemplateSpec) (run.Estimate, error) {
	body := int(float64(targetWords(spec)) * tokensPerWord)

	estimate := run.Estimate{Findings: make([]run.EstimateFinding, 0)}
	for _, step := range run.Enabled(record.Recipe) {
		def, known := e.registry.Lookup(step.Name)
		if !known {
			return run.Estimate{}, errors.New(errors.NotFound, "the recipe names the unknown step "+step.Name).
				WithDetail("step", step.Name)
		}
		if def.Price.Unpriced {
			estimate.Findings = append(estimate.Findings, run.EstimateFinding{
				Code:    CodeUnpricedStep,
				Message: "the step " + def.Name + " may call an image model, which this estimate does not price",
			})
		}
		if def.Role == "" {
			continue
		}

		calls := callsOf(def, spec, step.Params)
		if calls == 0 {
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

		usage := usageOf(outputOf(def, body))
		estimate.Tokens += usage.Total * calls
		estimate.USD += llm.Cost(usage, info) * float64(calls)
	}

	targets := max(len(record.Targets), 1)
	estimate.Tokens *= targets
	estimate.USD *= float64(targets)
	return estimate, nil
}

func usageOf(output int) llm.Usage {
	input := promptOverhead + int(float64(output)*inputShareOfOutput)
	return llm.Usage{Input: input, Output: output, Total: input + output}
}

func outputOf(def run.StepDef, body int) int {
	if def.Price.OutputTokens > 0 {
		return def.Price.OutputTokens
	}
	return body
}

func callsOf(def run.StepDef, spec template.TemplateSpec, params map[string]any) int {
	if def.Price.Calls == nil {
		return 1
	}
	return max(def.Price.Calls(spec, params), 0)
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
