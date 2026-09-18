package wails_test

import (
	"context"
	"testing"

	"go.uber.org/zap"

	"github.com/davidmovas/postulator/internal/application/imports"
	"github.com/davidmovas/postulator/internal/transport/wails"
)

type importsFake struct{ mode failure }

func (f importsFake) Inspect(context.Context, imports.InspectRequest) (imports.InspectResponse, error) {
	return answer[imports.InspectResponse](f.mode)
}

func (f importsFake) Preview(context.Context, imports.PreviewRequest) (imports.PreviewResponse, error) {
	return answer[imports.PreviewResponse](f.mode)
}

func (f importsFake) Apply(context.Context, imports.ApplyRequest) (imports.ApplyResponse, error) {
	return answer[imports.ApplyResponse](f.mode)
}

func (f importsFake) Export(context.Context, imports.ExportRequest) (imports.ExportResponse, error) {
	return answer[imports.ExportResponse](f.mode)
}

func (f importsFake) SaveMapping(context.Context, imports.SaveMappingRequest) (imports.SaveMappingResponse, error) {
	return answer[imports.SaveMappingResponse](f.mode)
}

func (f importsFake) ListMappings(context.Context, imports.ListMappingsRequest) (imports.ListMappingsResponse, error) {
	return answer[imports.ListMappingsResponse](f.mode)
}

func (f importsFake) DeleteMapping(context.Context, imports.DeleteMappingRequest) (imports.DeleteMappingResponse, error) {
	return answer[imports.DeleteMappingResponse](f.mode)
}

func TestImportServiceConvertsEveryFailure(t *testing.T) {
	t.Parallel()

	assertMethodNames(t, wails.NewImportService(zap.NewNop(), importsFake{}), []string{
		"Apply", "DeleteMapping", "Export", "Inspect", "ListMappings", "Preview", "SaveMapping",
	})
	assertEveryMethodConverts(t, wails.NewImportService(zap.NewNop(), importsFake{mode: missing}), missingBody)
	assertEveryMethodConverts(t, wails.NewImportService(zap.NewNop(), importsFake{mode: panicking}), panicBody)
}
