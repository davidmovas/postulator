package content

import (
	"cmp"
	"slices"
	"strings"

	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/template"
)

type Relation string

const (
	RelationUp      Relation = "up"
	RelationDown    Relation = "down"
	RelationSibling Relation = "sibling"
)

type BlockedReason string

const (
	BlockedNoCanonicalPage BlockedReason = "no_canonical_page"
	BlockedNoPage          BlockedReason = "no_page"
)

type LinkTarget struct {
	EntityID string   `json:"entityId"`
	PageID   string   `json:"pageId"`
	URL      string   `json:"url"`
	Anchors  []string `json:"anchors"`
	Relation Relation `json:"relation"`
	Required bool     `json:"required"`
	Weight   float64  `json:"weight"`
	Depth    int      `json:"depth"`
}

type BlockedTarget struct {
	EntityID string
	Relation Relation
	Required bool
	Weight   float64
	Depth    int
	Reason   BlockedReason
}

type LinkContext struct {
	PageID   string       `json:"pageId"`
	PageURL  string       `json:"pageUrl"`
	EntityID string       `json:"entityId"`
	Site     pagemap.Site `json:"site"`
	Targets  []LinkTarget `json:"targets"`
}

type LinkPlan struct {
	Context LinkContext
	Blocked []BlockedTarget
}

func (c LinkContext) Phrases() []string {
	out := make([]string, 0, len(c.Targets))
	for _, target := range c.Targets {
		if len(target.Anchors) > 0 {
			out = append(out, target.Anchors[0])
		}
	}
	return out
}

func (c LinkContext) ByPageID(pageID string) (LinkTarget, bool) {
	for _, target := range c.Targets {
		if target.PageID == pageID {
			return target, true
		}
	}
	return LinkTarget{}, false
}

type Subject struct {
	Site     pagemap.Site
	PageID   string
	PagePath string
	EntityID string
}

func PlanLinks(g graph.Graph, index pagemap.Index, subject Subject, policy template.LinkPolicy) LinkPlan {
	plan := LinkPlan{
		Context: LinkContext{
			PageID: subject.PageID, PageURL: subject.PagePath, EntityID: subject.EntityID,
			Site: subject.Site, Targets: make([]LinkTarget, 0),
		},
		Blocked: make([]BlockedTarget, 0),
	}

	self, known := g.Entity(subject.EntityID)
	if !known {
		return plan
	}

	walk := planner{
		g: g, index: index, seen: map[string]struct{}{subject.EntityID: {}},
		targets: make([]LinkTarget, 0), blocked: make([]BlockedTarget, 0),
	}

	page, mapped := canonical(index, self)
	switch {
	case subject.PageID == "" && mapped:
		plan.Context.PageID = page.ID
		plan.Context.PageURL = page.Path
	case subject.PageID != "" && mapped && page.ID != subject.PageID:
		walk.targets = append(walk.targets, LinkTarget{
			EntityID: self.ID, PageID: page.ID, URL: page.Path, Anchors: anchorsOf(self, g.Label(self.ID)),
			Relation: RelationUp, Weight: 1,
		})
	}

	walk.up(subject.EntityID, policy.Rules.UpDepth)
	if policy.Rules.DownLinks {
		walk.down(subject.EntityID)
	}
	walk.siblings(subject.EntityID, policy.Rules.SiblingMinWeight)

	plan.Context.Targets = walk.targets
	plan.Blocked = walk.blocked
	return plan
}

type planner struct {
	g       graph.Graph
	index   pagemap.Index
	seen    map[string]struct{}
	targets []LinkTarget
	blocked []BlockedTarget
}

func (p *planner) place(entity graph.Entity, relation Relation, required bool, weight float64, depth int) (LinkTarget, bool) {
	if _, dup := p.seen[entity.ID]; dup {
		return LinkTarget{}, false
	}
	p.seen[entity.ID] = struct{}{}

	page, ok := canonical(p.index, entity)
	if !ok {
		p.blocked = append(p.blocked, BlockedTarget{
			EntityID: entity.ID, Relation: relation, Required: required, Weight: weight, Depth: depth,
			Reason: BlockedNoCanonicalPage,
		})
		return LinkTarget{}, false
	}
	return LinkTarget{
		EntityID: entity.ID, PageID: page.ID, URL: page.Path, Anchors: anchorsOf(entity, p.g.Label(entity.ID)),
		Relation: relation, Required: required, Weight: weight, Depth: depth,
	}, true
}

