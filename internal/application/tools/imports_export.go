package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/imports"
)

const importsExportName = "imports_export"

func importsExport(deps Deps) Tool {
	return newSiteTool(Def{
		Name:        importsExportName,
		Description: "Write the page map and the graph of the site to a workbook or a csv.",
		Risk:        RiskWrite,
	}, func(ctx context.Context, b Binding, in imports.ExportRequest) (imports.ExportResponse, error) {
		in.SiteID = b.SiteID
		return deps.Imports.Export(ctx, in)
	})
}
