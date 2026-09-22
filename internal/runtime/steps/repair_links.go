package steps

import (
	"context"
	"strconv"
	"strings"

	port "github.com/davidmovas/postulator/internal/application/llm"
	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/graph"
	domainllm "github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	NameRepairLinks   = string(run.StepRepairLinks)
	ParamIterations   = "iterations"
	defaultIterations = 2
	maxIterations     = 4
	repairTokens      = 256
)

type repairPrompt struct {
	Page      pagemap.Page
	Entity    graph.Entity
	Phrase    string
	TargetURL string
	Relation  string
	Paragraph string
}

func RepairLinks(deps Deps) run.StepDef {
	return run.StepDef{
		Name:     NameRepairLinks,
		Role:     domainllm.RoleLinker,
		Requires: []run.ArtifactKind{run.ArtifactLinkContext, run.ArtifactBodyHTML},
		Produces: []run.ArtifactKind{run.ArtifactBodyHTML},
		Retry:    run.RetryPolicy{Max: 2},
		Price: run.Price{
			OutputTokens: repairTokens,
			Calls: func(spec template.TemplateSpec, params map[string]any) int {
				return iterations(params) * spec.LinkRules.UpDepth
			},
		},
		Run: func(ctx context.Context, sc *run.StepContext) (run.Result, error) {
			lc, err := linkContextOf(sc)
			if err != nil {
				return run.Result{}, err
			}
			doc, err := bodyOf(sc)
			if err != nil {
				return run.Result{}, err
			}
			policy, err := effectivePolicy(ctx, deps, sc)
			if err != nil {
				return run.Result{}, err
			}
			entity, err := entityOf(ctx, deps, sc)
			if err != nil {
				return run.Result{}, err
			}
			ref, err := deps.Profiles.Resolve(ctx, sc.Run.SiteID, domainllm.RoleLinker, sc.Spec.ModelProfiles)
			if err != nil {
				return run.Result{}, err
			}

			result := content.InsertLinks(doc, lc, policy)
			tokens := 0
			repaired := 0

			for range iterationsOf(sc) {
				outstanding := requiredMissing(result)
				if len(outstanding) == 0 {
					break
				}

				for i := range outstanding {
					used, sentenceErr := appendSentence(ctx, deps, doc, sc, entity, ref, outstanding[i])
					if sentenceErr != nil {
						return run.Result{}, sentenceErr
					}
					tokens += used
					repaired++
				}
				result = content.InsertLinks(doc, lc, policy)
			}

			checkpoint := run.NewCheckpoint()
			if setErr := run.Set(checkpoint, checkpointLinks, result); setErr != nil {
				return run.Result{}, setErr
			}

			return run.Result{
				Artifacts:  []run.Artifact{{Kind: run.ArtifactBodyHTML, Blob: []byte(doc.HTML())}},
				Checkpoint: checkpoint,
				Tokens:     tokens,
				Message:    "repaired " + strconv.Itoa(repaired) + " required links",
			}, nil
		},
	}
}

func iterationsOf(sc *run.StepContext) int {
	return iterations(sc.Params)
}

func iterations(params map[string]any) int {
	value, ok := params[ParamIterations]
	if !ok {
		return defaultIterations
	}

	count, ok := value.(float64)
	if !ok {
		return defaultIterations
	}
	return min(max(int(count), 1), maxIterations)
}

func requiredMissing(result content.InsertResult) []content.LinkTarget {
	out := make([]content.LinkTarget, 0, len(result.Missing))
	for _, target := range result.Missing {
		if target.Required && len(target.Anchors) > 0 {
			out = append(out, target)
		}
	}
	return out
}

func appendSentence(ctx context.Context, deps Deps, doc *content.Document, sc *run.StepContext,
	entity graph.Entity, ref domainllm.ModelRef, target content.LinkTarget) (int, error) {
	paragraphs := doc.Paragraphs()
	if len(paragraphs) == 0 {
		return 0, errors.New(errors.Invalid, "the body carries no paragraph to repair").
			WithDetail("pageId", sc.Page.ID)
	}
	index := insertionPoint(doc, sc.Spec)

	system, user, err := render(NameRepairLinks, repairPrompt{
		Page: sc.Page, Entity: entity, Phrase: target.Anchors[0], TargetURL: target.URL,
		Relation: string(target.Relation), Paragraph: content.TextOf(paragraphs[index]),
	})
	if err != nil {
		return 0, err
	}

	response, usage, err := port.Structured[content.RepairResponse](ctx, deps.LLM, port.Request{
		Ref:       ref,
		System:    system,
		Messages:  []port.Message{{Role: port.RoleUser, Text: user}},
		MaxTokens: repairTokens,
		Meta:      callMeta(sc, NameRepairLinks),
	})
	if err != nil {
		return 0, err
	}

	sentence := strings.TrimSpace(response.Sentence)
	if sentence == "" {
		return usage.Total, errors.New(errors.Invalid, "the model returned no sentence to insert").
			WithDetail("phrase", target.Anchors[0])
	}
	if _, _, found := content.FindFold(sentence, target.Anchors[0]); !found {
		return usage.Total, errors.New(errors.Invalid, "the sentence does not carry the phrase it was asked for").
			WithDetail("phrase", target.Anchors[0])
	}

	if appendErr := doc.AppendSentence(index, sentence); appendErr != nil {
		return usage.Total, appendErr
	}
	return usage.Total, nil
}

func insertionPoint(doc *content.Document, spec template.TemplateSpec) int {
	paragraphs := doc.Paragraphs()
	if len(paragraphs) == 0 {
		return 0
	}
	if limit := spec.LinkRules.ParentLinkWithinParagraphs; limit > 0 {
		return min(limit, len(paragraphs)) - 1
	}
	return 0
}
