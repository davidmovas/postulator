package steps

import (
	"context"
	"encoding/json"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/application/llm"
	"github.com/davidmovas/postulator/internal/application/templates"
	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/graph"
	domainllm "github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/site"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	checkpointLinks = "links"

	wordsPerToken     = 1.6
	tokenHeadroom     = 512
	fallbackMaxTokens = 2048
	pureStepTimeout   = 30 * time.Second
)

type entityReader interface {
	ListBySite(ctx context.Context, siteID string) ([]graph.Entity, error)
}

type edgeReader interface {
	ListBySite(ctx context.Context, siteID string) ([]graph.Edge, error)
}

type pageStore interface {
	ListBySite(ctx context.Context, siteID string) ([]pagemap.Page, error)
	Get(ctx context.Context, id string) (pagemap.Page, error)
	Update(ctx context.Context, page pagemap.Page) error
}

type linkStore interface {
	ReplaceForPage(ctx context.Context, pageID string, links []pagemap.PageLink) error
}

type siteReader interface {
	Get(ctx context.Context, id string) (site.Site, error)
}

type siteWriter interface {
	Update(ctx context.Context, record site.Site) error
}

type siteClients interface {
	Client(ctx context.Context, siteID string) (*wp.Client, error)
}

type unitOfWork interface {
	Do(ctx context.Context, fn func(context.Context) error) error
}

type policyReader interface {
	GetEffectivePolicy(ctx context.Context, req templates.GetEffectivePolicyRequest) (templates.GetEffectivePolicyResponse, error)
}

type profileResolver interface {
	Resolve(ctx context.Context, siteID string, role domainllm.Role, templateProfiles map[domainllm.Role]domainllm.ModelRef) (domainllm.ModelRef, error)
}

type Deps struct {
	Entities      entityReader
	Edges         edgeReader
	Pages         pageStore
	Links         linkStore
	Sites         siteReader
	SiteWriter    siteWriter
	WordPress     siteClients
	Policies      policyReader
	Profiles      profileResolver
	LLM           llm.Client
	ImageProvider ImageProvider
	ImageSources  map[template.ImageSource]ImageSource
	UnitOfWork    unitOfWork
	Clock         clock.Clock
}

func all(deps Deps) []run.StepDef {
	return []run.StepDef{
		ResolveContext(deps),
		GenerateBody(deps),
		GenerateMeta(deps),
		InsertLinks(deps),
		RepairLinks(deps),
		GenerateImages(deps),
		Validate(deps),
		Judge(deps),
		Publish(deps),
	}
}

func Register(registry *run.Registry, deps Deps) error {
	for _, def := range all(deps) {
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

func draftOf(sc *run.StepContext) (content.ContentDraft, error) {
	artifact, err := sc.Artifact(run.ArtifactDraft)
	if err != nil {
		return content.ContentDraft{}, err
	}

	var decoded content.ContentDraft
	if unmarshalErr := json.Unmarshal(artifact.Blob, &decoded); unmarshalErr != nil {
		return content.ContentDraft{}, errors.Wrap(unmarshalErr, errors.Internal, "the stored draft is not readable")
	}
	return decoded, nil
}

func decodeArtifact[T any](sc *run.StepContext, kind run.ArtifactKind) (value T, found bool, err error) {
	artifact, ok := sc.Artifacts[kind]
	if !ok || artifact.Purged || len(artifact.Blob) == 0 {
		return value, false, nil
	}
	if unmarshalErr := json.Unmarshal(artifact.Blob, &value); unmarshalErr != nil {
		return value, false, errors.Wrap(unmarshalErr, errors.Internal, "the stored "+string(kind)+" is not readable")
	}
	return value, true, nil
}
