package tools

import "github.com/davidmovas/postulator/internal/application/imports"

type columnArgs struct {
	Field  string `json:"field" description:"one of path, title, h1, primary_keyword, keywords, anchors, entity, entity_kind, parent_entity, related, page_kind, meta_title, meta_description, wp_type"`
	Column string `json:"column" description:"the spreadsheet header the field is read from"`
}

type mappingArgs struct {
	ID      string          `json:"id,omitempty"`
	Name    string          `json:"name,omitempty"`
	Columns []columnArgs    `json:"columns"`
	Options imports.Options `json:"options"`
}

func (m mappingArgs) mapping(siteID string) imports.Mapping {
	columns := make(map[string]string, len(m.Columns))
	for _, column := range m.Columns {
		columns[column.Field] = column.Column
	}
	return imports.Mapping{ID: m.ID, SiteID: siteID, Name: m.Name, Columns: columns, Options: m.Options}
}
