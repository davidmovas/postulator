package contextpack

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/keyword"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
)

type TopicSkip string

const (
	TopicNoBreadcrumbs TopicSkip = "no_breadcrumbs"
	TopicUnknownLink   TopicSkip = "unknown_link"
	TopicAmbiguousName TopicSkip = "ambiguous_name"
	TopicNoName        TopicSkip = "no_name"
	TopicNameTaken     TopicSkip = "name_taken"
	TopicDisagrees     TopicSkip = "disagrees"
	TopicParentSkipped TopicSkip = "parent_skipped"
	TopicRemoved       TopicSkip = "removed"
)

type TopicAction string

const (
	TopicCreate TopicAction = "create"
	TopicAdopt  TopicAction = "adopt"
	TopicRecord TopicAction = "record"
)

type TopicNode struct {
	Subject        string
	Action         TopicAction
	PageID         string
	Name           string
	Parent         string
	ParentName     string
	ParentEntityID string
	EntityID       string
	EdgeMissing    bool
	Facts          []graph.Fact
	Keywords       keyword.List
	Order          string
}

type TopicMiss struct {
	PageID string
	Path   string
	Skip   TopicSkip
}

type TopicPlan struct {
	Writes   []TopicNode
	Unplaced []TopicMiss
}

const (
	pageSubject  = "page:"
	topicSubject = "topic:"
)

type topicDraft struct {
	subject  string
	page     *pagemap.Page
	parent   string
	own      bool
	ownSkip  TopicSkip
	unknown  TopicSkip
	ownLabel string
	labels   map[string]int
	votes    map[string]int
	name     string
	depth    int
	sticky   TopicSkip
	skip     TopicSkip
	action   TopicAction
	entityID string
	wanted   bool
}

func (d *topicDraft) label(name string) {
	if name = strings.TrimSpace(name); name != "" {
		d.labels[name]++
	}
}

func (d *topicDraft) sortKey() string {
	if d.page != nil {
		return d.page.Path
	}
	return d.name
}

func (d *topicDraft) placing() bool {
	return d.skip == "" && (d.action == TopicCreate || d.action == TopicAdopt)
}

type topicBuild struct {
	m        Map
	drafts   map[string]*topicDraft
	order    []*topicDraft
	built    map[string]struct{}
	entities []graph.Entity
	carried  map[string]struct{}
}

func (m Map) TopicMap(chains map[string]pagemap.Chain, built []string) TopicPlan {
	b := topicBuild{
		m: m, drafts: map[string]*topicDraft{}, built: make(map[string]struct{}, len(built)),
		entities: m.snapshot.Graph.Entities(), carried: map[string]struct{}{},
	}
	for _, subject := range built {
		b.built[subject] = struct{}{}
	}
	pages := m.snapshot.Index.Pages()
	for i := range pages {
		if pages[i].EntityID != nil {
			b.carried[*pages[i].EntityID] = struct{}{}
		}
	}
	for i := range pages {
		if chain, held := chains[pages[i].ID]; held && carriable(pages[i]) {
			b.walk(pages[i], chain)
		}
	}
	b.settle()
	for {
		b.resolve()
		b.want()
		if !b.disagree() {
			break
		}
	}
	return b.plan()
}

func carriable(page pagemap.Page) bool {
	return page.WPID != nil && live(page) && !page.WPType.Term() && page.Path != pagemap.RootPath && page.TechnicalReason == pagemap.TechnicalNone
}

func (b *topicBuild) pageDraft(page pagemap.Page) *topicDraft {
	subject := pageSubject + page.ID
	if held, found := b.drafts[subject]; found {
		return held
	}
	draft := &topicDraft{subject: subject, page: &page, labels: map[string]int{}, votes: map[string]int{}}
	b.drafts[subject] = draft
	return draft
}

func (b *topicBuild) topicDraft(name, parent string) *topicDraft {
	key := graph.Key(name)
	if key == "" {
		return nil
	}
	subject := topicSubject + strings.TrimPrefix(parent, topicSubject) + "/" + key
	if held, found := b.drafts[subject]; found {
		return held
	}
	draft := &topicDraft{subject: subject, parent: parent, labels: map[string]int{}, votes: map[string]int{}}
	b.drafts[subject] = draft
	return draft
}

func (b *topicBuild) linkDraft(link pagemap.ChainLink, parent string, known bool) (draft *topicDraft, name string) {
	name = link.Name
	if link.PageID != "" {
		id := link.PageID
		if hub, folded := b.m.folded[id]; folded {
			id = hub
		}
		page, found := b.m.snapshot.Index.ByID(id)
		if found && carriable(page) {
			return b.pageDraft(page), name
		}
		if found && name == "" {
			name = page.Title
		}
	}
	if !known {
		return nil, name
	}
	return b.topicDraft(name, parent), name
}

