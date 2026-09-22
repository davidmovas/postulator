package tools

import "github.com/davidmovas/postulator/internal/application/imports"

type columnArgs struct {
	Field  string `json:"field" enum:"path,title,h1,primary_keyword,keywords,anchors,entity,entity_kind,parent_entity,related,page_kind,meta_title,meta_description,wp_type" description:"The page field this column fills"`
	Column string `json:"column" description:"The spreadsheet header the field is read from"`
}

type mappingArgs struct {
	ID      string          `json:"id,omitempty" description:"The id of a saved mapping to reuse, left out for a mapping given here"`
	Name    string          `json:"name,omitempty" description:"What to call the mapping when it is saved"`
	Columns []columnArgs    `json:"columns" description:"Which spreadsheet column fills which page field"`
	Options imports.Options `json:"options" description:"How to read the cells: what to strip from a path and what separates a list"`
}

func (m mappingArgs) mapping(siteID string) imports.Mapping {
	columns := make(map[string]string, len(m.Columns))
	for _, column := range m.Columns {
		columns[column.Field] = column.Column
	}
	return imports.Mapping{ID: m.ID, SiteID: siteID, Name: m.Name, Columns: columns, Options: m.Options}
}
