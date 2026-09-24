package steps

import (
	"context"
	"strings"

	port "github.com/davidmovas/postulator/internal/application/llm"
	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/graph"
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
	Canonical     string `json:"canonical" description:"The absolute canonical URL of the page"`
	OGTitle       string `json:"ogTitle" description:"The Open Graph title, which may be shorter than the meta title"`
	OGDescription string `json:"ogDescription" description:"The Open Graph description"`
}

type Meta struct {
	Title         string            `json:"title"`
	Description   string            `json:"description"`
	Canonical     string            `json:"canonical"`
	OGTitle       string            `json:"ogTitle"`
	OGDescription string            `json:"ogDescription"`
	Findings      []content.Finding `json:"findings,omitempty"`
}

type metaPrompt struct {
	Page      pagemap.Page
	Entity    graph.Entity
	Spec      template.TemplateSpec
	Draft     content.ContentDraft
	SiteName  string
	Canonical string
	Pattern   string
}

func GenerateMeta(deps Deps) run.StepDef {
	return run.StepDef{
		Name:     NameGenerateMeta,
		Role:     domainllm.RoleEditor,
		Requires: []run.ArtifactKind{run.ArtifactDraft},
		Produces: []run.ArtifactKind{run.ArtifactMeta},
		Retry:    run.RetryPolicy{Max: 3},
		Timeout:  editorTimeout,
		Price:    run.Price{OutputTokens: metaTokens},
		Run: func(ctx context.Context, sc *run.StepContext) (run.Result, error) {
			draft, err := draftOf(sc)
			if err != nil {
				return run.Result{}, err
			}
			entity, err := entityOf(ctx, deps, sc)
			if err != nil {
				return run.Result{}, err
			}
			owner, err := deps.Sites.Get(ctx, sc.Run.SiteID)
			if err != nil {
				return run.Result{}, err
			}
			ref, err := deps.Profiles.Resolve(ctx, sc.Run.SiteID, domainllm.RoleEditor, sc.Spec.ModelProfiles)
			if err != nil {
				return run.Result{}, err
			}

			canonical := pagemap.NewSite(owner.BaseURL).URL(sc.Page.Path)
			system, user, err := render(NameGenerateMeta, metaPrompt{
				Page: sc.Page, Entity: entity, Spec: sc.Spec, Draft: draft,
				SiteName:  owner.Name,
				Canonical: canonical,
				Pattern: template.Expand(sc.Spec.MetaRules.TitlePattern, template.Vars{
					PrimaryKeyword: entity.PrimaryKeyword, EntityName: entity.Name, SiteName: owner.Name, PageTitle: sc.Page.Title,
				}),
			})
			if err != nil {
				return run.Result{}, err
			}

			answer, usage, err := port.Structured[metaAnswer](ctx, deps.LLM, port.Request{
				Ref:       ref,
				System:    system,
				Messages:  []port.Message{{Role: port.RoleUser, Text: user}},
				MaxTokens: metaTokens,
				Meta:      callMeta(sc, NameGenerateMeta),
			})
			if err != nil {
				return run.Result{}, err
			}

			settled := settleMeta(answer, draft, canonical, sc.Spec.MetaRules, sc.Page)
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

func settleMeta(answer metaAnswer, draft content.ContentDraft, canonical string, rules template.MetaRules,
	page pagemap.Page) Meta {
	meta := Meta{
		Title:         strings.TrimSpace(answer.Title),
		Description:   strings.TrimSpace(answer.Description),
		Canonical:     strings.TrimSpace(answer.Canonical),
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
	if meta.Canonical == "" {
		meta.Canonical = canonical
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
