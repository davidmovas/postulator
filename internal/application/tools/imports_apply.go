package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/imports"
)

const importsApplyName = "imports_apply"

type importsApplyArgs struct {
	Path          string      `json:"path" description:"The absolute path of the .xlsx or .csv file on this machine"`
	Mapping       mappingArgs `json:"mapping" description:"Which spreadsheet column fills which page field"`
	SaveMappingAs string      `json:"saveMappingAs,omitempty" description:"Keep the mapping under this name so the next import can reuse it; leave it out to use it once"`
}

func importsApply(deps Deps) Tool {
	return newSiteTool(Def{
		Name:        importsApplyName,
		Description: "Apply a spreadsheet import to the site in one transaction.",
		Risk:        RiskDangerous,
	}, func(ctx context.Context, b Binding, in importsApplyArgs) (imports.ApplyResponse, error) {
		return deps.Imports.Apply(ctx, imports.ApplyRequest{
			SiteID: b.SiteID, Path: in.Path, Mapping: in.Mapping.mapping(b.SiteID),
			Options: imports.ApplyOptions{SaveMappingAs: in.SaveMappingAs},
		})
	})
}
