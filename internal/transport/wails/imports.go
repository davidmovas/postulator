package wails

import (
	"context"

	"go.uber.org/zap"

	"github.com/davidmovas/postulator/internal/application/imports"
	"github.com/davidmovas/postulator/internal/kernel/middleware"
)

type ImportsUseCase interface {
	Inspect(ctx context.Context, req imports.InspectRequest) (imports.InspectResponse, error)
	Preview(ctx context.Context, req imports.PreviewRequest) (imports.PreviewResponse, error)
	Apply(ctx context.Context, req imports.ApplyRequest) (imports.ApplyResponse, error)
	Export(ctx context.Context, req imports.ExportRequest) (imports.ExportResponse, error)
	SaveMapping(ctx context.Context, req imports.SaveMappingRequest) (imports.SaveMappingResponse, error)
	ListMappings(ctx context.Context, req imports.ListMappingsRequest) (imports.ListMappingsResponse, error)
	DeleteMapping(ctx context.Context, req imports.DeleteMappingRequest) (imports.DeleteMappingResponse, error)
}

type ImportService struct {
	inspect       middleware.Handler[imports.InspectRequest, imports.InspectResponse]
	preview       middleware.Handler[imports.PreviewRequest, imports.PreviewResponse]
	apply         middleware.Handler[imports.ApplyRequest, imports.ApplyResponse]
	export        middleware.Handler[imports.ExportRequest, imports.ExportResponse]
	saveMapping   middleware.Handler[imports.SaveMappingRequest, imports.SaveMappingResponse]
	listMappings  middleware.Handler[imports.ListMappingsRequest, imports.ListMappingsResponse]
	deleteMapping middleware.Handler[imports.DeleteMappingRequest, imports.DeleteMappingResponse]
}

func NewImportService(logger *zap.Logger, useCase Source[ImportsUseCase]) *ImportService {
	return &ImportService{
		inspect:       Wrap(logger, "import.inspect", call(useCase, ImportsUseCase.Inspect)),
		preview:       Wrap(logger, "import.preview", call(useCase, ImportsUseCase.Preview)),
		apply:         Wrap(logger, "import.apply", call(useCase, ImportsUseCase.Apply)),
		export:        Wrap(logger, "import.export", call(useCase, ImportsUseCase.Export)),
		saveMapping:   Wrap(logger, "import.saveMapping", call(useCase, ImportsUseCase.SaveMapping)),
		listMappings:  Wrap(logger, "import.listMappings", call(useCase, ImportsUseCase.ListMappings)),
		deleteMapping: Wrap(logger, "import.deleteMapping", call(useCase, ImportsUseCase.DeleteMapping)),
	}
}

func (s *ImportService) Inspect(c context.Context, req imports.InspectRequest) (imports.InspectResponse, error) {
	return s.inspect(c, req)
}

func (s *ImportService) Preview(c context.Context, req imports.PreviewRequest) (imports.PreviewResponse, error) {
	return s.preview(c, req)
}

func (s *ImportService) Apply(c context.Context, req imports.ApplyRequest) (imports.ApplyResponse, error) {
	return s.apply(c, req)
}

func (s *ImportService) Export(c context.Context, req imports.ExportRequest) (imports.ExportResponse, error) {
	return s.export(c, req)
}

func (s *ImportService) SaveMapping(c context.Context, req imports.SaveMappingRequest) (imports.SaveMappingResponse, error) {
	return s.saveMapping(c, req)
}

func (s *ImportService) ListMappings(c context.Context, req imports.ListMappingsRequest) (imports.ListMappingsResponse, error) {
	return s.listMappings(c, req)
}

func (s *ImportService) DeleteMapping(c context.Context, req imports.DeleteMappingRequest) (imports.DeleteMappingResponse, error) {
	return s.deleteMapping(c, req)
}
