package wails_test

import (
	"context"
	"testing"

	"go.uber.org/zap"

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

func TestTemplatesServiceConvertsEveryFailure(t *testing.T) {
	t.Parallel()

	assertMethodNames(t, wails.NewTemplatesService(zap.NewNop(), ready[wails.TemplatesUseCase](templatesFake{})), []string{
		"CreateTemplate", "DeleteOverride", "DeleteTemplate", "GetTemplate", "ListTemplates",
		"ResolveForPage", "SetOverride", "UpdateTemplate",
	})
	assertEveryMethodConverts(t, wails.NewTemplatesService(zap.NewNop(), ready[wails.TemplatesUseCase](templatesFake{mode: missing})), missingBody)
	assertEveryMethodConverts(t, wails.NewTemplatesService(zap.NewNop(), ready[wails.TemplatesUseCase](templatesFake{mode: panicking})), panicBody)
}
