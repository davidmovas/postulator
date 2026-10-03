package wails_test

import (
	"context"
	"testing"

	"go.uber.org/zap"

	appcontent "github.com/davidmovas/postulator/internal/application/content"
	appgraph "github.com/davidmovas/postulator/internal/application/graph"
	"github.com/davidmovas/postulator/internal/application/models"
	"github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/kernel/paging"
	"github.com/davidmovas/postulator/internal/transport/agent"
	"github.com/davidmovas/postulator/internal/transport/wails"
)

type modelsFake struct{ mode failure }

func (f modelsFake) ListModels(context.Context, models.ListModelsRequest) (models.ListModelsResponse, error) {
	return answer[models.ListModelsResponse](f.mode)
}

func (f modelsFake) UpsertModel(context.Context, models.UpsertModelRequest) (models.UpsertModelResponse, error) {
	return answer[models.UpsertModelResponse](f.mode)
}

func (f modelsFake) DisableModel(context.Context, models.DisableModelRequest) (models.DisableModelResponse, error) {
	return answer[models.DisableModelResponse](f.mode)
}

func (f modelsFake) GetProfiles(context.Context, models.GetProfilesRequest) (models.GetProfilesResponse, error) {
	return answer[models.GetProfilesResponse](f.mode)
}

func (f modelsFake) SetProfile(context.Context, models.SetProfileRequest) (models.SetProfileResponse, error) {
	return answer[models.SetProfileResponse](f.mode)
}

func (f modelsFake) TestProvider(context.Context, models.TestProviderRequest) (models.TestProviderResponse, error) {
	return answer[models.TestProviderResponse](f.mode)
}

func (f modelsFake) UsageSummary(context.Context, models.UsageSummaryRequest) (models.UsageSummaryResponse, error) {
	return answer[models.UsageSummaryResponse](f.mode)
}

func (f modelsFake) SpendReport(context.Context, models.SpendReportRequest) (models.SpendReportResponse, error) {
	return answer[models.SpendReportResponse](f.mode)
}

func (f modelsFake) ListCalls(context.Context, models.ListCallsRequest) (paging.List[models.Call], error) {
	return answer[paging.List[models.Call]](f.mode)
}

func (f modelsFake) SetProviderKey(context.Context, models.SetProviderKeyRequest) (models.SetProviderKeyResponse, error) {
	return answer[models.SetProviderKeyResponse](f.mode)
}

func TestModelsServiceConvertsEveryFailure(t *testing.T) {
	t.Parallel()

	assertMethodNames(t, wails.NewModelsService(zap.NewNop(), ready[wails.ModelsUseCase](modelsFake{})), []string{
		"DisableModel", "GetProfiles", "ListCalls", "ListModels", "SetProfile", "SpendReport", "TestProvider",
		"UpsertModel", "UsageSummary",
	})
	assertEveryMethodConverts(t, wails.NewModelsService(zap.NewNop(), ready[wails.ModelsUseCase](modelsFake{mode: missing})), missingBody)
	assertEveryMethodConverts(t, wails.NewModelsService(zap.NewNop(), ready[wails.ModelsUseCase](modelsFake{mode: panicking})), panicBody)
}

func TestEveryPurposeStepIsTheStepItsCallerNames(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		named   string
		purpose llm.Purpose
	}{
		{name: "an agent round", named: agent.ChatStep, purpose: llm.PurposeChat},
		{name: "pages proposed as entities", named: appgraph.NameProposeFromPages, purpose: llm.PurposeGraph},
		{name: "keywords proposed as entities", named: appgraph.NameProposeFromKeywords, purpose: llm.PurposeGraph},
		{name: "related edges proposed", named: appgraph.NameProposeRelated, purpose: llm.PurposeGraph},
		{name: "a page judged on demand", named: appcontent.NameJudge, purpose: llm.PurposeAudit},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := llm.PurposeOf("", tc.named); got != tc.purpose {
				t.Errorf("a call outside a run on step %q counts as %q, want %q", tc.named, got, tc.purpose)
			}
		})
	}
}
