package importmap

import (
	"slices"
	"strings"
	"time"

	"github.com/davidmovas/postulator/internal/domain/pagemap"
)

const (
	DefaultAnchorSeparator = "|"
	DefaultListSeparator   = ","
)

var placeholderCells = []string{"-", "—", "–", "n/a", "na", "none"}

type Options struct {
	PathPrefixStrip string   `json:"pathPrefixStrip,omitempty"`
	AnchorSeparator string   `json:"anchorSeparator,omitempty"`
	ListSeparator   string   `json:"listSeparator,omitempty"`
	Sheets          []string `json:"sheets,omitempty"`
	IndentColumns   []string `json:"indentColumns,omitempty"`
	LevelColumns    []string `json:"levelColumns,omitempty"`
	NoteColumns     []string `json:"noteColumns,omitempty"`
	RowType         RowType  `json:"rowType,omitempty"`
	NoHeader        bool     `json:"noHeader,omitempty"`
}

type RowType string

const (
	RowPages    RowType = "pages"
	RowProducts RowType = "products"
	RowKind     RowType = "kind"
)

func (r RowType) Valid() bool {
	switch r {
	case "", RowPages, RowProducts, RowKind:
		return true
	default:
		return false
	}
}

func unknownRowType(rowType RowType) error {
	return invalid("the row type is not recognized; rows are pages, products or read by their kind", "rowType").
		WithDetail("rowType", string(rowType))
}

func DefaultOptions() Options {
	return Options{
		AnchorSeparator: DefaultAnchorSeparator,
		ListSeparator:   DefaultListSeparator,
	}
}

func (o Options) OrDefault() Options {
	o.PathPrefixStrip = strings.TrimSpace(o.PathPrefixStrip)
	if o.AnchorSeparator == "" {
		o.AnchorSeparator = DefaultAnchorSeparator
	}
	if o.ListSeparator == "" {
		o.ListSeparator = DefaultListSeparator
	}
	return o
}

func (o Options) Separator(field Field) string {
	ready := o.OrDefault()
	if field == FieldAnchors {
		return ready.AnchorSeparator
	}
	return ready.ListSeparator
}

