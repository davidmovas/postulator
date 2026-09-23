package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/imports"
)

const importsInspectName = "imports_inspect"

func importsInspect(deps Deps) Tool {
	return newSiteTool(Def{
		Name:        importsInspectName,
		Description: "Read the headers and a sample of a spreadsheet and auto detect its mapping.",
		Risk:        RiskRead,
	}, func(ctx context.Context, b Binding, in imports.InspectRequest) (imports.InspectResponse, error) {
		in.SiteID = b.SiteID
		return deps.Imports.Inspect(ctx, in)
	})
}
