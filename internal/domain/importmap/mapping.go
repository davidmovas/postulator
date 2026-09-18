package importmap

import (
	"strings"
	"time"
)

const (
	DefaultKeywordSeparator = ","
	DefaultAnchorSeparator  = "|"
	DefaultListSeparator    = ","
)

type Options struct {
	PathPrefixStrip  string `json:"pathPrefixStrip,omitempty"`
	KeywordSeparator string `json:"keywordSeparator,omitempty"`
	AnchorSeparator  string `json:"anchorSeparator,omitempty"`
	ListSeparator    string `json:"listSeparator,omitempty"`
}

func DefaultOptions() Options {
	return Options{
		KeywordSeparator: DefaultKeywordSeparator,
		AnchorSeparator:  DefaultAnchorSeparator,
		ListSeparator:    DefaultListSeparator,
	}
}

func (o Options) OrDefault() Options {
	o.PathPrefixStrip = strings.TrimSpace(o.PathPrefixStrip)
	if o.KeywordSeparator == "" {
		o.KeywordSeparator = DefaultKeywordSeparator
	}
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
	switch field {
	case FieldKeywords:
		return ready.KeywordSeparator
	case FieldAnchors:
		return ready.AnchorSeparator
	default:
		return ready.ListSeparator
	}
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
	case len(m.Columns) == 0:
		return Mapping{}, invalid("mapping must map at least one column", "columns")
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

	_, hasPath := columns[FieldPath]
	_, hasEntity := columns[FieldEntity]
	if !hasPath && !hasEntity {
		return Mapping{}, invalid("mapping must carry a path or an entity column", "columns")
	}

	m.Columns = columns
	m.Options = m.Options.OrDefault()
	return m, nil
}

type Binding struct {
	index   map[Field]int
	options Options
}

func (m Mapping) Bind(headers []string) (Binding, error) {
	positions := make(map[string]int, len(headers))
	for i, header := range headers {
		key := normalizeHeader(header)
		if key == "" {
			continue
		}
		if _, taken := positions[key]; !taken {
			positions[key] = i
		}
	}

	index := make(map[Field]int, len(m.Columns))
	for _, field := range fields {
		column, mapped := m.Columns[field]
		if !mapped {
			continue
		}
		at, found := positions[normalizeHeader(column)]
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
	return Binding{index: index, options: m.Options.OrDefault()}, nil
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
	if !mapped || at >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[at])
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
		if at < len(row) && strings.TrimSpace(row[at]) != "" {
			return false
		}
	}
	return true
}
