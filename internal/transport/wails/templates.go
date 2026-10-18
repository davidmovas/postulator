package wails

import (
	"context"

	"go.uber.org/zap"

	"github.com/davidmovas/postulator/internal/application/templates"
	"github.com/davidmovas/postulator/internal/kernel/middleware"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

type TemplatesUseCase interface {
	CreateTemplate(ctx context.Context, req templates.CreateTemplateRequest) (templates.CreateTemplateResponse, error)
	UpdateTemplate(ctx context.Context, req templates.UpdateTemplateRequest) (templates.UpdateTemplateResponse, error)
	DeleteTemplate(ctx context.Context, req templates.DeleteTemplateRequest) (templates.DeleteTemplateResponse, error)
	GetTemplate(ctx context.Context, req templates.GetTemplateRequest) (templates.GetTemplateResponse, error)
	ListTemplates(ctx context.Context, req templates.ListTemplatesRequest) (paging.List[templates.Template], error)
	SetOverride(ctx context.Context, req templates.SetOverrideRequest) (templates.SetOverrideResponse, error)
	DeleteOverride(ctx context.Context, req templates.DeleteOverrideRequest) (templates.DeleteOverrideResponse, error)
	ResolveForPage(ctx context.Context, req templates.ResolveForPageRequest) (templates.ResolveForPageResponse, error)
}

type TemplatesService struct {
	createTemplate middleware.Handler[templates.CreateTemplateRequest, templates.CreateTemplateResponse]
	updateTemplate middleware.Handler[templates.UpdateTemplateRequest, templates.UpdateTemplateResponse]
	deleteTemplate middleware.Handler[templates.DeleteTemplateRequest, templates.DeleteTemplateResponse]
	getTemplate    middleware.Handler[templates.GetTemplateRequest, templates.GetTemplateResponse]
	listTemplates  middleware.Handler[templates.ListTemplatesRequest, paging.List[templates.Template]]
	setOverride    middleware.Handler[templates.SetOverrideRequest, templates.SetOverrideResponse]
	deleteOverride middleware.Handler[templates.DeleteOverrideRequest, templates.DeleteOverrideResponse]
	resolveForPage middleware.Handler[templates.ResolveForPageRequest, templates.ResolveForPageResponse]
}

func NewTemplatesService(logger *zap.Logger, useCase Source[TemplatesUseCase]) *TemplatesService {
	return &TemplatesService{
		createTemplate: Wrap(logger, "templates.createTemplate", call(useCase, TemplatesUseCase.CreateTemplate)),
		updateTemplate: Wrap(logger, "templates.updateTemplate", call(useCase, TemplatesUseCase.UpdateTemplate)),
		deleteTemplate: Wrap(logger, "templates.deleteTemplate", call(useCase, TemplatesUseCase.DeleteTemplate)),
		getTemplate:    Wrap(logger, "templates.getTemplate", call(useCase, TemplatesUseCase.GetTemplate)),
		listTemplates:  Wrap(logger, "templates.listTemplates", call(useCase, TemplatesUseCase.ListTemplates)),
		setOverride:    Wrap(logger, "templates.setOverride", call(useCase, TemplatesUseCase.SetOverride)),
		deleteOverride: Wrap(logger, "templates.deleteOverride", call(useCase, TemplatesUseCase.DeleteOverride)),
		resolveForPage: Wrap(logger, "templates.resolveForPage", call(useCase, TemplatesUseCase.ResolveForPage)),
	}
}

func (s *TemplatesService) CreateTemplate(c context.Context, req templates.CreateTemplateRequest) (templates.CreateTemplateResponse, error) {
	return s.createTemplate(c, req)
}

func (s *TemplatesService) UpdateTemplate(c context.Context, req templates.UpdateTemplateRequest) (templates.UpdateTemplateResponse, error) {
	return s.updateTemplate(c, req)
}

func (s *TemplatesService) DeleteTemplate(c context.Context, req templates.DeleteTemplateRequest) (templates.DeleteTemplateResponse, error) {
	return s.deleteTemplate(c, req)
}

func (s *TemplatesService) GetTemplate(c context.Context, req templates.GetTemplateRequest) (templates.GetTemplateResponse, error) {
	return s.getTemplate(c, req)
}

func (s *TemplatesService) ListTemplates(c context.Context, req templates.ListTemplatesRequest) (paging.List[templates.Template], error) {
	return s.listTemplates(c, req)
}

func (s *TemplatesService) SetOverride(c context.Context, req templates.SetOverrideRequest) (templates.SetOverrideResponse, error) {
	return s.setOverride(c, req)
}

func (s *TemplatesService) DeleteOverride(c context.Context, req templates.DeleteOverrideRequest) (templates.DeleteOverrideResponse, error) {
	return s.deleteOverride(c, req)
}

func (s *TemplatesService) ResolveForPage(c context.Context, req templates.ResolveForPageRequest) (templates.ResolveForPageResponse, error) {
	return s.resolveForPage(c, req)
}
