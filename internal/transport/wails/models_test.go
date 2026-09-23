package wails_test

import (
	"context"
	"testing"

	"go.uber.org/zap"

	"github.com/davidmovas/postulator/internal/application/models"
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

func (f modelsFake) SetProviderKey(context.Context, models.SetProviderKeyRequest) (models.SetProviderKeyResponse, error) {
	return answer[models.SetProviderKeyResponse](f.mode)
}

func TestModelsServiceConvertsEveryFailure(t *testing.T) {
	t.Parallel()

	assertMethodNames(t, wails.NewModelsService(zap.NewNop(), ready[wails.ModelsUseCase](modelsFake{})), []string{
		"DisableModel", "GetProfiles", "ListModels", "SetProfile", "TestProvider", "UpsertModel", "UsageSummary",
	})
	assertEveryMethodConverts(t, wails.NewModelsService(zap.NewNop(), ready[wails.ModelsUseCase](modelsFake{mode: missing})), missingBody)
	assertEveryMethodConverts(t, wails.NewModelsService(zap.NewNop(), ready[wails.ModelsUseCase](modelsFake{mode: panicking})), panicBody)
}
