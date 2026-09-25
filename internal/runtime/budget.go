package runtime

import (
	"context"
	"slices"
	"strconv"

	"github.com/davidmovas/postulator/internal/application/templates"
	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	tokensPerWord       = 2.2
	promptOverhead      = 800
	fallbackTargetWords = 800
	inputShareOfOutput  = 0.5

	CodeUnpricedStep       = "unpriced_step"
	CodeTemplateUnresolved = "template_unresolved"
	CodeRecipeDiffers      = "recipe_differs"
	CodeModelUnresolved    = "model_unresolved"
	CodeModelUnknown       = "model_unknown"
	CodeProviderKeyMissing = "provider_key_missing"
	CodeImagesStepOff      = "images_step_off"
)

type pricing struct {
	findings []run.EstimateFinding
	targets  []run.Target
	seen     map[string]struct{}
	tokens   int
	usd      float64
}

func (p *pricing) add(finding run.EstimateFinding) {
	key := finding.Code + "\x00" + finding.PageID + "\x00" + finding.Message
	if _, dup := p.seen[key]; dup {
		return
	}
	p.seen[key] = struct{}{}
	p.findings = append(p.findings, finding)
}

func (e *Engine) EstimateRun(ctx context.Context, record run.Run) (run.Estimate, error) {
	defs, err := e.enabledDefs(record.Recipe)
	if err != nil {
		return run.Estimate{}, err
	}

	priced := &pricing{findings: make([]run.EstimateFinding, 0), seen: make(map[string]struct{})}
	targets, err := e.planTargets(ctx, record, priced)
	if err != nil {
		return run.Estimate{}, err
	}

	for i := range defs {
		if defs[i].Preflight == nil {
			continue
		}
		findings, preflightErr := defs[i].Preflight(ctx, record, targets)
		if preflightErr != nil {
			return run.Estimate{}, preflightErr
		}
		for _, finding := range findings {
			priced.add(finding)
		}
	}

	priced.targets = e.pricedTargets(record, targets)
	for t := range priced.targets {
		for i := range defs {
			if priceErr := e.price(ctx, record, defs[i], &priced.targets[t], priced); priceErr != nil {
				return run.Estimate{}, priceErr
			}
		}
	}
	return run.Estimate{Tokens: priced.tokens, USD: priced.usd, Findings: priced.findings}, nil
}

func (e *Engine) enabledDefs(recipe []template.StepSpec) ([]run.StepDef, error) {
	enabled := run.Enabled(recipe)
	defs := make([]run.StepDef, 0, len(enabled))
	for _, step := range enabled {
		def, known := e.registry.Lookup(step.Name)
		if !known {
			return nil, errors.New(errors.NotFound, "the recipe names the unknown step "+step.Name).
				WithDetail("step", step.Name)
		}
		defs = append(defs, def)
	}
	return defs, nil
}

func (e *Engine) planTargets(ctx context.Context, record run.Run, priced *pricing) (map[string]run.Target, error) {
	targets := make(map[string]run.Target, len(record.Targets))
	if !record.Kind.PageScoped() {
		return targets, nil
	}

	_, ownsRecipe := record.Kind.Recipe()
	undrawn := writesWithoutImages(record.Recipe)
	pages := e.targetPages(ctx, record.Targets)
	for _, targetID := range record.Targets {
		target := run.Target{Page: pages[targetID]}
		resolved, err := e.deps.Specs.ResolveForPage(ctx, templates.ResolveForPageRequest{PageID: targetID})
		if err != nil {
			if !errors.IsCode(err, errors.NotFound) && !errors.IsCode(err, errors.Invalid) {
				return nil, err
			}
			priced.add(run.EstimateFinding{
				Severity: content.SeverityError, Code: CodeTemplateUnresolved, PageID: targetID, Path: target.Page.Path,
				Message: "no template answers for " + pathOrID(target.Page, targetID) + ": " + err.Error(),
			})
			targets[targetID] = target
			continue
		}
		target.Spec = resolved.Spec
		targets[targetID] = target

		if own := stepNames(resolved.Spec.Recipe); !ownsRecipe && len(own) > 0 && !slices.Equal(own, stepNames(record.Recipe)) {
			priced.add(run.EstimateFinding{
				Severity: content.SeverityWarn, Code: CodeRecipeDiffers, PageID: targetID, Path: target.Page.Path,
				Message: "the template of " + pathOrID(target.Page, targetID) +
					" names other steps than this run follows; the run's recipe applies to every page",
			})
		}
		if wanted := resolved.Spec.Images.Wanted(); undrawn && wanted > 0 {
			priced.add(run.EstimateFinding{
				Severity: content.SeverityWarn, Code: CodeImagesStepOff, PageID: targetID, Path: target.Page.Path,
				Message: "the template of " + pathOrID(target.Page, targetID) + " asks for " + strconv.Itoa(wanted) +
					" images, but this run's recipe leaves the image step out, so the page is written without them",
			})
		}
	}
	return targets, nil
}

