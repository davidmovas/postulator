package importmap

type Use string

const (
	UseField   Use = "field"
	UseLevel   Use = "level"
	UseTrail   Use = "trail"
	UseNote    Use = "note"
	UseIndent  Use = "indent"
	UseIgnored Use = "ignored"
)

type ColumnUse struct {
	Header string
	Use    Use
	Field  Field
}

func (m Mapping) Uses(headers []string) []ColumnUse {
	positions := indexHeaders(headers)
	out := make([]ColumnUse, len(headers))
	for i, header := range headers {
		out[i] = ColumnUse{Header: header, Use: UseIgnored}
	}

	mark := func(column string, use Use, field Field) {
		at, found := positions.find(column)
		if !found || out[at].Use != UseIgnored {
			return
		}
		out[at].Use, out[at].Field = use, field
	}
	for _, field := range fields {
		if column, mapped := m.Columns[field]; mapped {
			mark(column, UseField, field)
		}
	}
	for _, column := range rootColumns(m.Options.LevelColumns) {
		mark(column, UseLevel, "")
	}
	for _, column := range levelColumns(m.Options.LevelColumns) {
		mark(column, UseTrail, "")
	}
	for _, column := range m.Options.NoteColumns {
		mark(column, UseNote, "")
	}
	for _, column := range m.Options.IndentColumns {
		mark(column, UseIndent, "")
	}
	return out
}

func (m Mapping) Dropped(headers []string, field Field) []string {
	if _, read := m.Columns[field]; !read {
		return nil
	}
	out := make([]string, 0)
	for _, use := range m.Uses(headers) {
		if detected, known := Detect(use.Header); known && detected == field && use.Use == UseIgnored {
			out = append(out, use.Header)
		}
	}
	return out
}