func (o Options) Split(field Field, raw string) []string {
	out := make([]string, 0, strings.Count(raw, o.Separator(field))+1)
	for _, part := range strings.Split(raw, o.Separator(field)) {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func (o Options) Join(field Field, values []string) string {
	return strings.Join(values, o.Separator(field))
}

func (o Options) Strip(path string) string {
	if o.PathPrefixStrip == "" {
		return path
	}
	if len(path) >= len(o.PathPrefixStrip) && strings.EqualFold(path[:len(o.PathPrefixStrip)], o.PathPrefixStrip) {
		return path[len(o.PathPrefixStrip):]
	}
	return path
}

type Mapping struct {
	ID        string
	SiteID    string
	Name      string
	Columns   map[Field]string
	Options   Options
	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewMapping(m Mapping) (Mapping, error) {
	m.Name = strings.TrimSpace(m.Name)
	switch {
	case m.ID == "":
		return Mapping{}, invalid("mapping id must not be empty", "id")
	case m.SiteID == "":
		return Mapping{}, invalid("mapping site id must not be empty", "siteId")
	case m.Name == "":
		return Mapping{}, invalid("mapping name must not be empty", "name")
	case len(m.Columns) == 0 && len(m.Options.IndentColumns) == 0 && len(m.Options.LevelColumns) == 0:
		return Mapping{}, invalid("mapping must map at least one column", "columns")
	case !m.Options.RowType.Valid():
		return Mapping{}, unknownRowType(m.Options.RowType)
	}

	columns := make(map[Field]string, len(m.Columns))
	for field, column := range m.Columns {
		trimmed := strings.TrimSpace(column)
		switch {
		case !field.Valid():
			return Mapping{}, invalid("import field is not recognized", "columns").WithDetail("importField", string(field))
		case trimmed == "":
			return Mapping{}, invalid("mapped column must not be empty", "columns").WithDetail("importField", string(field))
		}
		columns[field] = trimmed
	}

	indent := make([]string, 0, len(m.Options.IndentColumns))
	for _, column := range m.Options.IndentColumns {
		trimmed := strings.TrimSpace(column)
		if trimmed == "" {
			return Mapping{}, invalid("an indent column must not be empty", "indentColumns")
		}
		indent = append(indent, trimmed)
	}
	if len(indent) == 0 {
		indent = nil
	}
	m.Options.IndentColumns = indent

	levels, err := namedColumns(m.Options.LevelColumns, "levelColumns", "a level column")
	if err != nil {
		return Mapping{}, err
	}
	notes, err := namedColumns(m.Options.NoteColumns, "noteColumns", "a note column")
	if err != nil {
		return Mapping{}, err
	}
	if apartErr := apart(columns, levels, notes); apartErr != nil {
		return Mapping{}, apartErr
	}
	m.Options.LevelColumns, m.Options.NoteColumns = levels, notes

	_, hasPath := columns[FieldPath]
	_, hasEntity := columns[FieldEntity]
	if !hasPath && !hasEntity && len(indent) == 0 && len(levels) == 0 {
		return Mapping{}, invalid("mapping must carry a path, an entity column, indent columns or level columns", "columns")
	}

	m.Columns = columns
	m.Options = m.Options.OrDefault()
	return m, nil
}

func namedColumns(raw []string, field, what string) ([]string, error) {
	out := make([]string, 0, len(raw))
	for _, column := range raw {
		trimmed := strings.TrimSpace(column)
		if trimmed == "" {
			return nil, invalid(what+" must not be empty", field)
		}
		out = append(out, trimmed)
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

func apart(columns map[Field]string, levels, notes []string) error {
	taken := make(map[string]string, len(columns)+len(levels)+len(notes))
	claim := func(column, use, field string) error {
		key := strings.ToLower(column)
		if held, seen := taken[key]; seen {
			return invalid("the column "+column+" is read twice, as "+held+" and as "+use, field).
				WithDetail("column", column)
		}
		taken[key] = use
		return nil
	}
	for _, field := range fields {
		if column, mapped := columns[field]; mapped {
			taken[strings.ToLower(column)] = string(field)
		}
	}
	for _, column := range levels {
		if err := claim(column, "a level", "levelColumns"); err != nil {
			return err
		}
	}
	for _, column := range notes {
		if err := claim(column, "a note", "noteColumns"); err != nil {
			return err
		}
	}
	return nil
}

type noteColumn struct {
	label string
	at    int
}

type levelColumn struct {
	at       int
	category bool
}

type Level struct {
	Name     string
	Category bool
}

type Binding struct {
	index   map[Field]int
	indent  []int
	levels  []levelColumn
	notes   []noteColumn
	options Options
}

type headerIndex struct {
	exact      map[string]int
	normalized map[string]int
}

func indexHeaders(headers []string) headerIndex {
	index := headerIndex{exact: make(map[string]int, len(headers)), normalized: make(map[string]int, len(headers))}
	for i, header := range headers {
		if trimmed := strings.TrimSpace(header); trimmed != "" {
			if _, taken := index.exact[trimmed]; !taken {
				index.exact[trimmed] = i
			}
		}
		if key := Words(header); key != "" {
			if _, taken := index.normalized[key]; !taken {
				index.normalized[key] = i
			}
		}
	}
	return index
}

func (h headerIndex) find(column string) (int, bool) {
	if at, found := h.exact[strings.TrimSpace(column)]; found {
		return at, true
	}
	at, found := h.normalized[Words(column)]
	return at, found
}

func (m Mapping) Bind(headers []string) (Binding, error) {
	positions := indexHeaders(headers)

	index := make(map[Field]int, len(m.Columns))
	for _, field := range fields {
		column, mapped := m.Columns[field]
		if !mapped {
			continue
		}
		at, found := positions.find(column)
		if !found {
			return Binding{}, invalid("the mapped column is not in the file", "columns").
				WithDetail("importField", string(field)).WithDetail("column", column)
		}
		index[field] = at
	}
	for field := range m.Columns {
		if !field.Valid() {
			return Binding{}, invalid("import field is not recognized", "columns").WithDetail("importField", string(field))
		}
	}
	if !m.Options.RowType.Valid() {
		return Binding{}, unknownRowType(m.Options.RowType)
	}

	indent, err := bindAll(positions, m.Options.IndentColumns, "indentColumns", "the indent column is not in the file")
	if err != nil {
		return Binding{}, err
	}
	levels, err := bindLevels(positions, m.Options.LevelColumns)
	if err != nil {
		return Binding{}, err
	}
	notes, err := bindNotes(positions, m.Options.NoteColumns)
	if err != nil {
		return Binding{}, err
	}
	return Binding{index: index, indent: indent, levels: levels, notes: notes, options: m.Options.OrDefault()}, nil
}

func bindLevels(positions headerIndex, columns []string) ([]levelColumn, error) {
	found, err := bindAll(positions, columns, "levelColumns", "the level column is not in the file")
	if err != nil {
		return nil, err
	}
	levels := make([]levelColumn, 0, len(found))
	for i, at := range found {
		levels = append(levels, levelColumn{at: at, category: !rootLevel(columns[i])})
	}
	return levels, nil
}

func bindNotes(positions headerIndex, columns []string) ([]noteColumn, error) {
	found, err := bindAll(positions, columns, "noteColumns", "the note column is not in the file")
	if err != nil {
		return nil, err
	}
	notes := make([]noteColumn, 0, len(found))
	for i, at := range found {
		notes = append(notes, noteColumn{label: strings.TrimSpace(columns[i]), at: at})
	}
	return notes, nil
}

func bindAll(positions headerIndex, columns []string, field, missing string) ([]int, error) {
	out := make([]int, 0, len(columns))
	for _, column := range columns {
		at, found := positions.find(column)
		if !found {
			return nil, invalid(missing, field).WithDetail("column", column)
		}
		out = append(out, at)
	}
	return out, nil
}

func cellAt(row []string, at int) string {
	if at >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[at])
}

func levelCell(row []string, at int) string {
	value := cellAt(row, at)
	if slices.Contains(placeholderCells, strings.ToLower(value)) {
		return ""
	}
	return value
}

func (b Binding) Levels(row []string) []Level {
	out := make([]Level, 0, len(b.levels))
	for _, column := range b.levels {
		if name := levelCell(row, column.at); name != "" {
			out = append(out, Level{Name: name, Category: column.category})
		}
	}
	return out
}

func (b Binding) Notes(row []string) []pagemap.Note {
	notes := make([]pagemap.Note, 0, len(b.notes))
	for _, column := range b.notes {
		notes = append(notes, pagemap.Note{Label: column.label, Text: cellAt(row, column.at)})
	}
	return pagemap.NewNotes(notes)
}

func (b Binding) Options() Options {
	return b.options
}

func (b Binding) Has(field Field) bool {
	_, mapped := b.index[field]
	return mapped
}

func (b Binding) Text(row []string, field Field) string {
	at, mapped := b.index[field]
	if !mapped {
		return ""
	}
	return cellAt(row, at)
}

func (b Binding) List(row []string, field Field) []string {
	raw := b.Text(row, field)
	if raw == "" {
		return nil
	}
	return b.options.Split(field, raw)
}

func (b Binding) Path(row []string) string {
	return strings.TrimSpace(b.options.Strip(b.Text(row, FieldPath)))
}

func (b Binding) Blank(row []string) bool {
	for _, at := range b.index {
		if cellAt(row, at) != "" {
			return false
		}
	}
	if len(b.Levels(row)) > 0 {
		return false
	}
	for _, column := range b.notes {
		if cellAt(row, column.at) != "" {
			return false
		}
	}
	return true
}