func (b *topicBuild) walk(page pagemap.Page, chain pagemap.Chain) {
	self := b.pageDraft(page)
	self.ownLabel = strings.TrimSpace(chain.Label)
	if chain.Break == pagemap.ChainNoTrail {
		return
	}
	parent, known, broke := "", true, TopicSkip("")
	for _, link := range chain.Links {
		if link.Break != pagemap.ChainWhole {
			known, broke = false, TopicSkip(link.Break)
			continue
		}
		draft, name := b.linkDraft(link, parent, known)
		if draft == nil {
			known = false
			continue
		}
		if draft == self {
			continue
		}
		draft.label(name)
		if draft.page != nil {
			if known {
				draft.votes[parent]++
			} else if draft.unknown == "" {
				draft.unknown = broke
			}
		}
		parent, known = draft.subject, true
	}
	switch {
	case known:
		self.own, self.parent = true, parent
	case broke != "":
		self.ownSkip = broke
	}
}

func (b *topicBuild) settle() {
	subjects := make([]string, 0, len(b.drafts))
	for subject := range b.drafts {
		subjects = append(subjects, subject)
	}
	slices.Sort(subjects)
	for _, subject := range subjects {
		d := b.drafts[subject]
		d.name = d.ownLabel
		if d.name == "" {
			d.name = mostUsed(d.labels)
		}
		if d.page != nil {
			b.place(d)
			if d.name == "" {
				d.name = strings.TrimSpace(b.m.snapshot.siteName(*d.page))
			}
		}
		if d.sticky == "" && graph.Key(d.name) == "" {
			d.sticky = TopicNoName
		}
	}
	b.cutLoops(subjects)
	depths := make(map[string]int, len(subjects))
	for _, subject := range subjects {
		b.depthOf(b.drafts[subject], depths)
		b.order = append(b.order, b.drafts[subject])
	}
	slices.SortFunc(b.order, func(x, y *topicDraft) int {
		return cmp.Or(cmp.Compare(x.depth, y.depth), strings.Compare(x.sortKey(), y.sortKey()), strings.Compare(x.subject, y.subject))
	})
}

func (b *topicBuild) place(d *topicDraft) {
	switch {
	case d.own:
	case d.ownSkip != "":
		d.sticky = d.ownSkip
	case len(d.votes) == 1:
		for parent := range d.votes {
			d.parent = parent
		}
		d.own = true
	case len(d.votes) > 1:
		d.sticky = TopicDisagrees
	default:
		d.sticky = cmp.Or(d.unknown, TopicNoBreadcrumbs)
	}
}

func mostUsed(labels map[string]int) string {
	best, count := "", 0
	for name, used := range labels {
		if used > count || used == count && name < best {
			best, count = name, used
		}
	}
	return best
}

func (b *topicBuild) cutLoops(subjects []string) {
	done := make(map[string]struct{}, len(subjects))
	for _, subject := range subjects {
		path := make([]string, 0)
		at := make(map[string]int)
		for current := subject; current != ""; {
			if _, finished := done[current]; finished {
				break
			}
			if first, looped := at[current]; looped {
				for _, member := range path[first:] {
					b.drafts[member].sticky, b.drafts[member].parent = TopicDisagrees, ""
				}
				break
			}
			draft, found := b.drafts[current]
			if !found {
				break
			}
			at[current] = len(path)
			path = append(path, current)
			current = draft.parent
		}
		for _, member := range path {
			done[member] = struct{}{}
		}
	}
}

func (b *topicBuild) depthOf(d *topicDraft, depths map[string]int) int {
	if depth, known := depths[d.subject]; known {
		return depth
	}
	depth := 0
	if parent, found := b.drafts[d.parent]; d.parent != "" && found {
		depth = b.depthOf(parent, depths) + 1
	}
	depths[d.subject], d.depth = depth, depth
	return depth
}

func (b *topicBuild) resolve() {
	for _, d := range b.order {
		d.skip, d.action, d.entityID = d.sticky, "", ""
	}
	for start := 0; start < len(b.order); {
		end := start
		for end < len(b.order) && b.order[end].depth == b.order[start].depth {
			end++
		}
		level := b.order[start:end]
		for _, d := range level {
			b.decide(d)
		}
		clashes(level)
		start = end
	}
}

func (b *topicBuild) decide(d *topicDraft) {
	if d.skip != "" {
		return
	}
	var parent *topicDraft
	if d.parent != "" {
		parent = b.drafts[d.parent]
	}
	if parent != nil && parent.skip != "" {
		d.skip = TopicParentSkipped
		return
	}
	_, recorded := b.built[d.subject]
	if d.page != nil && d.page.EntityID != nil {
		d.entityID = *d.page.EntityID
		if !recorded {
			d.action = TopicRecord
		}
		return
	}
	if d.page != nil && recorded {
		d.skip = TopicRemoved
		return
	}
	scope := ""
	if parent != nil {
		scope = parent.entityID
	}
	var existing graph.Entity
	held := false
	if parent == nil || parent.action != TopicCreate {
		existing, held = sameNameUnder(b.entities, scope, d.name)
	}
	switch {
	case d.page == nil && held:
		d.entityID = existing.ID
		if !recorded {
			d.action = TopicRecord
		}
	case d.page == nil && recorded:
		d.skip = TopicRemoved
	case !held:
		d.action = TopicCreate
	case b.takenBy(existing, d.page.ID):
		d.skip = TopicNameTaken
	default:
		d.action, d.entityID = TopicAdopt, existing.ID
	}
}

