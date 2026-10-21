package steps

import (
	"context"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/images"
	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/application"
	appcontent "github.com/davidmovas/postulator/internal/application/content"
	"github.com/davidmovas/postulator/internal/application/llm"
	"github.com/davidmovas/postulator/internal/application/observe"
	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/graph"
	domainllm "github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/site"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/clock"
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
	Insert(ctx context.Context, page pagemap.Page) error
	Update(ctx context.Context, page pagemap.Page) error
	UpdatePublished(ctx context.Context, page pagemap.Page) error
}

type linkStore interface {
	ReplaceForPage(ctx context.Context, pageID string, source pagemap.LinkSource, links []pagemap.PageLink) error
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

type bodyRecorder interface {
	Record(ctx context.Context, pageID, raw string, at time.Time) error
	Get(ctx context.Context, pageID string) (current, base block.Snapshot, found bool, err error)
	ReplaceWritten(ctx context.Context, pageID string, hashes []string, runID string, at time.Time) error
	ListWritten(ctx context.Context, pageID string) ([]block.Written, error)
}

type fieldRecorder interface {
	Record(ctx context.Context, pageID string, key field.Key, value, runID string, at time.Time) error
	ListForPage(ctx context.Context, pageID string) ([]field.Value, error)
	ListBySite(ctx context.Context, siteID string) (map[string][]field.Value, error)
}

type typeSets interface {
	Set(ctx context.Context, siteID string) (pagetype.Set, error)
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
	Pages         pageStore
	Links         linkStore
	Items         itemReader
	Artifacts     artifactReader
	Sites         siteReader
	SiteWriter    siteWriter
	WordPress     siteClients
	Types         typeSets
	Profiles      profileResolver
	Tuning        stepTuning
	Models        modelInfo
	Pictures      pictures
	Content       judgeService
	LLM           llm.Client
	ImageProvider ImageProvider
	ImageModel    *domainllm.ModelRef
	ImageSources  map[template.ImageSource]ImageSource
	Observe       *observe.Service
	Bodies        bodyRecorder
	Fields        fieldRecorder
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

func (d Deps) namedIndex(ctx context.Context, siteID string, pages []pagemap.Page) (pagemap.Index, error) {
	if d.Fields == nil {
		return pagemap.NewIndex(pages), nil
	}
	stored, err := d.Fields.ListBySite(ctx, siteID)
	if err != nil {
		return pagemap.Index{}, err
	}
	return pagemap.NewIndexNamed(pages, pagemap.Names(stored)), nil
}

func (d Deps) drifted(ctx context.Context, page pagemap.Page) (bool, error) {
	if d.Bodies == nil {
		return false, nil
	}
	current, base, found, err := d.Bodies.Get(ctx, page.ID)
	if err != nil {
		return false, err
	}
	return found && current.Hash != base.Hash, nil
}

func (d Deps) baseHash(ctx context.Context, page pagemap.Page) (string, error) {
	if d.Bodies == nil {
		return "", nil
	}
	_, base, found, err := d.Bodies.Get(ctx, page.ID)
	if err != nil || !found {
		return "", err
	}
	return base.Hash, nil
}

func (d Deps) storeName(ctx context.Context, page pagemap.Page) (string, error) {
	if d.Fields == nil || page.WPType != pagemap.WPProduct {
		return "", nil
	}
	values, err := d.Fields.ListForPage(ctx, page.ID)
	if err != nil {
		return "", err
	}
	for i := range values {
		if values[i].Key == field.Title {
			return values[i].Current, nil
		}
	}
	return "", nil
}

func (d Deps) inUnit(ctx context.Context, apply func(context.Context) error) error {
	if d.UnitOfWork == nil {
		return apply(ctx)
	}
	return d.UnitOfWork.Do(ctx, apply)
}
