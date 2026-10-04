package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/imports"
)

const importsApplyName = "imports_apply"

type importsApplyArgs struct {
	Path          string      `json:"path" description:"Absolute path of the .xlsx or .csv file"`
	Mapping       mappingArgs `json:"mapping,omitempty" description:"Sheet mapping; default the first sheet, detected"`
	Sheets        []sheetArgs `json:"sheets,omitempty" description:"Several sheets in workbook order, each mapped"`
	SaveMappingAs string      `json:"saveMappingAs,omitempty" description:"Save mappings under this name, per sheet as name / sheet"`
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
