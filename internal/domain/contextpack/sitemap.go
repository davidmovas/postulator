package contextpack

import (
	"slices"
	"strings"

	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
)

type BodyState string

const (
	BodyEmpty   BodyState = "empty"
	BodyClient  BodyState = "client"
	BodyOurs    BodyState = "ours"
	BodyMixed   BodyState = "mixed"
	BodyDynamic BodyState = "dynamic"
)

func BodyStates() []BodyState {
	return []BodyState{BodyEmpty, BodyClient, BodyOurs, BodyMixed, BodyDynamic}
}

type MapNode struct {
	Page         Ref                     `json:"page"`
	ParentID     string                  `json:"parentId"`
	Why          *ParentWhy              `json:"why,omitempty"`
	Depth        int                     `json:"depth"`
	EntityID     string                  `json:"entityId,omitempty"`
	Status       pagemap.Status          `json:"status"`
	Twin         *Ref                    `json:"twin,omitempty"`
	ParentReason pagemap.ParentReason    `json:"parentReason"`
	ParentMethod pagemap.ParentMethod    `json:"parentMethod"`
	Technical    pagemap.TechnicalReason `json:"technical"`
	ReadOnly     pagemap.ReadOnlyReason  `json:"readOnly"`
	Body         BodyState               `json:"body"`
	LinksIn      int                     `json:"linksIn"`
	LinksOut     int                     `json:"linksOut"`
	Findings     int                     `json:"findings"`
	Children     int                     `json:"children"`
	Descendants  int                     `json:"descendants"`
	Trail        []Ref                   `json:"trail,omitempty"`
}

type MapQuery struct {
	Under  string
	Text   string
	TypeID string
}

type Place struct {
	Ref    Ref
	TypeID string
}

type Map struct {
	snapshot Snapshot
	links    map[string]LinkFacts
	rows     []pagemap.Page
	parent   map[string]string
	children map[string][]string
	twin     map[string]string
	folded   map[string]string
	open     map[string]int
}

func (s Snapshot) Map(links map[string]LinkFacts) Map {
	m := Map{
		snapshot: s, links: links, parent: map[string]string{}, children: map[string][]string{},
		twin: map[string]string{}, folded: map[string]string{}, open: map[string]int{},
	}
	for i := range s.Findings {
		if held := &s.Findings[i]; held.Status == finding.Open && held.PageID != nil {
			m.open[*held.PageID]++
		}
	}
	live := s.livePages()
	m.pair(live)
	m.rows = slices.DeleteFunc(live, func(page pagemap.Page) bool {
		_, gone := m.folded[page.ID]
		return gone
	})
	for i := range m.rows {
		page := &m.rows[i]
		parent := s.parentOf(*page)
		if hub, held := m.folded[parent]; held {
			parent = hub
		}
		if parent == page.ID {
			parent = ""
		}
		m.parent[page.ID] = parent
	}
	m.cutLoops()
	for i := range m.rows {
		parent := m.parent[m.rows[i].ID]
		m.children[parent] = append(m.children[parent], m.rows[i].ID)
	}
	return m
}

func (m Map) pair(live []pagemap.Page) {
	for i := range live {
		term := live[i]
		if !term.WPType.Term() {
			continue
		}
		hub, paired := m.snapshot.Index.HubTwin(term)
		if !paired {
			continue
		}
		if terms := m.snapshot.Index.TermTwins(hub); len(terms) != 1 || terms[0].ID != term.ID {
			continue
		}
		m.twin[hub.ID], m.folded[term.ID] = term.ID, hub.ID
	}
}

func (m Map) cutLoops() {
	for i := range m.rows {
		seen := map[string]struct{}{}
		at := m.rows[i].ID
		for at != "" {
			if _, looped := seen[at]; looped {
				m.parent[m.firstOfLoop(at)] = ""
				break
			}
			seen[at] = struct{}{}
			at = m.parent[at]
		}
	}
}

func (m Map) firstOfLoop(start string) string {
	first, firstPath := start, m.pathOf(start)
	for at := m.parent[start]; at != start && at != ""; at = m.parent[at] {
		if path := m.pathOf(at); path < firstPath {
			first, firstPath = at, path
		}
	}
	return first
}

func (m Map) pathOf(pageID string) string {
	page, _ := m.snapshot.Index.ByID(pageID)
	return page.Path
}

func (m Map) row(pageID string) (string, bool) {
	if hub, held := m.folded[pageID]; held {
		pageID = hub
	}
	_, isRow := m.parent[pageID]
	return pageID, isRow
}

func (m Map) Node(pageID string) (MapNode, bool) {
	id, isRow := m.row(pageID)
	if !isRow {
		return MapNode{}, false
	}
	return m.node(id), true
}

func (m Map) Children(parentID string) []MapNode {
	if parentID != "" {
		id, isRow := m.row(parentID)
		if !isRow {
			return []MapNode{}
		}
		parentID = id
	}
	ids := m.children[parentID]
	out := make([]MapNode, 0, len(ids))
	for _, id := range ids {
		out = append(out, m.node(id))
	}
	return out
}