func (b *topicBuild) takenBy(existing graph.Entity, pageID string) bool {
	if _, used := b.carried[existing.ID]; used {
		return true
	}
	return existing.CanonicalPageID != nil && *existing.CanonicalPageID != pageID
}

func clashes(level []*topicDraft) {
	groups := map[string][]*topicDraft{}
	for _, d := range level {
		if d.placing() {
			key := d.parent + "\x00" + graph.Key(d.name)
			groups[key] = append(groups[key], d)
		}
	}
	for _, group := range groups {
		if len(group) < 2 {
			continue
		}
		for _, d := range group {
			d.skip, d.action, d.entityID = TopicNameTaken, "", ""
		}
	}
}

func (b *topicBuild) want() {
	for _, d := range b.order {
		d.wanted = false
	}
	for _, d := range b.order {
		if d.page != nil && d.placing() && b.m.snapshot.Types.NeedsEntity(*d.page) {
			b.mark(d)
		}
	}
}

func (b *topicBuild) mark(d *topicDraft) {
	for at := d; at != nil && !at.wanted; {
		at.wanted = true
		if !at.placing() || at.parent == "" {
			return
		}
		at = b.drafts[at.parent]
	}
}

func (b *topicBuild) disagree() bool {
	changed := false
	for _, d := range b.order {
		if d.page == nil || !d.wanted || !d.placing() {
			continue
		}
		if b.graphAncestor(d) != b.treeAncestor(d.page.ID) {
			d.sticky = TopicDisagrees
			changed = true
		}
	}
	return changed
}

func (b *topicBuild) graphAncestor(d *topicDraft) string {
	seen := map[string]struct{}{}
	for at := d.parent; at != ""; {
		if _, looped := seen[at]; looped {
			return ""
		}
		seen[at] = struct{}{}
		parent := b.drafts[at]
		if parent == nil {
			return ""
		}
		if parent.page != nil {
			return parent.page.ID
		}
		at = parent.parent
	}
	return ""
}

func (b *topicBuild) carries(pageID string) bool {
	if page, found := b.m.snapshot.Index.ByID(pageID); found && page.EntityID != nil {
		return true
	}
	d := b.drafts[pageSubject+pageID]
	return d != nil && d.wanted && d.placing()
}

func (b *topicBuild) treeAncestor(pageID string) string {
	start := pageID
	if hub, folded := b.m.folded[start]; folded {
		start = hub
	}
	seen := map[string]struct{}{start: {}}
	for at := b.m.parent[start]; at != ""; at = b.m.parent[at] {
		if _, looped := seen[at]; looped {
			return ""
		}
		seen[at] = struct{}{}
		ancestor, found := b.m.snapshot.Index.ByID(at)
		if !found || ancestor.Path == pagemap.RootPath {
			return ""
		}
		if b.carries(at) {
			return at
		}
	}
	return ""
}

func (b *topicBuild) plan() TopicPlan {
	out := TopicPlan{Writes: make([]TopicNode, 0), Unplaced: make([]TopicMiss, 0)}
	for _, d := range b.order {
		if !d.wanted || d.skip != "" || d.action == "" {
			continue
		}
		node := TopicNode{
			Subject: d.subject, Action: d.action, Name: d.name, Parent: d.parent, EntityID: d.entityID,
			Order: fmt.Sprintf("%06d\x00%s\x00%s", d.depth, d.sortKey(), d.subject),
		}
		if parent := b.drafts[d.parent]; parent != nil {
			node.ParentEntityID, node.ParentName = parent.entityID, parent.name
		}
		if d.page != nil {
			node.PageID = d.page.ID
			if d.action == TopicCreate {
				node.Facts, node.Keywords = b.m.snapshot.storeFacts(*d.page), d.page.Keywords
			}
		}
		node.EdgeMissing = d.action == TopicAdopt && node.ParentEntityID != "" && !b.hasParent(d.entityID, node.ParentEntityID)
		out.Writes = append(out.Writes, node)
	}
	pages := b.m.snapshot.Index.Pages()
	for i := range pages {
		page := &pages[i]
		if !live(*page) || !b.m.snapshot.Types.NeedsEntity(*page) {
			continue
		}
		d := b.drafts[pageSubject+page.ID]
		switch {
		case d == nil:
			out.Unplaced = append(out.Unplaced, TopicMiss{PageID: page.ID, Path: page.Path, Skip: TopicNoBreadcrumbs})
		case d.skip != "":
			out.Unplaced = append(out.Unplaced, TopicMiss{PageID: page.ID, Path: page.Path, Skip: d.skip})
		}
	}
	return out
}

func (b *topicBuild) hasParent(entityID, parentID string) bool {
	return slices.ContainsFunc(b.m.snapshot.Graph.Parents(entityID, 1), func(parent graph.Entity) bool { return parent.ID == parentID })
}
