package steps

import (
	"context"
	"strings"

	port "github.com/davidmovas/postulator/internal/application/llm"
	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/contextpack"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/keyword"
	domainllm "github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/template"
)

const (
	NameGenerateMeta = string(run.StepGenerateMeta)

	CodeMetaNotWritten = "meta_not_written"

	metaTokens = 512
	ellipsis   = "…"
)

type metaAnswer struct {
	Title         string `json:"title" description:"The meta title of the page, following the title pattern"`
	Description   string `json:"description" description:"The meta description, one sentence that earns the click"`
	OGTitle       string `json:"ogTitle" description:"The Open Graph title, which may be shorter than the meta title"`
	OGDescription string `json:"ogDescription" description:"The Open Graph description"`
}

type Meta struct {
	Title         string            `json:"title"`
	Description   string            `json:"description"`
	OGTitle       string            `json:"ogTitle"`
	OGDescription string            `json:"ogDescription"`
	Findings      []content.Finding `json:"findings,omitempty"`
}

type metaPrompt struct {
	Page        pagemap.Page
	Entity      graph.Entity
	Keywords    keyword.List
	Spec        template.TemplateSpec
	Draft       content.ContentDraft
	SiteName    string
	Pattern     string
	Picture     string
	Description []string
	Standing    bool
}

func metaRequires(scope []field.Key) []run.ArtifactKind {
	if run.WritesBody(scope) {
		return []run.ArtifactKind{run.ArtifactDraft}
	}
	return nil
}

func keepsBody(sc *run.StepContext) bool {
	scope := sc.Fields()
	return len(scope) > 0 && !run.WritesBody(scope)
}

func asItStands(facts contextpack.Facts, page pagemap.Page) content.ContentDraft {
	standing := content.ContentDraft{Title: page.Title, H1: facts.H1, Summary: facts.Page.Summary.Text}
	if title, held := facts.Field(field.Title); held && strings.TrimSpace(title.Current) != "" {
		standing.Title = strings.TrimSpace(title.Current)
	}
	body, held := facts.Field(field.Body)
	if !held {
		body, held = facts.Field(field.ProductDescription)
	}
	if !held {
		return standing
	}
	doc, err := content.Parse(body.Current)
	if err != nil {
		return standing
	}
	for _, heading := range doc.Headings() {
		if text := strings.TrimSpace(content.TextOf(heading)); heading.Data != "h1" && text != "" {
			standing.Sections = append(standing.Sections, content.DraftSection{Heading: text})
		}
	}
	return standing
}