func pageless(index pagemap.Index, entity graph.Entity) bool {
	return entity.CanonicalPageID == nil && len(index.ByEntity(entity.ID)) == 0
}

func (p *planner) passOver(entity graph.Entity, relation Relation, weight float64, depth int) {
	p.seen[entity.ID] = struct{}{}
	p.blocked = append(p.blocked, BlockedTarget{
		EntityID: entity.ID, Relation: relation, Weight: weight, Depth: depth, Reason: BlockedNoPage,
	})
}

func (p *planner) up(entityID string, upDepth int) {
	frontier := []string{entityID}
	for depth := 1; depth <= upDepth && len(frontier) > 0; depth++ {
		next := make([]string, 0)
		for _, from := range frontier {
			p.upFrom(from, depth, &next)
		}
		frontier = next
	}
}

func (p *planner) upFrom(from string, depth int, next *[]string) {
	parents := p.g.Parents(from, 1)
	for i := range parents {
		if _, dup := p.seen[parents[i].ID]; dup {
			continue
		}
		if pageless(p.index, parents[i]) {
			p.passOver(parents[i], RelationUp, 1, depth)
			p.upFrom(parents[i].ID, depth, next)
			continue
		}
		if target, ok := p.place(parents[i], RelationUp, true, 1, depth); ok {
			p.targets = append(p.targets, target)
		}
		*next = append(*next, parents[i].ID)
	}
}

func (p *planner) down(entityID string) {
	out := make([]LinkTarget, 0)
	p.downFrom(entityID, &out)

	slices.SortStableFunc(out, func(a, b LinkTarget) int {
		if a.Weight != b.Weight {
			return cmp.Compare(b.Weight, a.Weight)
		}
		return strings.Compare(a.URL, b.URL)
	})
	p.targets = append(p.targets, out...)
}

func (p *planner) downFrom(from string, out *[]LinkTarget) {
	children := p.g.Children(from)
	for i := range children {
		if _, dup := p.seen[children[i].ID]; dup {
			continue
		}
		if pageless(p.index, children[i]) {
			p.passOver(children[i], RelationDown, children[i].Score, 1)
			p.downFrom(children[i].ID, out)
			continue
		}
		if target, ok := p.place(children[i], RelationDown, false, children[i].Score, 1); ok {
			*out = append(*out, target)
		}
	}
}

func (p *planner) siblings(entityID string, minWeight float64) {
	related := p.g.Related(entityID, minWeight)
	for i := range related {
		if target, ok := p.place(related[i].Entity, RelationSibling, false, related[i].Weight, 1); ok {
			p.targets = append(p.targets, target)
		}
	}
}

func MayLinkTo(g graph.Graph, index pagemap.Index, entityID string) []graph.Entity {
	if _, known := g.Entity(entityID); !known {
		return []graph.Entity{}
	}

	seen := map[string]struct{}{entityID: {}}
	out := make([]graph.Entity, 0)
	keep := func(entity graph.Entity) bool {
		if _, dup := seen[entity.ID]; dup {
			return false
		}
		seen[entity.ID] = struct{}{}
		out = append(out, entity)
		return true
	}

	climb := []string{entityID}
	for len(climb) > 0 {
		from := climb[0]
		climb = climb[1:]
		parents := g.Parents(from, 1)
		for i := range parents {
			if keep(parents[i]) && pageless(index, parents[i]) {
				climb = append(climb, parents[i].ID)
			}
		}
	}
	queue := []string{entityID}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		children := g.Children(current)
		for i := range children {
			if keep(children[i]) {
				queue = append(queue, children[i].ID)
			}
		}
	}
	related := g.Related(entityID, 0)
	for i := range related {
		keep(related[i].Entity)
	}
	return out
}

func canonical(index pagemap.Index, entity graph.Entity) (pagemap.Page, bool) {
	if entity.CanonicalPageID == nil {
		return pagemap.Page{}, false
	}
	return index.ByID(*entity.CanonicalPageID)
}

func anchorsOf(entity graph.Entity, label string) []string {
	out := make([]string, 0, len(entity.Anchors)+1)
	for _, anchor := range entity.Anchors {
		if text := strings.TrimSpace(anchor.Text); text != "" {
			out = append(out, text)
		}
	}
	if len(out) == 0 && strings.TrimSpace(label) != "" {
		out = append(out, strings.TrimSpace(label))
	}
	return out
}
