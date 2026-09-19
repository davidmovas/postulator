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

const BlockedNoCanonicalPage BlockedReason = "no_canonical_page"

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

func (c LinkContext) ByURL(url string) (LinkTarget, bool) {
	for _, target := range c.Targets {
		if target.URL == url {
			return target, true
		}
	}
	return LinkTarget{}, false
}

func (c LinkContext) ByPageID(pageID string) (LinkTarget, bool) {
	for _, target := range c.Targets {
		if target.PageID == pageID {
			return target, true
		}
	}
	return LinkTarget{}, false
}

func BuildLinkContext(g graph.Graph, index pagemap.Index, entityID string, policy template.LinkPolicy) LinkContext {
	return PlanLinks(g, index, entityID, policy).Context
}

func PlanLinks(g graph.Graph, index pagemap.Index, entityID string, policy template.LinkPolicy) LinkPlan {
	plan := LinkPlan{
		Context: LinkContext{EntityID: entityID, Targets: make([]LinkTarget, 0)},
		Blocked: make([]BlockedTarget, 0),
	}

	self, known := g.Entity(entityID)
	if !known {
		return plan
	}
	if page, ok := canonical(index, self); ok {
		plan.Context.PageID = page.ID
		plan.Context.PageURL = page.Path
	}

	walk := planner{
		g: g, index: index, seen: map[string]struct{}{entityID: {}},
		targets: make([]LinkTarget, 0), blocked: make([]BlockedTarget, 0),
	}
	walk.up(entityID, policy.Rules.UpDepth)
	if policy.Rules.DownLinks {
		walk.down(entityID)
	}
	walk.siblings(entityID, policy.Rules.SiblingMinWeight)

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
		EntityID: entity.ID, PageID: page.ID, URL: page.Path, Anchors: anchorsOf(entity),
		Relation: relation, Required: required, Weight: weight, Depth: depth,
	}, true
}

func (p *planner) up(entityID string, upDepth int) {
	previous := 0
	for depth := 1; depth <= upDepth; depth++ {
		level := p.g.Parents(entityID, depth)
		for i := previous; i < len(level); i++ {
			if target, ok := p.place(level[i], RelationUp, true, 1, depth); ok {
				p.targets = append(p.targets, target)
			}
		}
		previous = len(level)
	}
}

func (p *planner) down(entityID string) {
	children := p.g.Children(entityID)

	out := make([]LinkTarget, 0, len(children))
	for i := range children {
		if target, ok := p.place(children[i], RelationDown, false, children[i].Score, 1); ok {
			out = append(out, target)
		}
	}

	slices.SortStableFunc(out, func(a, b LinkTarget) int {
		if a.Weight != b.Weight {
			return cmp.Compare(b.Weight, a.Weight)
		}
		return strings.Compare(a.URL, b.URL)
	})
	p.targets = append(p.targets, out...)
}

func (p *planner) siblings(entityID string, minWeight float64) {
	related := p.g.Related(entityID, minWeight)
	for i := range related {
		if target, ok := p.place(related[i].Entity, RelationSibling, false, related[i].Weight, 1); ok {
			p.targets = append(p.targets, target)
		}
	}
}

func canonical(index pagemap.Index, entity graph.Entity) (pagemap.Page, bool) {
	if entity.CanonicalPageID == nil {
		return pagemap.Page{}, false
	}
	return index.ByID(*entity.CanonicalPageID)
}

func anchorsOf(entity graph.Entity) []string {
	out := make([]string, 0, len(entity.Anchors)+1)
	for _, anchor := range entity.Anchors {
		if text := strings.TrimSpace(anchor.Text); text != "" {
			out = append(out, text)
		}
	}
	if len(out) == 0 && strings.TrimSpace(entity.Name) != "" {
		out = append(out, strings.TrimSpace(entity.Name))
	}
	return out
}
