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

type LinkContext struct {
	PageID   string       `json:"pageId"`
	PageURL  string       `json:"pageUrl"`
	EntityID string       `json:"entityId"`
	Targets  []LinkTarget `json:"targets"`
}

func (c LinkContext) Required() []LinkTarget {
	out := make([]LinkTarget, 0, len(c.Targets))
	for _, target := range c.Targets {
		if target.Required {
			out = append(out, target)
		}
	}
	return out
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

func BuildLinkContext(g graph.Graph, index pagemap.Index, entityID string, policy template.LinkPolicy) LinkContext {
	context := LinkContext{EntityID: entityID, Targets: make([]LinkTarget, 0)}

	self, known := g.Entity(entityID)
	if !known {
		return context
	}
	if page, ok := canonical(index, self); ok {
		context.PageID = page.ID
		context.PageURL = page.Path
	}

	seen := map[string]struct{}{entityID: {}}
	context.Targets = append(context.Targets, upTargets(g, index, entityID, policy, seen)...)
	if policy.Rules.DownLinks {
		context.Targets = append(context.Targets, downTargets(g, index, entityID, seen)...)
	}
	context.Targets = append(context.Targets, siblingTargets(g, index, entityID, policy, seen)...)
	return context
}

func upTargets(g graph.Graph, index pagemap.Index, entityID string, policy template.LinkPolicy, seen map[string]struct{}) []LinkTarget {
	out := make([]LinkTarget, 0)

	previous := 0
	for depth := 1; depth <= policy.Rules.UpDepth; depth++ {
		level := g.Parents(entityID, depth)
		for i := previous; i < len(level); i++ {
			entity := &level[i]
			if _, dup := seen[entity.ID]; dup {
				continue
			}
			page, ok := canonical(index, *entity)
			if !ok {
				continue
			}
			seen[entity.ID] = struct{}{}
			out = append(out, LinkTarget{
				EntityID: entity.ID, PageID: page.ID, URL: page.Path, Anchors: anchorsOf(*entity),
				Relation: RelationUp, Required: true, Weight: 1, Depth: depth,
			})
		}
		previous = len(level)
	}
	return out
}

func downTargets(g graph.Graph, index pagemap.Index, entityID string, seen map[string]struct{}) []LinkTarget {
	children := g.Children(entityID)

	out := make([]LinkTarget, 0, len(children))
	for i := range children {
		entity := &children[i]
		if _, dup := seen[entity.ID]; dup {
			continue
		}
		page, ok := canonical(index, *entity)
		if !ok {
			continue
		}
		seen[entity.ID] = struct{}{}
		out = append(out, LinkTarget{
			EntityID: entity.ID, PageID: page.ID, URL: page.Path, Anchors: anchorsOf(*entity),
			Relation: RelationDown, Weight: entity.Score, Depth: 1,
		})
	}

	slices.SortStableFunc(out, func(a, b LinkTarget) int {
		if a.Weight != b.Weight {
			return cmp.Compare(b.Weight, a.Weight)
		}
		return strings.Compare(a.URL, b.URL)
	})
	return out
}

func siblingTargets(g graph.Graph, index pagemap.Index, entityID string, policy template.LinkPolicy, seen map[string]struct{}) []LinkTarget {
	related := g.Related(entityID, policy.Rules.SiblingMinWeight)

	out := make([]LinkTarget, 0, len(related))
	for i := range related {
		neighbor := &related[i]
		if _, dup := seen[neighbor.Entity.ID]; dup {
			continue
		}
		page, ok := canonical(index, neighbor.Entity)
		if !ok {
			continue
		}
		seen[neighbor.Entity.ID] = struct{}{}
		out = append(out, LinkTarget{
			EntityID: neighbor.Entity.ID, PageID: page.ID, URL: page.Path, Anchors: anchorsOf(neighbor.Entity),
			Relation: RelationSibling, Weight: neighbor.Weight, Depth: 1,
		})
	}
	return out
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
