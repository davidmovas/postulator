package steps

import (
	"context"
	"encoding/json"
	"time"

	"github.com/davidmovas/postulator/internal/application/llm"
	"github.com/davidmovas/postulator/internal/application/templates"
	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/graph"
	domainllm "github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	CheckpointLinks = "links"

	wordsPerToken     = 1.6
	tokenHeadroom     = 512
	fallbackMaxTokens = 2048
	stepTimeout       = 4 * time.Minute
	pureStepTimeout   = 30 * time.Second
)

type entityReader interface {
	ListBySite(ctx context.Context, siteID string) ([]graph.Entity, error)
}

type edgeReader interface {
	ListBySite(ctx context.Context, siteID string) ([]graph.Edge, error)
}

type pageReader interface {
	ListBySite(ctx context.Context, siteID string) ([]pagemap.Page, error)
}

type policyReader interface {
	GetEffectivePolicy(ctx context.Context, req templates.GetEffectivePolicyRequest) (templates.GetEffectivePolicyResponse, error)
}

type profileResolver interface {
	Resolve(ctx context.Context, siteID string, role domainllm.Role, templateProfiles map[domainllm.Role]domainllm.ModelRef) (domainllm.ModelRef, error)
}

type Deps struct {
	Entities entityReader
	Edges    edgeReader
	Pages    pageReader
	Policies policyReader
	Profiles profileResolver
	LLM      llm.Client
}

func All(deps Deps) []run.StepDef {
	return []run.StepDef{
		ResolveContext(deps),
		GenerateBody(deps),
		InsertLinks(deps),
		RepairLinks(deps),
		Validate(deps),
	}
}

func Register(registry *run.Registry, deps Deps) error {
	for _, def := range All(deps) {
		if err := registry.Register(def); err != nil {
			return err
		}
	}
	return nil
}

func effectivePolicy(ctx context.Context, deps Deps, sc *run.StepContext) (template.LinkPolicy, error) {
	resp, err := deps.Policies.GetEffectivePolicy(ctx, templates.GetEffectivePolicyRequest{SiteID: sc.Run.SiteID})
	if err != nil {
		return template.LinkPolicy{}, err
	}

	return template.LinkPolicy{
		Rules:          sc.Spec.LinkRules,
		ForbidExternal: resp.Policy.ForbidExternal,
		ForbidSelf:     resp.Policy.ForbidSelf,
		AnchorStrategy: template.AnchorStrategy(resp.Policy.AnchorStrategy),
	}, nil
}

func linkContextOf(sc *run.StepContext) (content.LinkContext, error) {
	artifact, err := sc.Artifact(run.ArtifactLinkContext)
	if err != nil {
		return content.LinkContext{}, err
	}

	var lc content.LinkContext
	if unmarshalErr := json.Unmarshal(artifact.Blob, &lc); unmarshalErr != nil {
		return content.LinkContext{}, errors.Wrap(unmarshalErr, errors.Internal, "the stored link context is not readable")
	}
	return lc, nil
}

func bodyOf(sc *run.StepContext) (*content.Document, error) {
	artifact, err := sc.Artifact(run.ArtifactBodyHTML)
	if err != nil {
		return nil, err
	}
	return content.Parse(string(artifact.Blob))
}

func entityOf(ctx context.Context, deps Deps, sc *run.StepContext) (graph.Entity, error) {
	if sc.Page.EntityID == nil {
		return graph.Entity{}, errors.New(errors.Invalid, "the page is not mapped to an entity, so it has no graph context").
			WithDetail("pageId", sc.Page.ID)
	}

	entities, err := deps.Entities.ListBySite(ctx, sc.Run.SiteID)
	if err != nil {
		return graph.Entity{}, err
	}
	for i := range entities {
		if entities[i].ID == *sc.Page.EntityID {
			return entities[i], nil
		}
	}
	return graph.Entity{}, errors.New(errors.NotFound, "the entity the page is mapped to is gone").
		WithDetail("entityId", *sc.Page.EntityID)
}

func encode(value any, what string) ([]byte, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, errors.Wrap(err, errors.Internal, "encode the "+what)
	}
	return encoded, nil
}

func maxTokens(spec template.TemplateSpec) int {
	words := 0
	for i := range spec.Sections {
		words += spec.Sections[i].TargetWords
	}
	if words == 0 {
		words = spec.Length.Max
	}
	if words == 0 {
		return fallbackMaxTokens
	}
	return int(float64(words)*wordsPerToken) + tokenHeadroom
}

func callMeta(sc *run.StepContext, step string) llm.CallMeta {
	return llm.CallMeta{RunID: sc.Run.ID, ItemID: sc.Item.ID, Step: step}
}
