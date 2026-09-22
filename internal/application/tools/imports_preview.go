package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/imports"
)

const importsPreviewName = "imports_preview"

type importsPreviewArgs struct {
	Path    string      `json:"path" description:"The absolute path of the .xlsx or .csv file on this machine"`
	Mapping mappingArgs `json:"mapping,omitempty" description:"Which spreadsheet column fills which page field; leave it out and the columns detected by imports_inspect are used"`
}

func importsPreview(deps Deps) Tool {
	return newSiteTool(Def{
		Name: importsPreviewName,
		Description: "Preview what a spreadsheet import would create: the counts, a sample of the pages " +
			"and every warning and error grouped by its code.",
		Risk: RiskRead,
	}, func(ctx context.Context, b Binding, in importsPreviewArgs) (imports.PreviewSummaryResponse, error) {
		return deps.Imports.PreviewSummary(ctx, imports.PreviewRequest{
			SiteID: b.SiteID, Path: in.Path, Mapping: in.Mapping.mapping(b.SiteID),
		})
	})
}
