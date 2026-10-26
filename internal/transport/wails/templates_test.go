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

func (f pageTypesFake) ListRules(context.Context, pagetypes.ListRulesRequest) (pagetypes.RulesResponse, error) {
	return answer[pagetypes.RulesResponse](f.mode)
}

func (f pageTypesFake) SaveRule(context.Context, pagetypes.SaveRuleRequest) (pagetypes.Rule, error) {
	return answer[pagetypes.Rule](f.mode)
}

func (f pageTypesFake) DeleteRule(context.Context, pagetypes.DeleteRuleRequest) (pagetypes.DeleteRuleResponse, error) {
	return answer[pagetypes.DeleteRuleResponse](f.mode)
}

func (f pageTypesFake) ReorderRules(context.Context, pagetypes.ReorderRulesRequest) (pagetypes.RulesResponse, error) {
	return answer[pagetypes.RulesResponse](f.mode)
}

type classificationFake struct{ mode failure }

func (f classificationFake) Preview(context.Context, pagetypes.PreviewRequest) (pagetypes.PreviewResponse, error) {
	return answer[pagetypes.PreviewResponse](f.mode)
}

func (f classificationFake) PreviewPages(context.Context, pagetypes.PreviewPagesRequest) (paging.List[pagetypes.PreviewPage], error) {
	return answer[paging.List[pagetypes.PreviewPage]](f.mode)
}

func (f classificationFake) Apply(context.Context, pagetypes.ApplyRequest) (pagetypes.ApplyResponse, error) {
	return answer[pagetypes.ApplyResponse](f.mode)
}

func (f classificationFake) TestRule(context.Context, pagetypes.TestRuleRequest) (pagetypes.TestRuleResponse, error) {
	return answer[pagetypes.TestRuleResponse](f.mode)
}

func templatesService(mode failure) *wails.TemplatesService {
	return wails.NewTemplatesService(zap.NewNop(), ready[wails.TemplatesUseCase](templatesFake{mode: mode}),
		ready[wails.PageTypesUseCase](pageTypesFake{mode: mode}), ready[wails.ClassificationUseCase](classificationFake{mode: mode}))
}

func TestTemplatesServiceConvertsEveryFailure(t *testing.T) {
	t.Parallel()

	assertMethodNames(t, templatesService(missing), []string{
		"ApplyClassification", "CreatePageType", "CreateTemplate", "DeleteOverride", "DeletePageType", "DeleteRule", "DeleteTemplate",
		"GetTemplate", "ListPageTypes", "ListRules", "ListTemplates", "PreviewClassification", "PreviewClassificationPages",
		"ReorderRules", "ResolveForPage", "SavePageType", "SaveRule", "SetOverride", "SetScope", "TestRule", "UpdateTemplate",
	})
	assertEveryMethodConverts(t, templatesService(missing), missingBody)
	assertEveryMethodConverts(t, templatesService(panicking), panicBody)
}
