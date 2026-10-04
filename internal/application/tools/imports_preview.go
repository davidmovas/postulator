package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/imports"
)

const importsPreviewName = "imports_preview"

type importsPreviewArgs struct {
	Path    string      `json:"path" description:"Absolute path of the .xlsx or .csv file"`
	Mapping mappingArgs `json:"mapping,omitempty" description:"Sheet mapping; default the first sheet, detected"`
	Sheets  []sheetArgs `json:"sheets,omitempty" description:"Several sheets in workbook order, each mapped"`
}

func importsPreview(deps Deps) Tool {
	return newSiteTool(Def{
		Name:        importsPreviewName,
		Description: "Preview what a spreadsheet import would create: counts, sample pages and findings by code.",
		Risk:        RiskRead,
	}, func(ctx context.Context, b Binding, in importsPreviewArgs) (imports.PreviewSummaryResponse, error) {
		return deps.Imports.PreviewSummary(ctx, imports.PreviewRequest{
			SiteID: b.SiteID, Path: in.Path, Mapping: in.Mapping.mapping(b.SiteID), Sheets: sheetMappings(b.SiteID, in.Sheets),
		})
	})
}
