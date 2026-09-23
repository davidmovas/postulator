package templates

import (
	"context"
	"strings"

	"github.com/davidmovas/postulator/internal/application"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/id"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

func (s *Service) CreateTemplate(ctx context.Context, req CreateTemplateRequest) (CreateTemplateResponse, error) {
	scope, err := scopeOf(req.Scope, req.SiteID)
	if err != nil {
		return CreateTemplateResponse{}, err
	}
	if targetErr := s.requireTarget(ctx, scope, req.SiteID); targetErr != nil {
		return CreateTemplateResponse{}, targetErr
	}

	now := s.now()
	record := template.Template{
		ID:        id.New(),
		Scope:     scope,
		SiteID:    req.SiteID,
		Name:      strings.TrimSpace(req.Name),
		PageKind:  strings.TrimSpace(req.PageKind),
		Version:   1,
		Spec:      req.Spec,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if scope == template.ScopeGlobal {
		record.SiteID = nil
	}
	if validErr := record.Validate(); validErr != nil {
		return CreateTemplateResponse{}, validErr
	}
	if doErr := s.uow.Do(ctx, func(c context.Context) error { return s.templates.Insert(c, record) }); doErr != nil {
		return CreateTemplateResponse{}, doErr
	}
	if publishErr := s.changed(); publishErr != nil {
		return CreateTemplateResponse{}, publishErr
	}
	return CreateTemplateResponse{Template: templateView(record)}, nil
}

func (s *Service) UpdateTemplate(ctx context.Context, req UpdateTemplateRequest) (UpdateTemplateResponse, error) {
	var updated template.Template
	err := s.uow.Do(ctx, func(c context.Context) error {
		current, getErr := s.templates.Get(c, req.ID)
		if getErr != nil {
			return getErr
		}
		next := current
		if req.Name != nil {
			next.Name = strings.TrimSpace(*req.Name)
		}
		if req.PageKind != nil {
			next.PageKind = strings.TrimSpace(*req.PageKind)
		}
		if req.Spec != nil {
			next.Spec = *req.Spec
		}
		next.Version = current.Version + 1
		next.UpdatedAt = s.now()
		if validErr := next.Validate(); validErr != nil {
			return validErr
		}
		if updateErr := s.templates.Update(c, next); updateErr != nil {
			return updateErr
		}
		updated = next
		return nil
	})
	if err != nil {
		return UpdateTemplateResponse{}, err
	}
	if publishErr := s.changed(); publishErr != nil {
		return UpdateTemplateResponse{}, publishErr
	}
	return UpdateTemplateResponse{Template: templateView(updated)}, nil
}

func (s *Service) DeleteTemplate(ctx context.Context, req DeleteTemplateRequest) (DeleteTemplateResponse, error) {
	if err := s.uow.Do(ctx, func(c context.Context) error { return s.templates.Delete(c, req.ID) }); err != nil {
		return DeleteTemplateResponse{}, err
	}
	if err := s.changed(); err != nil {
		return DeleteTemplateResponse{}, err
	}
	return DeleteTemplateResponse{}, nil
}

func (s *Service) GetTemplate(ctx context.Context, req GetTemplateRequest) (GetTemplateResponse, error) {
	record, err := s.templates.Get(ctx, req.ID)
	if err != nil {
		return GetTemplateResponse{}, err
	}
	overrides, err := s.templates.ListOverrides(ctx, record.ID)
	if err != nil {
		return GetTemplateResponse{}, err
	}
	return GetTemplateResponse{Template: templateView(record), Overrides: overrideViews(overrides)}, nil
}

func (s *Service) ListTemplates(ctx context.Context, req ListTemplatesRequest) (paging.List[Template], error) {
	scope, err := scopeFilter(req.Scope)
	if err != nil {
		return paging.List[Template]{}, err
	}
	key, desc, err := sortOf(req.Sort)
	if err != nil {
		return paging.List[Template]{}, err
	}
	q := template.Query{Scope: scope, SiteID: siteFilter(req.SiteID), PageKind: req.PageKind, Sort: key, Desc: desc}

	list, err := s.templates.List(ctx, q, application.PageRequest(req.ListRequest))
	if err != nil {
		return paging.List[Template]{}, err
	}
	return application.MapList(list, templateView), nil
}
