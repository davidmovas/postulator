package steps

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"slices"
	"strconv"

	"github.com/davidmovas/postulator/internal/application/llm"
	"github.com/davidmovas/postulator/internal/application/sitemodel"
	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/contextpack"
	"github.com/davidmovas/postulator/internal/domain/field"
	domainllm "github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const cacheableTokens = 1024

type stepTuning interface {
	Effort(role domainllm.Role) domainllm.ReasoningEffort
	InputBudget(role domainllm.Role) int
}

type modelInfo interface {
	Lookup(ctx context.Context, ref domainllm.ModelRef) (domainllm.ModelInfo, error)
}

type pictures interface {
	Picture(ctx context.Context, req sitemodel.PackRequest) (sitemodel.Picture, error)
}

func (d Deps) inputBudget(ctx context.Context, call modelCall, ceiling int, fixed ...string) int {
	if set := d.Tuning.InputBudget(call.role); set > 0 {
		return set
	}
	info, err := d.Models.Lookup(ctx, call.ref)
	if err != nil || info.ContextTokens <= 0 {
		return 0
	}
	room := info.ContextTokens - ceiling - domainllm.Allowance(d.Tuning.Effort(call.role))
	for _, text := range fixed {
		room -= domainllm.Tokens(text)
	}
	return max(room, 1)
}

func (d Deps) pictureOf(ctx context.Context, sc *run.StepContext, key field.Key, links *content.LinkContext, budget int) (sitemodel.Picture, error) {
	set, err := d.Types.Set(ctx, sc.Run.SiteID)
	if err != nil {
		return sitemodel.Picture{}, err
	}
	scope, _ := set.Scope(sc.Type.ID, key)
	if key == pagetype.LinkField(sc.Type.Preset) {
		scope.Links = sc.Links
	}
	claim := &sitemodel.Claim{Scope: scope, Links: links}
	if sc.Type.ID != "" || sc.Type.Name != "" {
		claim.Type = &sc.Type
	}
	return d.Pictures.Picture(ctx, sitemodel.PackRequest{PageID: sc.Page.ID, Field: string(key), Budget: budget, Claim: claim})
}

func (d Deps) fitted(ctx context.Context, sc *run.StepContext, call modelCall, key field.Key, links *content.LinkContext,
	ceiling int, bare func() (string, string, error), schema string) (sitemodel.Picture, error) {
	system, user, err := bare()
	if err != nil {
		return sitemodel.Picture{}, err
	}
	return d.pictureOf(ctx, sc, key, links, d.inputBudget(ctx, call, ceiling, system, user, schema))
}

const CodeInputBudgetTooSmall = "input_budget_too_small"

func packInput(deps Deps, role domainllm.Role, key func(page pagemap.Page) field.Key) func(context.Context, run.Target) (int, error) {
	if deps.Pictures == nil || deps.Tuning == nil {
		return nil
	}
	return func(ctx context.Context, target run.Target) (int, error) {
		seen, err := deps.Pictures.Picture(ctx, sitemodel.PackRequest{
			PageID: target.Page.ID, Field: string(key(target.Page)), Budget: deps.Tuning.InputBudget(role),
		})
		if err != nil {
			return 0, err
		}
		return seen.Fitted.Tokens, nil
	}
}

func inputPreflight(deps Deps, role domainllm.Role, key func(page pagemap.Page) field.Key) run.Preflight {
	return func(ctx context.Context, record run.Run, targets map[string]run.Target) ([]run.EstimateFinding, error) {
		findings := make([]run.EstimateFinding, 0)
		if deps.Pictures == nil || deps.Tuning == nil {
			return findings, nil
		}
		budget := deps.Tuning.InputBudget(role)
		if budget <= 0 {
			return findings, nil
		}
		for _, targetID := range record.Targets {
			page := targets[targetID].Page
			_, err := deps.Pictures.Picture(ctx, sitemodel.PackRequest{PageID: page.ID, Field: string(key(page)), Budget: budget})
			if err == nil {
				continue
			}
			refused, typed := stderrors.AsType[*errors.Error](err)
			if !typed || refused.Details["reason"] != contextpack.ReasonBudgetTooSmall {
				return nil, err
			}
			needed := 0
			if count, counted := refused.Details["needed"].(int); counted {
				needed = count
			}
			findings = append(findings, pageFinding(content.SeverityError, CodeInputBudgetTooSmall, page,
				"the "+string(role)+"'s input budget of "+strconv.Itoa(budget)+" tokens is smaller than the "+strconv.Itoa(needed)+
					" every call about "+page.Path+" must carry; raise llm.inputBudget."+string(role)+" in Settings → Models or set it to 0"))
		}
		return findings, nil
	}
}

func pageKey(page pagemap.Page) field.Key {
	if page.WPType == pagemap.WPProduct {
		return field.ProductDescription
	}
	return field.Body
}

func seoTitleKey(pagemap.Page) field.Key {
	return field.SEOTitle
}

func schemaText[T any]() string {
	schema, err := llm.SchemaFor[T]()
	if err != nil {
		return ""
	}
	encoded, err := json.Marshal(schema)
	if err != nil {
		return ""
	}
	return string(encoded)
}

func picture(fitted contextpack.Fitted, leave ...contextpack.SectionName) string {
	kept := fitted
	kept.Sections = slices.DeleteFunc(slices.Clone(fitted.Sections), func(section contextpack.Section) bool {
		return slices.Contains(leave, section.Name)
	})
	return kept.Text()
}

func cacheKey(step, siteID, system string) string {
	if domainllm.Tokens(system) < cacheableTokens {
		return ""
	}
	return step + ":" + siteID
}
