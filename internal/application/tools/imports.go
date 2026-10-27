package tools

import (
	"context"
	"strconv"
	"strings"

	"github.com/davidmovas/postulator/internal/application/imports"
	"github.com/davidmovas/postulator/internal/application/pagetypes"
	"github.com/davidmovas/postulator/internal/domain/importmap"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	importsName     = "imports"
	opInspect       = "inspect"
	opExport        = "export"
	opSaveMapping   = "save_mapping"
	opMappings      = "mappings"
	opDeleteMapping = "delete_mapping"
)

type columnArgs struct {
	Field  string `json:"field" enum:"path,title,h1,primary_keyword,keywords,anchors,entity,page_type,parent_entity,related,meta_title,meta_description,wp_type,own_entity,sku" description:"Page field"`
	Column string `json:"column" description:"Source header"`
}

type mappingArgs struct {
	ID      string           `json:"id,omitempty" description:"Saved mapping id, used when no columns are given"`
	Name    string           `json:"name,omitempty" description:"Save under this name"`
	Columns []columnArgs     `json:"columns,omitempty" description:"Header per page field; omit to detect"`
	Options *imports.Options `json:"options,omitempty" description:"Cell reading options"`
}

type sheetArgs struct {
	Sheet     string            `json:"sheet" description:"Sheet name from inspect"`
	MappingID string            `json:"mappingId,omitempty" description:"Saved mapping id; default detect"`
	RowType   importmap.RowType `json:"rowType,omitempty" enum:"pages,products,type" description:"New row type; default pages"`
}

func sheetMappings(siteID string, sheets []sheetArgs) []imports.SheetMapping {
	if len(sheets) == 0 {
		return nil
	}
	out := make([]imports.SheetMapping, 0, len(sheets))
	for _, sheet := range sheets {
		out = append(out, imports.SheetMapping{Sheet: sheet.Sheet, Mapping: imports.Mapping{
			ID: sheet.MappingID, SiteID: siteID, Options: imports.Options{RowType: sheet.RowType},
		}})
	}
	return out
}

func (m *mappingArgs) mapping(siteID string) imports.Mapping {
	if m == nil {
		return imports.Mapping{SiteID: siteID, Columns: map[string]string{}}
	}
	columns := make(map[string]string, len(m.Columns))
	for _, column := range m.Columns {
		columns[column.Field] = column.Column
	}
	built := imports.Mapping{ID: m.ID, SiteID: siteID, Name: m.Name, Columns: columns}
	if m.Options != nil {
		built.Options = *m.Options
	}
	return built
}

type importsArgs struct {
	Op            string       `json:"op" enum:"inspect,preview,apply,export,save_mapping,mappings,delete_mapping" description:"inspect reads a spreadsheet's headers and a sample and detects its mapping; preview says what an import would do; apply writes it in one transaction; export writes the page map and the graph to a file"`
	Path          string       `json:"path,omitempty" description:"inspect, preview, apply, export: the absolute path of the .xlsx or .csv file"`
	Sample        []string     `json:"sample,omitempty" description:"inspect: the sheets to sample; default the first, every sheet is listed"`
	NoHeader      bool         `json:"noHeader,omitempty" description:"inspect: no header row, columns go by letter"`
	Mapping       *mappingArgs `json:"mapping,omitempty" description:"preview, apply, save_mapping: the sheet mapping; default the first sheet, detected"`
	Sheets        []sheetArgs  `json:"sheets,omitempty" description:"preview, apply: several sheets in workbook order, each mapped"`
	SaveMappingAs string       `json:"saveMappingAs,omitempty" description:"apply: save the mappings under this name, per sheet as name / sheet"`
	Format        string       `json:"format,omitempty" enum:"xlsx,csv" description:"export: default from the file extension"`
	MappingID     string       `json:"mappingId,omitempty" description:"delete_mapping: the saved mapping"`
}

func importsTool(deps Deps) Tool {
	tool := confirming(newSiteTool(Def{
		Name:        importsName,
		Description: "Read a client's workbook and import it into the model, export the model to one, and keep the column mappings.",
		Risk:        RiskDangerous,
	}, func(ctx context.Context, b Binding, in importsArgs) (any, error) {
		return deps.manageImports(ctx, b, &in)
	}), func(ctx context.Context, b Binding, in importsArgs) (string, error) {
		if in.Op != opApply {
			return "", nil
		}
		return deps.importRewrites(ctx, b, &in)
	})
	tool = describing(tool, func(_ context.Context, _ Binding, in importsArgs) (string, error) {
		switch in.Op {
		case opApply:
			return "Apply the workbook " + in.Path + " to the model in one transaction; nothing reaches the site", nil
		case opExport:
			return "Write the page map and the graph to " + in.Path, nil
		case opSaveMapping:
			return "Save the column mapping " + quoted(in.Mapping.mapping("").Name), nil
		default:
			return "Delete the saved column mapping " + in.MappingID, nil
		}
	})
	return riskedBy(tool, map[string]Risk{
		opInspect: RiskRead, opPreview: RiskRead, opMappings: RiskRead, opExport: RiskWrite, opSaveMapping: RiskWrite,
	})
}

func (deps Deps) manageImports(ctx context.Context, b Binding, in *importsArgs) (any, error) {
	switch in.Op {
	case opInspect:
		return deps.Imports.Inspect(ctx, imports.InspectRequest{SiteID: b.SiteID, Path: in.Path, Sheets: in.Sample, NoHeader: in.NoHeader})
	case opPreview:
		return deps.Imports.PreviewSummary(ctx, imports.PreviewRequest{
			SiteID: b.SiteID, Path: in.Path, Mapping: in.Mapping.mapping(b.SiteID), Sheets: sheetMappings(b.SiteID, in.Sheets),
		})
	case opApply:
		return deps.Imports.Apply(ctx, imports.ApplyRequest{
			SiteID: b.SiteID, Path: in.Path, Mapping: in.Mapping.mapping(b.SiteID), Sheets: sheetMappings(b.SiteID, in.Sheets),
			Options: imports.ApplyOptions{SaveMappingAs: in.SaveMappingAs},
		})
	case opExport:
		return deps.Imports.Export(ctx, imports.ExportRequest{SiteID: b.SiteID, Path: in.Path, Format: in.Format})
	case opSaveMapping:
		return deps.Imports.SaveMapping(ctx, imports.SaveMappingRequest{Mapping: in.Mapping.mapping(b.SiteID)})
	case opMappings:
		return deps.Imports.ListMappings(ctx, imports.ListMappingsRequest{SiteID: b.SiteID})
	case opDeleteMapping:
		return deps.Imports.DeleteMapping(ctx, imports.DeleteMappingRequest{ID: in.MappingID})
	default:
		return nil, errors.New(errors.Invalid, "imports takes the op inspect, preview, apply, export, save_mapping, mappings or delete_mapping").
			WithDetail("field", "op")
	}
}

func (deps Deps) importRewrites(ctx context.Context, b Binding, in *importsArgs) (string, error) {
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
		row := &previewed.Report.Pages[i]
		if row.Action == string(imports.ActionUpdate) && rewriting[row.PageType] {
			typed = append(typed, row.Path)
		}
	}
	if len(typed) == 0 {
		return "", nil
	}
	return "Apply the import, which types " + strconv.Itoa(len(typed)) + " pages already in the map (" +
		strings.Join(typed, ", ") + ") into a type whose scope rewrites: their own text will be rewritten by the next run", nil
}
