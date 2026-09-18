package wails

import (
	"context"

	"go.uber.org/zap"

	"github.com/davidmovas/postulator/internal/application/models"
	"github.com/davidmovas/postulator/internal/kernel/middleware"
)

type ModelsUseCase interface {
	ListModels(ctx context.Context, req models.ListModelsRequest) (models.ListModelsResponse, error)
	UpsertModel(ctx context.Context, req models.UpsertModelRequest) (models.UpsertModelResponse, error)
	DisableModel(ctx context.Context, req models.DisableModelRequest) (models.DisableModelResponse, error)
	GetProfiles(ctx context.Context, req models.GetProfilesRequest) (models.GetProfilesResponse, error)
	SetProfile(ctx context.Context, req models.SetProfileRequest) (models.SetProfileResponse, error)
	TestProvider(ctx context.Context, req models.TestProviderRequest) (models.TestProviderResponse, error)
	UsageSummary(ctx context.Context, req models.UsageSummaryRequest) (models.UsageSummaryResponse, error)
}

type ModelsService struct {
	listModels   middleware.Handler[models.ListModelsRequest, models.ListModelsResponse]
	upsertModel  middleware.Handler[models.UpsertModelRequest, models.UpsertModelResponse]
	disableModel middleware.Handler[models.DisableModelRequest, models.DisableModelResponse]
	getProfiles  middleware.Handler[models.GetProfilesRequest, models.GetProfilesResponse]
	setProfile   middleware.Handler[models.SetProfileRequest, models.SetProfileResponse]
	testProvider middleware.Handler[models.TestProviderRequest, models.TestProviderResponse]
	usageSummary middleware.Handler[models.UsageSummaryRequest, models.UsageSummaryResponse]
}

func NewModelsService(logger *zap.Logger, useCase Source[ModelsUseCase]) *ModelsService {
	return &ModelsService{
		listModels:   Wrap(logger, "models.listModels", call(useCase, ModelsUseCase.ListModels)),
		upsertModel:  Wrap(logger, "models.upsertModel", call(useCase, ModelsUseCase.UpsertModel)),
		disableModel: Wrap(logger, "models.disableModel", call(useCase, ModelsUseCase.DisableModel)),
		getProfiles:  Wrap(logger, "models.getProfiles", call(useCase, ModelsUseCase.GetProfiles)),
		setProfile:   Wrap(logger, "models.setProfile", call(useCase, ModelsUseCase.SetProfile)),
		testProvider: Wrap(logger, "models.testProvider", call(useCase, ModelsUseCase.TestProvider)),
		usageSummary: Wrap(logger, "models.usageSummary", call(useCase, ModelsUseCase.UsageSummary)),
	}
}

func (s *ModelsService) ListModels(c context.Context, req models.ListModelsRequest) (models.ListModelsResponse, error) {
	return s.listModels(c, req)
}

func (s *ModelsService) UpsertModel(c context.Context, req models.UpsertModelRequest) (models.UpsertModelResponse, error) {
	return s.upsertModel(c, req)
}

func (s *ModelsService) DisableModel(c context.Context, req models.DisableModelRequest) (models.DisableModelResponse, error) {
	return s.disableModel(c, req)
}

func (s *ModelsService) GetProfiles(c context.Context, req models.GetProfilesRequest) (models.GetProfilesResponse, error) {
	return s.getProfiles(c, req)
}

func (s *ModelsService) SetProfile(c context.Context, req models.SetProfileRequest) (models.SetProfileResponse, error) {
	return s.setProfile(c, req)
}

func (s *ModelsService) TestProvider(c context.Context, req models.TestProviderRequest) (models.TestProviderResponse, error) {
	return s.testProvider(c, req)
}

func (s *ModelsService) UsageSummary(c context.Context, req models.UsageSummaryRequest) (models.UsageSummaryResponse, error) {
	return s.usageSummary(c, req)
}
