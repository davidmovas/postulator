package content

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/llm"
	"github.com/davidmovas/postulator/internal/application/sitemodel"
	"github.com/davidmovas/postulator/internal/application/templates"
	contentdomain "github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/keyword"
	domainllm "github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
	"github.com/davidmovas/postulator/internal/domain/template"
)

type pageStore interface {
	Get(ctx context.Context, id string) (pagemap.Page, error)
	ListBySite(ctx context.Context, siteID string) ([]pagemap.Page, error)
}

type entityStore interface {
	ListBySite(ctx context.Context, siteID string) ([]graph.Entity, error)
}

type edgeStore interface {
	ListBySite(ctx context.Context, siteID string) ([]graph.Edge, error)
}

type fieldStore interface {
	ListBySite(ctx context.Context, siteID string) (map[string][]field.Value, error)
}

type specResolver interface {
	ResolveForPage(ctx context.Context, req templates.ResolveForPageRequest) (templates.ResolveForPageResponse, error)
}

type typeSets interface {
	Set(ctx context.Context, siteID string) (pagetype.Set, error)
}

type profileResolver interface {
	Resolve(ctx context.Context, siteID string, role domainllm.Role, templateProfiles map[domainllm.Role]domainllm.ModelRef) (domainllm.ModelRef, error)
}

type rawReader interface {
	RawContent(ctx context.Context, siteID string, wpID int64, wpType string) (string, error)
}

type pictures interface {
	Picture(ctx context.Context, req sitemodel.PackRequest) (sitemodel.Picture, error)
}

type Deps struct {
	Pages    pageStore
	Entities entityStore
	Edges    edgeStore
	Fields   fieldStore
	Specs    specResolver
	Types    typeSets
	Profiles profileResolver
	Raw      rawReader
	Pictures pictures
	LLM      llm.Client
}

type Service struct {
	deps Deps
}

func New(deps Deps) *Service {
	return &Service{deps: deps}
}

type Snippet struct {
	Title       string
	Description string
}

type judgePrompt struct {
	Page     pagemap.Page
	Entity   graph.Entity
	Keywords keyword.List
	Spec     template.TemplateSpec
	Body     string
	Meta     Snippet
	HasMeta  bool
	Store    bool
	Short    string
	Targets  []contentdomain.LinkTarget
	Picture  string
}
