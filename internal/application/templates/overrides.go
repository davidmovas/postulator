package templates

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/site"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

func (s *Service) SetOverride(ctx context.Context, req SetOverrideRequest) (SetOverrideResponse, error) {
	var stored template.Override
	err := s.uow.Do(ctx, func(c context.Context) error {
		base, getErr := s.templates.Get(c, req.TemplateID)
		if getErr != nil {
			return getErr
		}
		now := s.now()
		override := template.Override{
			ID:         id.New(),
			TemplateID: base.ID,
			Scope:      template.OverrideScope(req.Scope),
			TargetID:   req.TargetID,
			Patch:      req.Patch,
			CreatedAt:  now,
			UpdatedAt:  now,
		}
		if validErr := override.Validate(); validErr != nil {
			return validErr
		}
		siteOverride, pageOverride, chainErr := s.chain(c, &base, &override)
		if chainErr != nil {
			return chainErr
		}
		if _, resolveErr := template.Resolve(base.Spec, siteOverride, pageOverride); resolveErr != nil {
			return resolveErr
		}
		saved, upsertErr := s.templates.UpsertOverride(c, override)
		if upsertErr != nil {
			return upsertErr
		}
		stored = saved
		return nil
	})
	if err != nil {
		return SetOverrideResponse{}, err
	}
	if publishErr := s.changed(); publishErr != nil {
		return SetOverrideResponse{}, publishErr
	}
	return SetOverrideResponse{Override: overrideView(stored)}, nil
}

func (s *Service) chain(ctx context.Context, base *template.Template, override *template.Override) (siteOverride, pageOverride json.RawMessage, err error) {
	switch override.Scope {
	case template.OverrideSite:
		if _, err = s.sites.Get(ctx, override.TargetID); err != nil {
			return nil, nil, err
		}
		return override.Patch, nil, nil
	case template.OverridePage:
		page, getErr := s.pages.Get(ctx, override.TargetID)
		if getErr != nil {
			return nil, nil, getErr
		}
		siteOverride, err = s.patch(ctx, base.ID, template.OverrideSite, page.SiteID)
		if err != nil {
			return nil, nil, err
		}
		return siteOverride, override.Patch, nil
	default:
		return nil, nil, errors.New(errors.Invalid, "override scope is not recognized").WithDetail("field", "scope")
	}
}

func (s *Service) patch(ctx context.Context, templateID string, scope template.OverrideScope, targetID string) (json.RawMessage, error) {
	override, err := s.templates.GetOverride(ctx, templateID, scope, targetID)
	if errors.IsCode(err, errors.NotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return override.Patch, nil
}

func (s *Service) DeleteOverride(ctx context.Context, req DeleteOverrideRequest) (DeleteOverrideResponse, error) {
	if err := s.uow.Do(ctx, func(c context.Context) error { return s.templates.DeleteOverride(c, req.ID) }); err != nil {
		return DeleteOverrideResponse{}, err
	}
	if err := s.changed(); err != nil {
		return DeleteOverrideResponse{}, err
	}
	return DeleteOverrideResponse{}, nil
}

func (s *Service) ResolveForPage(ctx context.Context, req ResolveForPageRequest) (ResolveForPageResponse, error) {
	page, err := s.pages.Get(ctx, req.PageID)
	if err != nil {
		return ResolveForPageResponse{}, err
	}

	owner, err := s.sites.Get(ctx, page.SiteID)
	if err != nil {
		return ResolveForPageResponse{}, err
	}
	picked := strings.TrimSpace(req.TemplateID)
	templateID := page.TemplateID
	if picked != "" {
		templateID = &picked
	}
	if templateID == nil {
		templateID = owner.Defaults.TemplateID
	}
	if templateID == nil {
		return ResolveForPageResponse{}, errors.New(errors.NotFound, "page has no template and its site has no default template").WithDetail("pageId", page.ID)
	}

	base, err := s.templates.Get(ctx, *templateID)
	if err != nil {
		return ResolveForPageResponse{}, err
	}
	if picked != "" && base.Scope == template.ScopeSite && (base.SiteID == nil || *base.SiteID != page.SiteID) {
		return ResolveForPageResponse{}, errors.New(errors.Invalid, "the template "+base.Name+
			" belongs to another site, so it cannot write "+page.Path).
			WithDetail("field", "templateId").WithDetail("pageId", page.ID)
	}
	siteOverride, err := s.patch(ctx, base.ID, template.OverrideSite, page.SiteID)
	if err != nil {
		return ResolveForPageResponse{}, err
	}
	pageOverride, err := s.patch(ctx, base.ID, template.OverridePage, page.ID)
	if err != nil {
		return ResolveForPageResponse{}, err
	}
	spec, err := template.Resolve(base.Spec, siteOverride, pageOverride)
	if err != nil {
		return ResolveForPageResponse{}, err
	}
	vars, err := s.varsFor(ctx, page, owner)
	if err != nil {
		return ResolveForPageResponse{}, err
	}
	return ResolveForPageResponse{TemplateID: base.ID, SiteID: page.SiteID, Version: base.Version, Spec: spec.Expanded(vars)}, nil
}

func (s *Service) varsFor(ctx context.Context, page pagemap.Page, owner site.Site) (template.Vars, error) {
	vars := template.Vars{SiteName: owner.Name, PageTitle: page.Title}
	if page.EntityID == nil {
		return vars, nil
	}
	entity, err := s.entities.Get(ctx, *page.EntityID)
	if errors.IsCode(err, errors.NotFound) {
		return vars, nil
	}
	if err != nil {
		return template.Vars{}, err
	}
	vars.PrimaryKeyword = entity.PrimaryKeyword
	vars.EntityName = entity.Name
	return vars, nil
}