func writesWithoutImages(recipe []template.StepSpec) bool {
	names := stepNames(recipe)
	return slices.Contains(names, string(run.StepGenerateBody)) && !slices.Contains(names, string(run.StepGenerateImages))
}

func (e *Engine) pricedTargets(record run.Run, targets map[string]run.Target) []run.Target {
	if !record.Kind.PageScoped() {
		return []run.Target{{}}
	}
	out := make([]run.Target, 0, len(record.Targets))
	for _, targetID := range record.Targets {
		out = append(out, targets[targetID])
	}
	return out
}

func (e *Engine) price(ctx context.Context, record run.Run, def run.StepDef, target *run.Target, priced *pricing) error {
	calls := callsOf(def, target.Spec, run.ParamsFor(record.Recipe, def.Name))
	if calls == 0 {
		return nil
	}
	if def.Price.Unpriced {
		priced.add(run.EstimateFinding{
			Severity: content.SeverityWarn, Code: CodeUnpricedStep,
			Message: "the step " + def.Name + " may call an image model, which this estimate does not price",
		})
		return nil
	}

	ref, priceable, err := e.modelOf(ctx, record, def, target, priced)
	if err != nil || !priceable {
		return err
	}

	configured, err := e.deps.Keys.Has(ctx, llm.SecretRef(ref.Provider))
	if err != nil {
		return err
	}
	if !configured {
		priced.add(run.EstimateFinding{
			Severity: content.SeverityError, Code: CodeProviderKeyMissing,
			Message: "the provider " + ref.Provider + " holds no API key, and " + def.Name + " calls " + ref.String() +
				"; add the key on the Models screen",
		})
	}

	info, err := e.deps.Catalog.Lookup(ctx, ref)
	if err != nil {
		if !errors.IsCode(err, errors.NotFound) && !errors.IsCode(err, errors.Invalid) {
			return err
		}
		priced.add(run.EstimateFinding{
			Severity: content.SeverityWarn, Code: CodeModelUnknown,
			Message: "the catalog does not price " + ref.String() + ", which " + def.Name + " calls, so its cost is not counted",
		})
		return nil
	}

	usage := usageOf(outputOf(def, int(float64(targetWords(target.Spec))*tokensPerWord)))
	priced.tokens += usage.Total * calls
	priced.usd += llm.Cost(usage, info) * float64(calls)
	return nil
}

func (e *Engine) modelOf(ctx context.Context, record run.Run, def run.StepDef, target *run.Target,
	priced *pricing) (ref llm.ModelRef, priceable bool, err error) {
	if def.Price.Ref != nil {
		return *def.Price.Ref, true, nil
	}
	if def.Role == "" {
		return llm.ModelRef{}, false, nil
	}

	ref, err = e.deps.Profiles.Resolve(ctx, record.SiteID, def.Role, target.Spec.ModelProfiles)
	if err != nil {
		if !errors.IsCode(err, errors.NotFound) && !errors.IsCode(err, errors.Invalid) {
			return llm.ModelRef{}, false, err
		}
		priced.add(run.EstimateFinding{
			Severity: content.SeverityError, Code: CodeModelUnresolved,
			Message: "no model answers for the role " + string(def.Role) + ", which " + def.Name + " needs: " + err.Error(),
		})
		return llm.ModelRef{}, false, nil
	}
	return ref, true, nil
}

func pathOrID(page pagemap.Page, id string) string {
	if page.Path != "" {
		return page.Path
	}
	return id
}

func stepNames(recipe []template.StepSpec) []string {
	out := make([]string, 0, len(recipe))
	for _, step := range run.Enabled(recipe) {
		out = append(out, step.Name)
	}
	return out
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
