package pagetypes

import (
	"context"
	"strings"
	"time"

	"github.com/davidmovas/postulator/internal/application"
	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
	"github.com/davidmovas/postulator/internal/domain/site"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

const (
	reasonLastOfPreset     = "last_of_preset"
	reasonReadOnlyRole     = "role_read_only"
	reasonExcerptOnProduct = "excerpt_is_short_description"
	reasonPersonOnly       = "field_not_ai_written"
)

type typeStore interface {
	Insert(ctx context.Context, t pagetype.Type) error
	Update(ctx context.Context, t pagetype.Type) error
	Delete(ctx context.Context, id string) error
	Get(ctx context.Context, id string) (pagetype.Type, error)
	ListBySite(ctx context.Context, siteID string) ([]pagetype.Type, error)
}

type scopeStore interface {
	Put(ctx context.Context, s field.Scope) error
	ListBySite(ctx context.Context, siteID string) ([]field.Scope, error)
}

type pageStore interface {
	ListBySite(ctx context.Context, siteID string) ([]pagemap.Page, error)
	ReassignType(ctx context.Context, fromTypeID, toTypeID string, reason pagemap.TypeReason, at time.Time) (int64, error)
}

type templateReader interface {
	Get(ctx context.Context, id string) (template.Template, error)
	List(ctx context.Context, q template.Query, page paging.Request) (paging.List[template.Template], error)
}

type siteReader interface {
	Get(ctx context.Context, id string) (site.Site, error)
}

type unitOfWork interface {
	Do(ctx context.Context, fn func(context.Context) error) error
}

type Deps struct {
	Types      typeStore
	Scopes     scopeStore
	Pages      pageStore
	Templates  templateReader
	Sites      siteReader
	UnitOfWork unitOfWork
	Publisher  application.Publisher
	Clock      clock.Clock
}

type Service struct {
	deps Deps
}

func New(deps Deps) *Service {
	return &Service{deps: deps}
}

func (s *Service) now() time.Time {
	return s.deps.Clock.Now().UTC().Truncate(time.Second)
}

func (s *Service) changed(siteID string) error {
	return s.deps.Publisher.Publish(events.SitesChanged, events.SitesChangedPayload{SiteID: siteID})
}

func (s *Service) Seed(ctx context.Context, siteID string) error {
	return s.deps.UnitOfWork.Do(ctx, func(c context.Context) error {
		existing, err := s.deps.Types.ListBySite(c, siteID)
		if err != nil || len(existing) > 0 {
			return err
		}
		owner, err := s.deps.Sites.Get(c, siteID)
		if err != nil {
			return err
		}
		now := s.now()
		for i, preset := range pagetype.Presets() {
			var templateID *string
			if owner.Defaults.TemplateID == nil {
				if templateID, err = s.builtIn(c, preset); err != nil {
					return err
				}
			}
			kind := pagetype.Type{
				ID: id.New(), SiteID: siteID, Name: preset.DefaultName(), Preset: preset, TemplateID: templateID,
				CreatedAt: now.Add(time.Duration(i) * time.Second), UpdatedAt: now,
			}
			if insertErr := s.insert(c, kind, now); insertErr != nil {
				return insertErr
			}
		}
		return nil
	})
}

func (s *Service) insert(ctx context.Context, kind pagetype.Type, now time.Time) error {
	if err := kind.Validate(); err != nil {
		return err
	}
	if err := s.deps.Types.Insert(ctx, kind); err != nil {
		return err
	}
	scopes := pagetype.DefaultScopes(kind.Preset, kind.ID)
	for i := range scopes {
		scopes[i].UpdatedAt = now
		if err := s.deps.Scopes.Put(ctx, scopes[i]); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) Set(ctx context.Context, siteID string) (pagetype.Set, error) {
	types, err := s.deps.Types.ListBySite(ctx, siteID)
	if err != nil {
		return pagetype.Set{}, err
	}
	scopes, err := s.deps.Scopes.ListBySite(ctx, siteID)
	if err != nil {
		return pagetype.Set{}, err
	}
	return pagetype.NewSet(types, scopes), nil
}

func (s *Service) List(ctx context.Context, req ListRequest) (ListResponse, error) {
	if strings.TrimSpace(req.SiteID) == "" {
		return ListResponse{}, errors.New(errors.Invalid, "a site is required").WithDetail("field", "siteId")
	}
	types, err := s.views(ctx, req.SiteID)
	if err != nil {
		return ListResponse{}, err
	}
	return ListResponse{Types: types, Fields: catalogView()}, nil
}

func (s *Service) Get(ctx context.Context, req GetRequest) (Type, error) {
	kind, err := s.deps.Types.Get(ctx, req.ID)
	if err != nil {
		return Type{}, err
	}
	return s.view(ctx, kind)
}

func (s *Service) views(ctx context.Context, siteID string) ([]Type, error) {
	types, err := s.deps.Types.ListBySite(ctx, siteID)
	if err != nil {
		return nil, err
	}
	scopes, err := s.deps.Scopes.ListBySite(ctx, siteID)
	if err != nil {
		return nil, err
	}
	pages, err := s.deps.Pages.ListBySite(ctx, siteID)
	if err != nil {
		return nil, err
	}
	set := pagetype.NewSet(types, scopes)
	counts := make(map[string]int, len(types))
	for i := range pages {
		if pages[i].Status != pagemap.StatusArchived {
			counts[set.TypeOf(pages[i]).ID]++
		}
	}
	out := make([]Type, 0, len(types))
	for i := range types {
		out = append(out, typeView(types[i], set, scopes, counts[types[i].ID]))
	}
	return out, nil
}

func (s *Service) view(ctx context.Context, kind pagetype.Type) (Type, error) {
	views, err := s.views(ctx, kind.SiteID)
	if err != nil {
		return Type{}, err
	}
	for i := range views {
		if views[i].ID == kind.ID {
			return views[i], nil
		}
	}
	return Type{}, errors.New(errors.NotFound, "page type not found").WithDetail("pageTypeId", kind.ID)
}

func (s *Service) Create(ctx context.Context, req CreateRequest) (Type, error) {
	now := s.now()
	kind := pagetype.Type{
		ID: id.New(), SiteID: req.SiteID, Name: strings.TrimSpace(req.Name), Preset: pagetype.Preset(req.Preset),
		CreatedAt: now, UpdatedAt: now,
	}
	err := s.deps.UnitOfWork.Do(ctx, func(c context.Context) error {
		templateID, templateErr := s.templateFor(c, req.SiteID, req.TemplateID)
		if req.TemplateID == nil {
			templateID, templateErr = s.builtIn(c, kind.Preset)
		}
		if templateErr != nil {
			return templateErr
		}
		kind.TemplateID = templateID
		return s.insert(c, kind, now)
	})
	if err != nil {
		return Type{}, err
	}
	if publishErr := s.changed(kind.SiteID); publishErr != nil {
		return Type{}, publishErr
	}
	return s.view(ctx, kind)
}

func (s *Service) Save(ctx context.Context, req SaveRequest) (Type, error) {
	var saved pagetype.Type
	err := s.deps.UnitOfWork.Do(ctx, func(c context.Context) error {
		kind, getErr := s.deps.Types.Get(c, req.ID)
		if getErr != nil {
			return getErr
		}
		if req.Name != nil {
			kind.Name = strings.TrimSpace(*req.Name)
		}
		if req.TemplateID != nil {
			templateID, templateErr := s.templateFor(c, kind.SiteID, req.TemplateID)
			if templateErr != nil {
				return templateErr
			}
			kind.TemplateID = templateID
		}
		kind.UpdatedAt = s.now()
		if validateErr := kind.Validate(); validateErr != nil {
			return validateErr
		}
		saved = kind
		return s.deps.Types.Update(c, kind)
	})
	if err != nil {
		return Type{}, err
	}
	if publishErr := s.changed(saved.SiteID); publishErr != nil {
		return Type{}, publishErr
	}
	return s.view(ctx, saved)
}

func (s *Service) builtIn(ctx context.Context, preset pagetype.Preset) (*string, error) {
	name := preset.DefaultTemplate()
	if name == "" {
		return nil, nil
	}
	global := template.ScopeGlobal
	shipped, err := s.deps.Templates.List(ctx, template.Query{Scope: &global, Name: name, Sort: template.SortCreatedAt}, paging.Request{Limit: 1})
	if err != nil || len(shipped.Items) == 0 {
		return nil, err
	}
	return &shipped.Items[0].ID, nil
}

func (s *Service) templateFor(ctx context.Context, siteID string, requested *string) (*string, error) {
	if requested == nil || strings.TrimSpace(*requested) == "" {
		return nil, nil
	}
	found, err := s.deps.Templates.Get(ctx, strings.TrimSpace(*requested))
	if err != nil {
		return nil, err
	}
	if found.Scope == template.ScopeSite && (found.SiteID == nil || *found.SiteID != siteID) {
		return nil, errors.New(errors.Invalid, "the template "+found.Name+" belongs to another site").WithDetail("field", "templateId")
	}
	return &found.ID, nil
}

func (s *Service) Delete(ctx context.Context, req DeleteRequest) (DeleteResponse, error) {
	var (
		moved  int64
		siteID string
	)
	err := s.deps.UnitOfWork.Do(ctx, func(c context.Context) error {
		kind, getErr := s.deps.Types.Get(c, req.ID)
		if getErr != nil {
			return getErr
		}
		siteID = kind.SiteID
		if strings.TrimSpace(req.ReassignTo) == "" || req.ReassignTo == kind.ID {
			return errors.New(errors.Invalid, "the type's pages need another type of the site").WithDetail("field", "reassignTo")
		}
		target, getErr := s.deps.Types.Get(c, req.ReassignTo)
		if getErr != nil {
			return getErr
		}
		if target.SiteID != kind.SiteID {
			return errors.New(errors.Invalid, "the pages can only move to a type of their own site").WithDetail("field", "reassignTo")
		}
		siblings, listErr := s.deps.Types.ListBySite(c, kind.SiteID)
		if listErr != nil {
			return listErr
		}
		if !hasAnother(siblings, kind) {
			return errors.New(errors.Invalid, "the last type of the "+kind.Preset.DefaultName()+" preset cannot be deleted").
				WithDetail("field", "id").WithDetail("reason", reasonLastOfPreset)
		}
		var reassignErr error
		if moved, reassignErr = s.deps.Pages.ReassignType(c, kind.ID, target.ID, pagemap.TypeByUser, s.now()); reassignErr != nil {
			return reassignErr
		}
		return s.deps.Types.Delete(c, kind.ID)
	})
	if err != nil {
		return DeleteResponse{}, err
	}
	if publishErr := s.changed(siteID); publishErr != nil {
		return DeleteResponse{}, publishErr
	}
	if publishErr := s.deps.Publisher.Publish(events.PagesChanged, events.PagesChangedPayload{SiteID: siteID}); publishErr != nil {
		return DeleteResponse{}, publishErr
	}
	return DeleteResponse{Moved: int(moved)}, nil
}

func hasAnother(types []pagetype.Type, kind pagetype.Type) bool {
	for i := range types {
		if types[i].ID != kind.ID && types[i].Preset == kind.Preset {
			return true
		}
	}
	return false
}

func (s *Service) SetScope(ctx context.Context, req SetScopeRequest) (Scope, error) {
	scope := field.Scope{
		PageTypeID: req.PageTypeID, Key: field.Key(strings.TrimSpace(req.Field)), Enabled: req.Enabled, AIWrites: req.AIWrites,
		Instruction: strings.TrimSpace(req.Instruction), MinLength: req.MinLength, MaxLength: req.MaxLength, Rewrites: req.Rewrites,
		Links: req.Links, Label: strings.TrimSpace(req.Label), Provider: field.Provider(req.Provider), UpdatedAt: s.now(),
	}
	var siteID string
	err := s.deps.UnitOfWork.Do(ctx, func(c context.Context) error {
		kind, getErr := s.deps.Types.Get(c, req.PageTypeID)
		if getErr != nil {
			return getErr
		}
		siteID = kind.SiteID
		if validateErr := scope.Validate(); validateErr != nil {
			return validateErr
		}
		if scope.Enabled && !kind.Preset.Writable() {
			return errors.New(errors.Invalid, "a "+kind.Preset.DefaultName()+" page is read only, so no field of it is written").
				WithDetail("field", "enabled").WithDetail("reason", reasonReadOnlyRole)
		}
		if spec, known := field.Lookup(scope.Key); known && !spec.AIWritable() && scope.AIWrites {
			return errors.New(errors.Invalid, "the "+string(scope.Key)+" is set by the site or a person, never written by the AI").
				WithDetail("field", "aiWrites").WithDetail("reason", reasonPersonOnly)
		}
		if scope.Key == field.Excerpt && kind.Preset == pagetype.Product {
			return errors.New(errors.Invalid, "a product's excerpt is its short description, which has a scope of its own").
				WithDetail("field", "field").WithDetail("reason", reasonExcerptOnProduct)
		}
		return s.deps.Scopes.Put(c, scope)
	})
	if err != nil {
		return Scope{}, err
	}
	if publishErr := s.changed(siteID); publishErr != nil {
		return Scope{}, publishErr
	}
	return scopeView(scope, true), nil
}
