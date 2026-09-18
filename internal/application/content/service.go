package content

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/llm"
	"github.com/davidmovas/postulator/internal/application/templates"
	contentdomain "github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/graph"
	domainllm "github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
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

type specResolver interface {
	ResolveForPage(ctx context.Context, req templates.ResolveForPageRequest) (templates.ResolveForPageResponse, error)
}

type policyReader interface {
	GetEffectivePolicy(ctx context.Context, req templates.GetEffectivePolicyRequest) (templates.GetEffectivePolicyResponse, error)
}

type profileResolver interface {
	Resolve(ctx context.Context, siteID string, role domainllm.Role, templateProfiles map[domainllm.Role]domainllm.ModelRef) (domainllm.ModelRef, error)
}

type rawReader interface {
	RawContent(ctx context.Context, siteID string, wpID int64) (string, error)
}

type Deps struct {
	Pages    pageStore
	Entities entityStore
	Edges    edgeStore
	Specs    specResolver
	Policies policyReader
	Profiles profileResolver
	Raw      rawReader
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
	Page    pagemap.Page
	Entity  graph.Entity
	Spec    template.TemplateSpec
	Body    string
	Meta    Snippet
	HasMeta bool
	Targets []contentdomain.LinkTarget
}
