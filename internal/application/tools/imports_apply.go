package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/imports"
)

const importsApplyName = "imports_apply"

type importsApplyArgs struct {
	Path          string      `json:"path" description:"The absolute path of the .xlsx or .csv file on this machine"`
	Mapping       mappingArgs `json:"mapping,omitempty" description:"How to read the sheet; leave it out to read the first sheet with the columns detected from its headers"`
	Sheets        []sheetArgs `json:"sheets,omitempty" description:"Import these sheets together in workbook order, each with its own mapping"`
	SaveMappingAs string      `json:"saveMappingAs,omitempty" description:"Keep the mapping under this name, each sheet's as name / sheet, so the next import can reuse it"`
}

func importsApply(deps Deps) Tool {
	return newSiteTool(Def{
		Name:        importsApplyName,
		Description: "Apply a spreadsheet import to the site in one transaction.",
		Risk:        RiskDangerous,
	}, func(ctx context.Context, b Binding, in importsApplyArgs) (imports.ApplyResponse, error) {
		return deps.Imports.Apply(ctx, imports.ApplyRequest{
			SiteID: b.SiteID, Path: in.Path, Mapping: in.Mapping.mapping(b.SiteID), Sheets: sheetMappings(b.SiteID, in.Sheets),
			Options: imports.ApplyOptions{SaveMappingAs: in.SaveMappingAs},
		})
	})
}