func GenerateMeta(deps Deps) run.StepDef {
	return run.StepDef{
		Name:      NameGenerateMeta,
		Role:      domainllm.RoleEditor,
		Requires:  []run.ArtifactKind{run.ArtifactDraft},
		Scoped:    metaRequires,
		Produces:  []run.ArtifactKind{run.ArtifactMeta},
		Retry:     run.RetryPolicy{Max: 3},
		Timeout:   editorTimeout,
		Price:     run.Price{OutputTokens: metaTokens, Input: packInput(deps, domainllm.RoleEditor, seoTitleKey)},
		Preflight: inputPreflight(deps, domainllm.RoleEditor, seoTitleKey),
		Run: func(ctx context.Context, sc *run.StepContext) (run.Result, error) {
			standing := keepsBody(sc)
			var draft content.ContentDraft
			if !standing {
				read, err := draftOf(sc)
				if err != nil {
					return run.Result{}, err
				}
				draft = read
			}
			entity, err := entityOf(ctx, deps, sc)
			if err != nil {
				return run.Result{}, err
			}
			owner, err := deps.Sites.Get(ctx, sc.Run.SiteID)
			if err != nil {
				return run.Result{}, err
			}
			call, err := modelFor(ctx, deps, sc, NameGenerateMeta, domainllm.RoleEditor)
			if err != nil {
				return run.Result{}, err
			}

			keywords := pagemap.Keywords(sc.Page, entity)
			prompt := metaPrompt{
				Page: sc.Page, Entity: entity, Keywords: keywords, Spec: sc.Spec, Draft: draft, Standing: standing,
				SiteName: owner.Name,
				Pattern: template.Expand(sc.Spec.MetaRules.TitlePattern, template.Vars{
					PrimaryKeyword: keywords.Main(), EntityName: entity.Name, SiteName: owner.Name, PageTitle: sc.Page.Title,
				}),
			}
			seen, err := deps.fitted(ctx, sc, call, field.SEOTitle, nil, metaTokens, func() (string, string, error) {
				return prompts.Render(NameGenerateMeta, prompt)
			}, schemaText[metaAnswer]())
			if err != nil {
				return run.Result{}, err
			}
			prompt.Picture = picture(seen.Fitted, contextpack.SectionTargets)
			if standing {
				draft = asItStands(seen.Facts, sc.Page)
				prompt.Draft = draft
			}
			if description, held := seen.Facts.Field(field.SEODescription); held {
				prompt.Description = contextpack.FieldRules(seen.Facts, description.Scope)
				if current := strings.TrimSpace(description.Current); current != "" {
					prompt.Description = append(prompt.Description, "Its current value: "+current)
				}
			}
			request, err := stepRequest(sc, call, prompt, metaTokens)
			if err != nil {
				return run.Result{}, err
			}

			answer, usage, err := port.Structured[metaAnswer](ctx, deps.LLM, request)
			if err != nil {
				return run.Result{}, err
			}

			settled := settleMeta(answer, draft, sc.Spec.MetaRules, sc.Page)
			blob, err := encode(settled, "page meta")
			if err != nil {
				return run.Result{}, err
			}

			return run.Result{
				Artifacts: []run.Artifact{{Kind: run.ArtifactMeta, Blob: blob}},
				Tokens:    usage.Total,
				Message:   "wrote the search snippet of " + sc.Page.Path,
			}, nil
		},
	}
}

func settleMeta(answer metaAnswer, draft content.ContentDraft, rules template.MetaRules, page pagemap.Page) Meta {
	meta := Meta{
		Title:         strings.TrimSpace(answer.Title),
		Description:   strings.TrimSpace(answer.Description),
		OGTitle:       strings.TrimSpace(answer.OGTitle),
		OGDescription: strings.TrimSpace(answer.OGDescription),
	}

	borrowed := make([]string, 0, 2)
	if meta.Title == "" {
		meta.Title = strings.TrimSpace(draft.Title)
		borrowed = append(borrowed, "title")
	}
	if meta.Description == "" {
		meta.Description = strings.TrimSpace(draft.Summary)
		borrowed = append(borrowed, "description")
	}
	if meta.OGTitle == "" {
		meta.OGTitle = meta.Title
	}
	meta.Description = clip(meta.Description, rules.DescriptionMax)
	if meta.OGDescription == "" {
		meta.OGDescription = meta.Description
	}
	meta.OGDescription = clip(meta.OGDescription, rules.DescriptionMax)

	if len(borrowed) > 0 {
		meta.Findings = []content.Finding{{
			Severity: content.SeverityWarn,
			Code:     CodeMetaNotWritten,
			Message: "the model returned no " + strings.Join(borrowed, " and no ") + " for " + page.Path +
				", so the search snippet repeats what the page already says",
			Details: map[string]any{"pageId": page.ID, "path": page.Path, "fields": borrowed},
		}}
	}
	return meta
}

func clip(text string, limit int) string {
	if limit <= 0 {
		return text
	}

	runes := []rune(text)
	if len(runes) <= limit {
		return text
	}

	cut := string(runes[:limit-1])
	if space := strings.LastIndexByte(cut, ' '); space > 0 {
		cut = cut[:space]
	}
	return strings.TrimRight(cut, " ,;:-") + ellipsis
}
