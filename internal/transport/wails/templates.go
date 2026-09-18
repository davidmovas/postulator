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
	CreatePolicy(ctx context.Context, req templates.CreatePolicyRequest) (templates.CreatePolicyResponse, error)
	UpdatePolicy(ctx context.Context, req templates.UpdatePolicyRequest) (templates.UpdatePolicyResponse, error)
	DeletePolicy(ctx context.Context, req templates.DeletePolicyRequest) (templates.DeletePolicyResponse, error)
	GetPolicy(ctx context.Context, req templates.GetPolicyRequest) (templates.GetPolicyResponse, error)
	ListPolicies(ctx context.Context, req templates.ListPoliciesRequest) (paging.List[templates.LinkPolicy], error)
	GetEffectivePolicy(ctx context.Context, req templates.GetEffectivePolicyRequest) (templates.GetEffectivePolicyResponse, error)
}

type TemplatesService struct {
	createTemplate     middleware.Handler[templates.CreateTemplateRequest, templates.CreateTemplateResponse]
	updateTemplate     middleware.Handler[templates.UpdateTemplateRequest, templates.UpdateTemplateResponse]
	deleteTemplate     middleware.Handler[templates.DeleteTemplateRequest, templates.DeleteTemplateResponse]
	getTemplate        middleware.Handler[templates.GetTemplateRequest, templates.GetTemplateResponse]
	listTemplates      middleware.Handler[templates.ListTemplatesRequest, paging.List[templates.Template]]
	setOverride        middleware.Handler[templates.SetOverrideRequest, templates.SetOverrideResponse]
	deleteOverride     middleware.Handler[templates.DeleteOverrideRequest, templates.DeleteOverrideResponse]
	resolveForPage     middleware.Handler[templates.ResolveForPageRequest, templates.ResolveForPageResponse]
	createPolicy       middleware.Handler[templates.CreatePolicyRequest, templates.CreatePolicyResponse]
	updatePolicy       middleware.Handler[templates.UpdatePolicyRequest, templates.UpdatePolicyResponse]
	deletePolicy       middleware.Handler[templates.DeletePolicyRequest, templates.DeletePolicyResponse]
	getPolicy          middleware.Handler[templates.GetPolicyRequest, templates.GetPolicyResponse]
	listPolicies       middleware.Handler[templates.ListPoliciesRequest, paging.List[templates.LinkPolicy]]
	getEffectivePolicy middleware.Handler[templates.GetEffectivePolicyRequest, templates.GetEffectivePolicyResponse]
}

func NewTemplatesService(logger *zap.Logger, useCase Source[TemplatesUseCase]) *TemplatesService {
	return &TemplatesService{
		createTemplate:     Wrap(logger, "templates.createTemplate", call(useCase, TemplatesUseCase.CreateTemplate)),
		updateTemplate:     Wrap(logger, "templates.updateTemplate", call(useCase, TemplatesUseCase.UpdateTemplate)),
		deleteTemplate:     Wrap(logger, "templates.deleteTemplate", call(useCase, TemplatesUseCase.DeleteTemplate)),
		getTemplate:        Wrap(logger, "templates.getTemplate", call(useCase, TemplatesUseCase.GetTemplate)),
		listTemplates:      Wrap(logger, "templates.listTemplates", call(useCase, TemplatesUseCase.ListTemplates)),
		setOverride:        Wrap(logger, "templates.setOverride", call(useCase, TemplatesUseCase.SetOverride)),
		deleteOverride:     Wrap(logger, "templates.deleteOverride", call(useCase, TemplatesUseCase.DeleteOverride)),
		resolveForPage:     Wrap(logger, "templates.resolveForPage", call(useCase, TemplatesUseCase.ResolveForPage)),
		createPolicy:       Wrap(logger, "templates.createPolicy", call(useCase, TemplatesUseCase.CreatePolicy)),
		updatePolicy:       Wrap(logger, "templates.updatePolicy", call(useCase, TemplatesUseCase.UpdatePolicy)),
		deletePolicy:       Wrap(logger, "templates.deletePolicy", call(useCase, TemplatesUseCase.DeletePolicy)),
		getPolicy:          Wrap(logger, "templates.getPolicy", call(useCase, TemplatesUseCase.GetPolicy)),
		listPolicies:       Wrap(logger, "templates.listPolicies", call(useCase, TemplatesUseCase.ListPolicies)),
		getEffectivePolicy: Wrap(logger, "templates.getEffectivePolicy", call(useCase, TemplatesUseCase.GetEffectivePolicy)),
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

func (s *TemplatesService) CreatePolicy(c context.Context, req templates.CreatePolicyRequest) (templates.CreatePolicyResponse, error) {
	return s.createPolicy(c, req)
}

func (s *TemplatesService) UpdatePolicy(c context.Context, req templates.UpdatePolicyRequest) (templates.UpdatePolicyResponse, error) {
	return s.updatePolicy(c, req)
}

func (s *TemplatesService) DeletePolicy(c context.Context, req templates.DeletePolicyRequest) (templates.DeletePolicyResponse, error) {
	return s.deletePolicy(c, req)
}

func (s *TemplatesService) GetPolicy(c context.Context, req templates.GetPolicyRequest) (templates.GetPolicyResponse, error) {
	return s.getPolicy(c, req)
}

func (s *TemplatesService) ListPolicies(c context.Context, req templates.ListPoliciesRequest) (paging.List[templates.LinkPolicy], error) {
	return s.listPolicies(c, req)
}

func (s *TemplatesService) GetEffectivePolicy(c context.Context, req templates.GetEffectivePolicyRequest) (templates.GetEffectivePolicyResponse, error) {
	return s.getEffectivePolicy(c, req)
}
