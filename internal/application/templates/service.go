package templates

import (
	"context"
	"time"

	"github.com/davidmovas/postulator/internal/application"
	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/site"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/dto"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

type templateStore interface {
	Insert(ctx context.Context, t template.Template) error
	Update(ctx context.Context, t template.Template) error
	Delete(ctx context.Context, id string) error
	Get(ctx context.Context, id string) (template.Template, error)
	List(ctx context.Context, q template.Query, page paging.Request) (paging.List[template.Template], error)
	UpsertOverride(ctx context.Context, o template.Override) (template.Override, error)
	GetOverride(ctx context.Context, templateID string, scope template.OverrideScope, targetID string) (template.Override, error)
	DeleteOverride(ctx context.Context, id string) error
	ListOverrides(ctx context.Context, templateID string) ([]template.Override, error)
}

type policyStore interface {
	Insert(ctx context.Context, p template.LinkPolicy) error
	Update(ctx context.Context, p template.LinkPolicy) error
	Delete(ctx context.Context, id string) error
	Get(ctx context.Context, id string) (template.LinkPolicy, error)
	List(ctx context.Context, q template.PolicyQuery, page paging.Request) (paging.List[template.LinkPolicy], error)
}

type pageReader interface {
	Get(ctx context.Context, id string) (pagemap.Page, error)
}

type siteReader interface {
	Get(ctx context.Context, id string) (site.Site, error)
}

type unitOfWork interface {
	Do(ctx context.Context, fn func(context.Context) error) error
}

type Service struct {
	templates templateStore
	policies  policyStore
	pages     pageReader
	sites     siteReader
	uow       unitOfWork
	publisher application.Publisher
	clock     clock.Clock
}

func New(templates templateStore, policies policyStore, pages pageReader, sites siteReader, uow unitOfWork, publisher application.Publisher, clk clock.Clock) *Service {
	return &Service{templates: templates, policies: policies, pages: pages, sites: sites, uow: uow, publisher: publisher, clock: clk}
}

func (s *Service) now() time.Time {
	return s.clock.Now().UTC().Truncate(time.Second)
}

func (s *Service) changed() error {
	return s.publisher.Publish(events.TemplatesChanged, events.TemplatesChangedPayload{})
}

func scopeOf(raw string, siteID *string) (scope template.Scope, err error) {
	scope = template.Scope(raw)
	if raw == "" {
		scope = template.ScopeGlobal
		if siteID != nil {
			scope = template.ScopeSite
		}
	}
	if !scope.Valid() {
		return "", errors.New(errors.Invalid, "scope is not recognized").WithDetail("field", "scope")
	}
	return scope, nil
}

func (s *Service) requireTarget(ctx context.Context, scope template.Scope, siteID *string) error {
	if scope != template.ScopeSite {
		return nil
	}
	if siteID == nil || *siteID == "" {
		return errors.New(errors.Invalid, "a site record needs a site id").WithDetail("field", "siteId")
	}
	_, err := s.sites.Get(ctx, *siteID)
	return err
}

func sortOf(sort *dto.Sort) (key template.Sort, desc bool, err error) {
	if sort == nil {
		return template.SortCreatedAt, false, nil
	}
	key = template.Sort(sort.Field)
	if !key.Valid() {
		return "", false, errors.New(errors.Invalid, "templates and policies are sorted by createdAt or name").WithDetail("field", "sort.field")
	}
	return key, sort.Desc, nil
}

func scopeFilter(raw string) (*template.Scope, error) {
	if raw == "" {
		return nil, nil
	}
	scope := template.Scope(raw)
	if !scope.Valid() {
		return nil, errors.New(errors.Invalid, "scope is not recognized").WithDetail("field", "scope")
	}
	return &scope, nil
}

func siteFilter(raw string) *string {
	if raw == "" {
		return nil
	}
	return &raw
}
