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

	metaTokens         = 512
	primaryPlaceholder = "{primaryKeyword}"
	sitePlaceholder    = "{siteName}"
	ellipsis           = "…"
)

type Meta struct {
	Title         string `json:"title" description:"The meta title of the page, following the title pattern"`
	Description   string `json:"description" description:"The meta description, one sentence that earns the click"`
	Canonical     string `json:"canonical" description:"The absolute canonical URL of the page"`
	OGTitle       string `json:"ogTitle" description:"The Open Graph title, which may be shorter than the meta title"`
	OGDescription string `json:"ogDescription" description:"The Open Graph description"`
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
				Pattern:   titlePattern(sc.Spec.MetaRules.TitlePattern, entity.PrimaryKeyword, owner.Name),
			})
			if err != nil {
				return run.Result{}, err
			}

			meta, usage, err := port.Structured[Meta](ctx, deps.LLM, port.Request{
				Ref:       ref,
				System:    system,
				Messages:  []port.Message{{Role: port.RoleUser, Text: user}},
				MaxTokens: metaTokens,
				Meta:      callMeta(sc, NameGenerateMeta),
			})
			if err != nil {
				return run.Result{}, err
			}

			settled := settleMeta(meta, draft, canonical, sc.Spec.MetaRules)
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

func titlePattern(pattern, primary, siteName string) string {
	if pattern == "" {
		return ""
	}
	return strings.NewReplacer(primaryPlaceholder, primary, sitePlaceholder, siteName).Replace(pattern)
}

func settleMeta(meta Meta, draft content.ContentDraft, canonical string, rules template.MetaRules) Meta {
	meta.Title = strings.TrimSpace(meta.Title)
	meta.Description = strings.TrimSpace(meta.Description)
	meta.Canonical = strings.TrimSpace(meta.Canonical)
	meta.OGTitle = strings.TrimSpace(meta.OGTitle)
	meta.OGDescription = strings.TrimSpace(meta.OGDescription)

	if meta.Title == "" {
		meta.Title = strings.TrimSpace(draft.Title)
	}
	if meta.Description == "" {
		meta.Description = strings.TrimSpace(draft.Summary)
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
