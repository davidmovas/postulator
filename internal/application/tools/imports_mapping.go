package tools

import (
	"github.com/davidmovas/postulator/internal/application/imports"
	"github.com/davidmovas/postulator/internal/domain/importmap"
)

type columnArgs struct {
	Field  string `json:"field" enum:"path,title,h1,primary_keyword,keywords,anchors,entity,entity_kind,parent_entity,related,page_kind,meta_title,meta_description,wp_type,own_entity" description:"Page field"`
	Column string `json:"column" description:"Spreadsheet header it is read from"`
}

type mappingArgs struct {
	ID      string           `json:"id,omitempty" description:"Saved mapping id; with no columns it is used, but for the row type and sheets given here"`
	Name    string           `json:"name,omitempty" description:"Name to save it under"`
	Columns []columnArgs     `json:"columns,omitempty" description:"Column per page field; none detects them from the headers"`
	Options *imports.Options `json:"options,omitempty" description:"How to read the cells; left out, the defaults"`
}

type sheetArgs struct {
	Sheet     string            `json:"sheet" description:"Sheet name from imports_inspect"`
	MappingID string            `json:"mappingId,omitempty" description:"Saved mapping id; left out, the columns are detected"`
	RowType   importmap.RowType `json:"rowType,omitempty" enum:"pages,products,kind" description:"What a new row becomes; left out, pages"`
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

func (m mappingArgs) mapping(siteID string) imports.Mapping {
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
