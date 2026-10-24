package tools

import (
	"context"
	"strconv"
	"strings"

	"github.com/davidmovas/postulator/internal/application/imports"
	"github.com/davidmovas/postulator/internal/application/pagetypes"
)

const importsApplyName = "imports_apply"

type importsApplyArgs struct {
	Path          string      `json:"path" description:"Absolute path of the .xlsx or .csv file"`
	Mapping       mappingArgs `json:"mapping,omitempty" description:"Sheet mapping; default the first sheet, detected"`
	Sheets        []sheetArgs `json:"sheets,omitempty" description:"Several sheets in workbook order, each mapped"`
	SaveMappingAs string      `json:"saveMappingAs,omitempty" description:"Save mappings under this name, per sheet as name / sheet"`
}

func importsApply(deps Deps) Tool {
	return confirming(newSiteTool(Def{
		Name:        importsApplyName,
		Description: "Apply a spreadsheet import to the site in one transaction.",
		Risk:        RiskDangerous,
	}, func(ctx context.Context, b Binding, in importsApplyArgs) (imports.ApplyResponse, error) {
		return deps.Imports.Apply(ctx, imports.ApplyRequest{
			SiteID: b.SiteID, Path: in.Path, Mapping: in.Mapping.mapping(b.SiteID), Sheets: sheetMappings(b.SiteID, in.Sheets),
			Options: imports.ApplyOptions{SaveMappingAs: in.SaveMappingAs},
		})
	}), func(ctx context.Context, b Binding, in importsApplyArgs) (string, error) {
		listed, err := deps.PageTypes.List(ctx, pagetypes.ListRequest{SiteID: b.SiteID})
		if err != nil {
			return "", err
		}
		rewriting := make(map[string]bool, len(listed.Types))
		for i := range listed.Types {
			if len(rewrittenFields(listed.Types[i])) > 0 {
				rewriting[listed.Types[i].Name] = true
			}
		}
		if len(rewriting) == 0 {
			return "", nil
		}
		previewed, err := deps.Imports.Preview(ctx, imports.PreviewRequest{
			SiteID: b.SiteID, Path: in.Path, Mapping: in.Mapping.mapping(b.SiteID), Sheets: sheetMappings(b.SiteID, in.Sheets),
		})
		if err != nil {
			return "", err
		}
		typed := make([]string, 0)
		for i := range previewed.Report.Pages {
			row := previewed.Report.Pages[i]
			if row.Action == string(imports.ActionUpdate) && rewriting[row.PageType] {
				typed = append(typed, row.Path)
			}
		}
		if len(typed) == 0 {
			return "", nil
		}
		return "Apply the import, which types " + strconv.Itoa(len(typed)) + " pages already in the map (" +
			strings.Join(typed, ", ") + ") into a type whose scope rewrites: their own text will be rewritten by the next run", nil
	})
}
