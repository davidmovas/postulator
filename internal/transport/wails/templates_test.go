package wails_test

import (
	"context"
	"testing"

	"go.uber.org/zap"

	"github.com/davidmovas/postulator/internal/application/pagetypes"
	"github.com/davidmovas/postulator/internal/application/templates"
	"github.com/davidmovas/postulator/internal/kernel/paging"
	"github.com/davidmovas/postulator/internal/transport/wails"
)

type templatesFake struct{ mode failure }

func (f templatesFake) CreateTemplate(context.Context, templates.CreateTemplateRequest) (templates.CreateTemplateResponse, error) {
	return answer[templates.CreateTemplateResponse](f.mode)
}

func (f templatesFake) UpdateTemplate(context.Context, templates.UpdateTemplateRequest) (templates.UpdateTemplateResponse, error) {
	return answer[templates.UpdateTemplateResponse](f.mode)
}

func (f templatesFake) DeleteTemplate(context.Context, templates.DeleteTemplateRequest) (templates.DeleteTemplateResponse, error) {
	return answer[templates.DeleteTemplateResponse](f.mode)
}

func (f templatesFake) GetTemplate(context.Context, templates.GetTemplateRequest) (templates.GetTemplateResponse, error) {
	return answer[templates.GetTemplateResponse](f.mode)
}

func (f templatesFake) ListTemplates(context.Context, templates.ListTemplatesRequest) (paging.List[templates.Template], error) {
	return answer[paging.List[templates.Template]](f.mode)
}

func (f templatesFake) SetOverride(context.Context, templates.SetOverrideRequest) (templates.SetOverrideResponse, error) {
	return answer[templates.SetOverrideResponse](f.mode)
}

func (f templatesFake) DeleteOverride(context.Context, templates.DeleteOverrideRequest) (templates.DeleteOverrideResponse, error) {
	return answer[templates.DeleteOverrideResponse](f.mode)
}

func (f templatesFake) ResolveForPage(context.Context, templates.ResolveForPageRequest) (templates.ResolveForPageResponse, error) {
	return answer[templates.ResolveForPageResponse](f.mode)
}

type pageTypesFake struct{ mode failure }

func (f pageTypesFake) List(context.Context, pagetypes.ListRequest) (pagetypes.ListResponse, error) {
	return answer[pagetypes.ListResponse](f.mode)
}

func (f pageTypesFake) Create(context.Context, pagetypes.CreateRequest) (pagetypes.Type, error) {
	return answer[pagetypes.Type](f.mode)
}

func (f pageTypesFake) Save(context.Context, pagetypes.SaveRequest) (pagetypes.Type, error) {
	return answer[pagetypes.Type](f.mode)
}

func (f pageTypesFake) Delete(context.Context, pagetypes.DeleteRequest) (pagetypes.DeleteResponse, error) {
	return answer[pagetypes.DeleteResponse](f.mode)
}

func (f pageTypesFake) SetScope(context.Context, pagetypes.SetScopeRequest) (pagetypes.Scope, error) {
	return answer[pagetypes.Scope](f.mode)
}

func templatesService(mode failure) *wails.TemplatesService {
	return wails.NewTemplatesService(zap.NewNop(), ready[wails.TemplatesUseCase](templatesFake{mode: mode}),
		ready[wails.PageTypesUseCase](pageTypesFake{mode: mode}))
}

func TestTemplatesServiceConvertsEveryFailure(t *testing.T) {
	t.Parallel()

	assertMethodNames(t, templatesService(missing), []string{
		"CreatePageType", "CreateTemplate", "DeleteOverride", "DeletePageType", "DeleteTemplate", "GetTemplate",
		"ListPageTypes", "ListTemplates", "ResolveForPage", "SavePageType", "SetOverride", "SetScope", "UpdateTemplate",
	})
	assertEveryMethodConverts(t, templatesService(missing), missingBody)
	assertEveryMethodConverts(t, templatesService(panicking), panicBody)
}
