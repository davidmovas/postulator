package contextpack

import (
	"strings"

	"github.com/davidmovas/postulator/internal/domain/graph"
)

const TrailSeparator = "/"

type MapRow struct {
	PageID       string
	SiteID       string
	ParentID     string
	Path         string
	Depth        int
	Trail        string
	Children     int
	Descendants  int
	TwinID       string
	HubID        string
	Row          bool
	TypeID       string
	TwinTypeID   string
	Title        string
	TitleKey     string
	PathKey      string
	TwinTitleKey string
	TwinPathKey  string
}

type MapRowQuery struct {
	ParentID *string
	MaxDepth *int
	SiteID   string
	Under    string
	Text     string
	TypeID   string
	Within   string
	Except   string
	MinDepth int
	ByDepth  bool
}

func (r MapRow) Ancestors() []string {
	ids := strings.Split(strings.TrimSuffix(r.Trail, TrailSeparator), TrailSeparator)
	if len(ids) <= 1 {
		return []string{}
	}
	return ids[:len(ids)-1]
}

func (r MapRow) Mentions(text string) bool {
	return strings.Contains(r.TitleKey, text) || strings.Contains(r.PathKey, text) ||
		(r.TwinID != "" && (strings.Contains(r.TwinTitleKey, text) || strings.Contains(r.TwinPathKey, text)))
}

func (r MapRow) Typed(typeID string) bool {
	return r.TypeID == typeID || (r.TwinID != "" && r.TwinTypeID == typeID)
}

func (r MapRow) Holding(rows []MapRow, typeID string) bool {
	if r.Typed(typeID) {
		return true
	}
	for i := range rows {
		if rows[i].SiteID == r.SiteID && rows[i].TypeID == typeID && len(rows[i].Trail) > len(r.Trail) && strings.HasPrefix(rows[i].Trail, r.Trail) {
			return true
		}
	}
	return false
}

func (q MapRowQuery) Holds(r MapRow) bool {
	switch {
	case r.SiteID != q.SiteID, !r.Row:
		return false
	case q.ParentID != nil && r.ParentID != *q.ParentID:
		return false
	case q.Except != "" && r.PageID == q.Except:
		return false
	case r.Depth < q.MinDepth, q.MaxDepth != nil && r.Depth > *q.MaxDepth:
		return false
	case q.Under != "" && !strings.HasPrefix(r.Trail, q.Under):
		return false
	case q.TypeID != "" && !r.Typed(q.TypeID):
		return false
	default:
		return q.Text == "" || r.Mentions(q.Text)
	}
}

func (m Map) Rows() []MapRow {
	pages := m.snapshot.Index.Pages()
	trails := make(map[string]string, len(m.rows))
	depths := make(map[string]int, len(m.rows))
	sizes := make(map[string]int, len(m.rows))
	var visit func(id, trail string, depth int) int
	visit = func(id, trail string, depth int) int {
		trail += id + TrailSeparator
		trails[id], depths[id] = trail, depth
		size := 0
		for _, child := range m.children[id] {
			size += 1 + visit(child, trail, depth+1)
		}
		sizes[id] = size
		return size
	}
	for _, root := range m.children[""] {
		visit(root, "", 0)
	}
	out := make([]MapRow, 0, len(pages))
	for i := range pages {
		page := pages[i]
		title := m.snapshot.titleOf(page)
		row := MapRow{
			PageID: page.ID, SiteID: page.SiteID, Path: page.Path, TypeID: m.snapshot.typeOf(page).ID,
			Title: title, TitleKey: graph.Key(title), PathKey: graph.Key(page.Path),
		}
		_, row.Row = m.parent[page.ID]
		switch hub, folded := m.folded[page.ID]; {
		case row.Row:
			row.ParentID, row.Trail, row.Depth = m.parent[page.ID], trails[page.ID], depths[page.ID]
			row.Children, row.Descendants = len(m.children[page.ID]), sizes[page.ID]
			if termID, paired := m.twin[page.ID]; paired {
				term, _ := m.snapshot.Index.ByID(termID)
				twinTitle := m.snapshot.titleOf(term)
				row.TwinID, row.TwinTypeID = termID, m.snapshot.typeOf(term).ID
				row.TwinTitleKey, row.TwinPathKey = graph.Key(twinTitle), graph.Key(term.Path)
			}
		case folded:
			row.HubID, row.ParentID = hub, hub
			row.Trail, row.Depth = trails[hub]+page.ID+TrailSeparator, depths[hub]+1
		}
		out = append(out, row)
	}
	return out
}