func (m Map) Matches(q MapQuery) []MapNode {
	var scope map[string]struct{}
	if q.Under != "" {
		scope = make(map[string]struct{})
		for _, id := range m.Under(q.Under) {
			scope[id] = struct{}{}
		}
	}
	text := graph.Key(q.Text)
	out := make([]MapNode, 0)
	for i := range m.rows {
		page := m.rows[i]
		if _, inside := scope[page.ID]; scope != nil && !inside {
			continue
		}
		if q.TypeID != "" && !m.typed(page, q.TypeID) {
			continue
		}
		if text != "" && !m.says(page, text) {
			continue
		}
		node := m.node(page.ID)
		node.Trail = m.trail(page.ID)
		out = append(out, node)
	}
	return out
}

func (m Map) Under(pageID string) []string {
	root, isRow := m.row(pageID)
	if !isRow {
		return []string{}
	}
	out := make([]string, 0)
	queue := []string{root}
	seen := map[string]struct{}{}
	for len(queue) > 0 {
		next := queue[0]
		queue = queue[1:]
		if _, done := seen[next]; done {
			continue
		}
		seen[next] = struct{}{}
		out = append(out, next)
		if term, paired := m.twin[next]; paired {
			out = append(out, term)
		}
		queue = append(queue, m.children[next]...)
	}
	return out
}

func (m Map) Place(pageID string) (Place, bool) {
	page, found := m.snapshot.Index.ByID(pageID)
	if !found {
		return Place{}, false
	}
	return Place{Ref: m.snapshot.ref(page, false), TypeID: m.snapshot.typeOf(page).ID}, true
}

func (m Map) Trail(pageID string) []Ref {
	id, isRow := m.row(pageID)
	if !isRow {
		return []Ref{}
	}
	return m.trail(id)
}

func (m Map) Type(typeID string) (pagetype.Type, bool) {
	return m.snapshot.Types.ByID(typeID)
}

func (m Map) node(pageID string) MapNode {
	page, _ := m.snapshot.Index.ByID(pageID)
	linked := m.links[pageID]
	node := MapNode{
		Page: m.snapshot.ref(page, false), ParentReason: page.ParentReason, ParentMethod: page.ParentMethod,
		Technical: page.TechnicalReason, ReadOnly: page.ReadOnlyReason, Body: BodyEmpty,
		LinksIn: linked.Inbound, LinksOut: linked.Outbound, Findings: m.open[pageID],
		Children: len(m.children[pageID]), Descendants: m.descendants(pageID), Status: page.Status,
	}
	if page.EntityID != nil {
		node.EntityID = *page.EntityID
	}
	if termID, paired := m.twin[pageID]; paired {
		term, _ := m.snapshot.Index.ByID(termID)
		ref := m.snapshot.ref(term, false)
		node.Twin = &ref
		node.Findings += m.open[termID]
		node.LinksIn += m.links[termID].Inbound
		node.LinksOut += m.links[termID].Outbound
	}
	return node
}

func (m Map) descendants(pageID string) int {
	count := 0
	for _, id := range m.Under(pageID) {
		if _, folded := m.folded[id]; !folded && id != pageID {
			count++
		}
	}
	return count
}

func (m Map) trail(pageID string) []Ref {
	out := make([]Ref, 0)
	seen := map[string]struct{}{pageID: {}}
	for at := m.parent[pageID]; at != ""; at = m.parent[at] {
		if _, looped := seen[at]; looped {
			break
		}
		seen[at] = struct{}{}
		page, _ := m.snapshot.Index.ByID(at)
		out = append(out, m.snapshot.ref(page, false))
	}
	slices.Reverse(out)
	return out
}

func (m Map) typed(page pagemap.Page, typeID string) bool {
	if m.snapshot.typeOf(page).ID == typeID {
		return true
	}
	termID, paired := m.twin[page.ID]
	if !paired {
		return false
	}
	term, _ := m.snapshot.Index.ByID(termID)
	return m.snapshot.typeOf(term).ID == typeID
}

func (m Map) says(page pagemap.Page, text string) bool {
	if m.mentions(page, text) {
		return true
	}
	termID, paired := m.twin[page.ID]
	if !paired {
		return false
	}
	term, _ := m.snapshot.Index.ByID(termID)
	return m.mentions(term, text)
}

func (m Map) mentions(page pagemap.Page, text string) bool {
	return strings.Contains(graph.Key(m.snapshot.titleOf(page)), text) || strings.Contains(graph.Key(page.Path), text)
}

func BodyOf(raw string, written []string) BodyState {
	ours := make(map[string]struct{}, len(written))
	for _, hash := range written {
		ours[hash] = struct{}{}
	}
	var mine, theirs, dynamic int
	blocks := block.Parse(raw).Blocks
	for i := range blocks {
		held := &blocks[i]
		_, isOurs := ours[held.Hash]
		switch {
		case isOurs:
			mine++
		case held.Dynamic || held.Kind == block.Shortcode:
			dynamic++
		case strings.TrimSpace(held.Raw) != "":
			theirs++
		}
	}
	switch {
	case mine > 0 && theirs > 0:
		return BodyMixed
	case mine > 0:
		return BodyOurs
	case theirs > 0:
		return BodyClient
	case dynamic > 0:
		return BodyDynamic
	default:
		return BodyEmpty
	}
}
