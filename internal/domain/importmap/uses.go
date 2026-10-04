package importmap

type Use string

const (
	UseField   Use = "field"
	UseLevel   Use = "level"
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
	for _, column := range m.Options.NoteColumns {
		mark(column, UseNote, "")
	}
	for _, column := range m.Options.IndentColumns {
		mark(column, UseIndent, "")
	}
	return out
}
