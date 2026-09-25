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
	CheckpointRepairs = "repairs"

	CodePhraseTemplated = "phrase_templated"

	defaultIterations = 2
	maxIterations     = 4
	repairTokens      = 256

	leadWhy = "The template asks the first paragraph of the page to carry the primary keyword, and it does not yet."
)

type repairPrompt struct {
	Page      pagemap.Page
	Entity    graph.Entity
	Phrase    string
	Why       string
	Paragraph string
}

type owedPhrase struct {
	text  string
	why   string
	url   string
	index int
	lead  bool
}

func RepairLinks(deps Deps) run.StepDef {
	return run.StepDef{
		Name:     NameRepairLinks,
		Role:     domainllm.RoleLinker,
		Requires: []run.ArtifactKind{run.ArtifactLinkContext, run.ArtifactBodyHTML},
		Produces: []run.ArtifactKind{run.ArtifactBodyHTML},
		Retry:    run.RetryPolicy{Max: 2},
		Timeout:  editorTimeout,
		Price: run.Price{
			OutputTokens: repairTokens,
			Calls: func(spec template.TemplateSpec, params map[string]any) int {
				phrases := spec.LinkRules.UpDepth
				if spec.KeywordRules.PrimaryInFirstParagraph {
					phrases++
				}
				return iterations(params) * phrases
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

			linker := sentenceWriter{deps: deps, sc: sc, entity: entity, ref: ref, tries: iterationsOf(sc)}
			findings := make([]content.Finding, 0)
			settled := make(map[string]struct{})
			tokens, written := 0, 0

			result := content.InsertLinks(doc, lc, policy)
			for {
				owed, ok := nextOwed(doc, result, policy, sc.Spec, entity, settled)
				if !ok {
					break
				}
				settled[owed.text] = struct{}{}

				sentence, used, writeErr := linker.write(ctx, doc, owed)
				tokens += used
				if writeErr != nil {
					return run.Result{}, writeErr
				}
				if sentence == "" {
					sentence = templated(owed)
					findings = append(findings, templatedFinding(owed, linker.tries))
				}
				if placeErr := settleSentence(doc, owed.index, sentence); placeErr != nil {
					return run.Result{}, placeErr
				}
				written++
				result = content.InsertLinks(doc, lc, policy)
			}

			body, err := doc.Render()
			if err != nil {
				return run.Result{}, err
			}

			checkpoint := run.NewCheckpoint()
			if setErr := run.Set(checkpoint, checkpointLinks, result); setErr != nil {
				return run.Result{}, setErr
			}
			if setErr := run.Set(checkpoint, CheckpointRepairs, findings); setErr != nil {
				return run.Result{}, setErr
			}

			return run.Result{
				Artifacts:  []run.Artifact{{Kind: run.ArtifactBodyHTML, Blob: []byte(body)}},
				Checkpoint: checkpoint,
				Tokens:     tokens,
				Message:    repairMessage(written, len(findings)),
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

func nextOwed(doc *content.Document, result content.InsertResult, policy template.LinkPolicy,
	spec template.TemplateSpec, entity graph.Entity, settled map[string]struct{}) (owedPhrase, bool) {
	primary := strings.TrimSpace(entity.PrimaryKeyword)
	if spec.KeywordRules.PrimaryInFirstParagraph && primary != "" {
		if _, done := settled[primary]; !done && !leadCarries(doc, primary) {
			return owedPhrase{text: primary, why: leadWhy, lead: true}, true
		}
	}

	for i := range result.Decisions {
		decision := &result.Decisions[i]
		target := decision.Target
		if len(target.Anchors) == 0 {
			continue
		}
		if decision.Outcome != content.OutcomeAnchorNotFound && decision.Outcome != content.OutcomePositionRule {
			continue
		}
		if _, done := settled[target.Anchors[0]]; done {
			continue
		}
		return owedPhrase{
			text:  target.Anchors[0],
			url:   target.URL,
			index: placeFor(doc, policy, target),
			why:   whyOwed(target),
		}, true
	}
	return owedPhrase{}, false
}

func placeFor(doc *content.Document, policy template.LinkPolicy, target content.LinkTarget) int {
	if target.Relation == content.RelationUp {
		return insertionPoint(doc, policy)
	}
	return max(len(doc.Paragraphs())-1, 0)
}

func whyOwed(target content.LinkTarget) string {
	switch target.Relation {
	case content.RelationDown:
		return "The page has to link down to " + target.URL + ", a page below it in the entity graph, " +
			"and no paragraph carries an anchor for it yet; the sentence closes the page."
	case content.RelationSibling:
		return "The page has to link to " + target.URL + ", a page related to it in the entity graph, " +
			"and no paragraph carries an anchor for it yet; the sentence closes the page."
	default:
		return "The page has to link to " + target.URL + ", which sits " + string(target.Relation) +
			" of this page in the entity graph, and no paragraph where that link may go carries an anchor for it yet."
	}
}

func leadCarries(doc *content.Document, phrase string) bool {
	paragraphs := doc.Paragraphs()
	if len(paragraphs) == 0 {
		return false
	}
	_, _, found := content.FindFold(content.TextOf(paragraphs[0]), phrase)
	return found
}

func insertionPoint(doc *content.Document, policy template.LinkPolicy) int {
	paragraphs := doc.Paragraphs()
	if len(paragraphs) == 0 {
		return 0
	}
	if limit := policy.Rules.ParentLinkWithinParagraphs; limit > 0 {
		return min(limit, len(paragraphs)) - 1
	}
	return 0
}

func settleSentence(doc *content.Document, index int, sentence string) error {
	paragraphs := doc.Paragraphs()
	if len(paragraphs) == 0 {
		return doc.PrependParagraph(sentence)
	}
	return doc.AppendSentence(min(index, len(paragraphs)-1), sentence)
}

type sentenceWriter struct {
	deps   Deps
	sc     *run.StepContext
	entity graph.Entity
	ref    domainllm.ModelRef
	tries  int
}

func (w sentenceWriter) write(ctx context.Context, doc *content.Document, owed owedPhrase) (sentence string, used int, err error) {
	for range w.tries {
		if err := ctx.Err(); err != nil {
			return "", used, stoppedWhileWriting(err, owed)
		}

		system, user, renderErr := render(NameRepairLinks, repairPrompt{
			Page: w.sc.Page, Entity: w.entity, Phrase: owed.text, Why: owed.why, Paragraph: contextParagraph(doc, owed.index),
		})
		if renderErr != nil {
			return "", used, renderErr
		}

		response, usage, callErr := port.Structured[content.RepairResponse](ctx, w.deps.LLM, port.Request{
			Ref:       w.ref,
			System:    system,
			Messages:  []port.Message{{Role: port.RoleUser, Text: user}},
			MaxTokens: repairTokens,
			Meta:      callMeta(w.sc, NameRepairLinks),
		})
		used += usage.Total
		if callErr != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return "", used, stoppedWhileWriting(ctxErr, owed)
			}
			if errors.IsCode(callErr, errors.Cancelled) {
				return "", used, callErr
			}
			continue
		}

		answered := strings.TrimSpace(response.Sentence)
		if _, _, found := content.FindFold(answered, owed.text); answered != "" && found {
			return answered, used, nil
		}
	}
	return "", used, nil
}

func stoppedWhileWriting(cause error, owed owedPhrase) error {
	return errors.Wrap(cause, errors.Cancelled, "the run stopped before a sentence carrying "+
		strconv.Quote(owed.text)+" was written")
}

func contextParagraph(doc *content.Document, index int) string {
	paragraphs := doc.Paragraphs()
	if len(paragraphs) == 0 {
		return ""
	}
	return content.TextOf(paragraphs[min(index, len(paragraphs)-1)])
}

func templated(owed owedPhrase) string {
	if owed.lead {
		return "This page is about " + owed.text + "."
	}
	return "Read more about " + owed.text + "."
}

func templatedFinding(owed owedPhrase, tries int) content.Finding {
	details := map[string]any{"phrase": owed.text, "paragraphIndex": owed.index, "tries": tries, "lead": owed.lead}
	if owed.url != "" {
		details["url"] = owed.url
	}
	return content.Finding{
		Severity: content.SeverityWarn,
		Code:     CodePhraseTemplated,
		Message: "the linker wrote no sentence carrying " + strconv.Quote(owed.text) + " in " + strconv.Itoa(tries) +
			" tries, so a plain one was added to the body; rewrite it on the site if it reads poorly",
		Details: details,
	}
}

func repairMessage(written, templatedCount int) string {
	if written == 0 {
		return "every owed phrase was already in place"
	}
	message := "wrote " + strconv.Itoa(written) + " owed phrases into the body"
	if templatedCount > 0 {
		message += ", " + strconv.Itoa(templatedCount) + " of them as plain sentences"
	}
	return message
}
