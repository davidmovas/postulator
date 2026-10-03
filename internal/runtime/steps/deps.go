package steps

import (
	"context"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/images"
	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/application"
	appcontent "github.com/davidmovas/postulator/internal/application/content"
	"github.com/davidmovas/postulator/internal/application/llm"
	"github.com/davidmovas/postulator/internal/application/templates"
	"github.com/davidmovas/postulator/internal/domain/graph"
	domainllm "github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/site"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type entityReader interface {
	ListBySite(ctx context.Context, siteID string) ([]graph.Entity, error)
}

type edgeReader interface {
	ListBySite(ctx context.Context, siteID string) ([]graph.Edge, error)
}

type termStore interface {
	ListBySite(ctx context.Context, siteID string) ([]graph.Term, error)
	Upsert(ctx context.Context, t graph.Term) error
	Delete(ctx context.Context, entityID string, taxonomy graph.Taxonomy) error
}

type pageStore interface {
	ListBySite(ctx context.Context, siteID string) ([]pagemap.Page, error)
	Get(ctx context.Context, id string) (pagemap.Page, error)
	Insert(ctx context.Context, page pagemap.Page) error
	Update(ctx context.Context, page pagemap.Page) error
}

type linkStore interface {
	ReplaceForPage(ctx context.Context, pageID string, links []pagemap.PageLink) error
	ListForPage(ctx context.Context, pageID string) ([]pagemap.PageLink, error)
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

type templateReader interface {
	GetEffectivePolicy(ctx context.Context, req templates.GetEffectivePolicyRequest) (templates.GetEffectivePolicyResponse, error)
	ResolveForPage(ctx context.Context, req templates.ResolveForPageRequest) (templates.ResolveForPageResponse, error)
}

type profileResolver interface {
	Resolve(ctx context.Context, siteID string, role domainllm.Role, templateProfiles map[domainllm.Role]domainllm.ModelRef) (domainllm.ModelRef, error)
}

type judgeService interface {
	Assess(ctx context.Context, req appcontent.AssessRequest) (appcontent.AssessResponse, error)
}

type itemReader interface {
	ByRun(ctx context.Context, runID string) ([]run.Item, error)
}

type artifactReader interface {
	ByItem(ctx context.Context, itemID string) ([]run.Artifact, error)
}

type ImageProvider interface {
	Generate(ctx context.Context, prompt images.Prompt) (images.Image, error)
}

type ImageSource interface {
	Pick(ctx context.Context, query images.Query) ([]images.Image, error)
}

type Deps struct {
	Entities      entityReader
	Edges         edgeReader
	Terms         termStore
	Pages         pageStore
	Links         linkStore
	Items         itemReader
	Artifacts     artifactReader
	Sites         siteReader
	SiteWriter    siteWriter
	WordPress     siteClients
	Policies      templateReader
	Profiles      profileResolver
	Content       judgeService
	LLM           llm.Client
	ImageProvider ImageProvider
	ImageModel    *domainllm.ModelRef
	ImageSources  map[template.ImageSource]ImageSource
	UnitOfWork    unitOfWork
	Publisher     application.Publisher
	Clock         clock.Clock
	BatchSize     int
}

func (d Deps) now() time.Time {
	if d.Clock == nil {
		return time.Time{}
	}
	return d.Clock.Now().UTC().Truncate(time.Second)
}

func (d Deps) categoryStores() error {
	if err := d.entityReader(); err != nil {
		return err
	}
	return d.termStore()
}

func (d Deps) entityReader() error {
	if d.Entities == nil {
		return errors.New(errors.Internal, "the run steps were given no entity reader, so no category chain can be read")
	}
	return nil
}

func (d Deps) termStore() error {
	if d.Terms == nil {
		return errors.New(errors.Internal, "the run steps were given no term store, so no category can be kept")
	}
	return nil
}

func (d Deps) inUnit(ctx context.Context, apply func(context.Context) error) error {
	if d.UnitOfWork == nil {
		return apply(ctx)
	}
	return d.UnitOfWork.Do(ctx, apply)
}
