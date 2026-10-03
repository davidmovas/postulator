package tools

import (
	"github.com/davidmovas/postulator/internal/application/imports"
	"github.com/davidmovas/postulator/internal/domain/importmap"
)

type columnArgs struct {
	Field  string `json:"field" enum:"path,title,h1,primary_keyword,keywords,anchors,entity,entity_kind,parent_entity,related,page_kind,meta_title,meta_description,wp_type,own_entity" description:"The page field this column fills"`
	Column string `json:"column" description:"The spreadsheet header the field is read from"`
}

type mappingArgs struct {
	ID      string           `json:"id,omitempty" description:"A saved mapping's id; with no columns an import uses it, taking only the row type and sheets given here"`
	Name    string           `json:"name,omitempty" description:"What to call the mapping when it is saved"`
	Columns []columnArgs     `json:"columns,omitempty" description:"Which spreadsheet column fills which page field; with none, an import detects them from the headers"`
	Options *imports.Options `json:"options,omitempty" description:"How to read the cells: what to strip from a path and what separates a list; leave it out for the defaults"`
}

type sheetArgs struct {
	Sheet     string            `json:"sheet" description:"The sheet, as imports_inspect named it"`
	MappingID string            `json:"mappingId,omitempty" description:"A saved mapping's id; leave it out to detect the columns"`
	RowType   importmap.RowType `json:"rowType,omitempty" enum:"pages,products,kind" description:"What a new row becomes, pages by default"`
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
