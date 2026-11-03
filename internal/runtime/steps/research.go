package steps

import (
	"context"
	"strconv"
	"strings"
	"time"

	port "github.com/davidmovas/postulator/internal/application/llm"
	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/contextpack"
	domainllm "github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	NameResearch = string(run.StepResearch)
	ParamFocus   = "focus"

	researchAnswerTokens = 1600
	researchTimeout      = 10 * time.Minute
)

var searchTokensByDepth = map[domainllm.SearchDepth]int{
	domainllm.DepthLow:    3000,
	domainllm.DepthMedium: 6000,
	domainllm.DepthHigh:   12000,
}

type searchTuning interface {
	WebSearch() port.WebSearch
}

type researchPrompt struct {
	Picture string
	Spec    template.TemplateSpec
	Focus   string
	Product bool
}

func Research(deps Deps) run.StepDef {
	return run.StepDef{
		Name:     NameResearch,
		Role:     domainllm.RoleResearcher,
		Produces: []run.ArtifactKind{run.ArtifactResearch},
		Retry:    run.RetryPolicy{Max: 2},
		Timeout:  researchTimeout,
		Price: run.Price{
			OutputTokens: researchAnswerTokens, Input: packInput(deps, domainllm.RoleResearcher, pageKey),
			Searches: searchesPriced(deps),
		},
		Preflight: inputPreflight(deps, domainllm.RoleResearcher, pageKey),
		Run: func(ctx context.Context, sc *run.StepContext) (run.Result, error) {
			if deps.Search == nil {
				return run.Result{}, errors.New(errors.Internal, "the research step was built without the web search settings")
			}
			call, err := modelFor(ctx, deps, sc, NameResearch, domainllm.RoleResearcher)
			if err != nil {
				return run.Result{}, err
			}
			prompt := researchPrompt{Spec: sc.Spec, Focus: focusOf(sc.Params), Product: sc.Page.WPType == pagemap.WPProduct}
			seen, err := deps.fitted(ctx, sc, call, pageKey(sc.Page), nil, researchAnswerTokens, func() (string, string, error) {
				return prompts.Render(NameResearch, prompt)
			}, "")
			if err != nil {
				return run.Result{}, err
			}
			prompt.Picture = picture(seen.Fitted, contextpack.SectionTargets)
			request, err := stepRequest(sc, call, prompt, researchAnswerTokens)
			if err != nil {
				return run.Result{}, err
			}
			search := deps.Search.WebSearch()
			request.WebSearch = &search

			answer, err := deps.LLM.Complete(ctx, request)
			if err != nil {
				return run.Result{}, err
			}
			if answer.FinishReason == port.FinishContentFilter {
				return run.Result{}, errors.New(errors.External, "the model refused to research "+sc.Page.Path).WithRetry(0)
			}
			found := content.Research{Text: strings.TrimSpace(answer.Text), Sources: sourcesOf(answer.Sources), Searches: answer.Usage.WebSearches}
			encoded, err := encode(found, "research")
			if err != nil {
				return run.Result{}, err
			}
			return run.Result{
				Artifacts: []run.Artifact{{Kind: run.ArtifactResearch, Blob: encoded}},
				Tokens:    answer.Usage.Total,
				Message: "researched " + sc.Page.Path + ": " + strconv.Itoa(len(found.Facts())) + " facts from " +
					strconv.Itoa(len(found.Sources)) + " sources in " + strconv.Itoa(found.Searches) + " searches",
			}, nil
		},
	}
}

func searchesPriced(deps Deps) func() (int, int) {
	if deps.Search == nil {
		return nil
	}
	return func() (int, int) {
		search := deps.Search.WebSearch()
		return search.MaxCalls, searchTokensByDepth[search.Depth]
	}
}

func focusOf(params map[string]any) string {
	if focus, given := params[ParamFocus].(string); given {
		return strings.TrimSpace(focus)
	}
	return ""
}

func sourcesOf(found []port.Source) []content.Source {
	out := make([]content.Source, 0, len(found))
	for _, source := range found {
		out = append(out, content.Source{URL: source.URL, Title: source.Title})
	}
	return out
}

func researchOf(sc *run.StepContext) (*content.Research, error) {
	found, held, err := decodeArtifact[content.Research](sc, run.ArtifactResearch)
	if err != nil || !held || found.Empty() {
		return nil, err
	}
	return &found, nil
}
