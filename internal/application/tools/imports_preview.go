package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/imports"
)

const importsPreviewName = "imports_preview"

type importsPreviewArgs struct {
	Path    string      `json:"path" description:"The absolute path of the .xlsx or .csv file on this machine"`
	Mapping mappingArgs `json:"mapping,omitempty" description:"How to read the sheet; leave it out to read the first sheet with the columns detected from its headers"`
	Sheets  []sheetArgs `json:"sheets,omitempty" description:"Import these sheets together in workbook order, each with its own mapping"`
}

func importsPreview(deps Deps) Tool {
	return newSiteTool(Def{
		Name: importsPreviewName,
		Description: "Preview what a spreadsheet import would create: the counts, a sample of the pages " +
			"and every warning and error grouped by its code.",
		Risk: RiskRead,
	}, func(ctx context.Context, b Binding, in importsPreviewArgs) (imports.PreviewSummaryResponse, error) {
		return deps.Imports.PreviewSummary(ctx, imports.PreviewRequest{
			SiteID: b.SiteID, Path: in.Path, Mapping: in.Mapping.mapping(b.SiteID), Sheets: sheetMappings(b.SiteID, in.Sheets),
		})
	})
}
