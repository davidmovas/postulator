package imports

import (
	"cmp"
	"maps"
	"slices"

	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/importmap"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

func sheetAnchors(texts []string) []graph.Anchor {
	out := make([]graph.Anchor, 0, len(texts))
	for _, text := range texts {
		out = append(out, graph.Anchor{Text: text, Source: graph.AnchorUser, Weight: 1})
	}
	return out
}

func anchorTexts(anchors []graph.Anchor) []string {
	out := make([]string, 0, len(anchors))
	for i := range anchors {
		out = append(out, anchors[i].Text)
	}
	return out
}

func edgeKey(e graph.Edge) string {
	return string(e.Kind) + "|" + e.FromEntityID + "|" + e.ToEntityID
}

func sameEntity(a, b graph.Entity) bool {
	return a.Kind == b.Kind && a.Keywords.Equal(b.Keywords) &&
		slices.Equal(anchorTexts(a.Anchors), anchorTexts(b.Anchors))
}

func (b *builder) planEntities() error {
	b.resolved = make(map[string]graph.Entity, len(b.order))
	for _, at := range b.order {
		u := &b.units[at]
		kind := b.kindOf(u)
		step := b.updateEntity
		if u.matched == "" {
			step = b.createEntity
		}
		if err := step(u, kind); err != nil {
			return err
		}
	}
	return nil
}

func (b *builder) kindOf(u *unit) graph.Kind {
	kind, known := kindOf(u.kind)
	if !known {
		b.p.noteAt(u.at, string(importmap.FieldEntityKind), CodeUnknownEntityKind,
			"the entity kind is not recognized and was read as a topic: "+u.kind)
	}
	return kind
}

func (b *builder) createEntity(u *unit, kind graph.Kind) error {
	var scope *string
	if parentID := b.parentID(u); parentID != "" && parentID != u.id {
		scope = &parentID
	}
	entity, err := graph.NewEntity(graph.Entity{
		ID: u.id, SiteID: b.state.siteID, Name: u.name, Kind: cmp.Or(kind, u.defaultKind()),
		ScopeID: scope, Keywords: u.keywords, Anchors: sheetAnchors(u.anchors),
		Source: graph.SourceImport, CreatedAt: b.now, UpdatedAt: b.now,
	})
	if err != nil {
		return err
	}
	b.resolved[entity.ID] = entity
	b.p.entities = append(b.p.entities, plannedEntity{entity: entity, created: true})
	b.reportEntity(u, entity, ActionCreate)
	return nil
}

func (b *builder) updateEntity(u *unit, kind graph.Kind) error {
	current := b.byID[u.matched]
	next := current
	next.Kind = cmp.Or(kind, current.Kind)
	next.Keywords = current.Keywords.Merge(u.keywords)
	next.Anchors = sheetAnchors(union(anchorTexts(current.Anchors), u.anchors))
	next.UpdatedAt = b.now
	entity, err := graph.NewEntity(next)
	if err != nil {
		return err
	}
	b.resolved[entity.ID] = entity
	if sameEntity(entity, current) {
		b.reportEntity(u, entity, ActionSkip)
		return nil
	}
	b.p.entities = append(b.p.entities, plannedEntity{entity: entity, created: false})
	b.reportEntity(u, entity, ActionUpdate)
	return nil
}

func (b *builder) reportEntity(u *unit, entity graph.Entity, action Action) {
	b.p.report.Entities = append(b.p.report.Entities, entityView(b.p.sheetAt(u.at), entity, b.parentName(u), action))
}

func (b *builder) planEdges() error {
	seen := maps.Clone(b.state.edgeKeys)
	for _, at := range b.order {
		u := &b.units[at]
		if err := b.planParentEdge(u, seen); err != nil {
			return err
		}
		if err := b.planRelatedEdges(u, seen); err != nil {
			return err
		}
	}
	return nil
}

func (b *builder) planParentEdge(u *unit, seen map[string]struct{}) error {
	parentID := b.parentID(u)
	switch {
	case parentID == "":
	case parentID == u.id:
		b.p.noteAt(u.parent.at, string(importmap.FieldParentEntity), CodeSelfEdge, "the entity names itself as its parent: "+u.name)
	case u.parent.weak && u.matched != "" && b.parents[u.matched] > 0:
	default:
		return b.addEdge(u, parentID, graph.EdgeParent, seen)
	}
	return nil
}

func (b *builder) planRelatedEdges(u *unit, seen map[string]struct{}) error {
	for _, name := range u.related {
		relatedID, found := b.relatedID(u, name)
		switch {
		case !found:
		case relatedID == u.id:
			b.p.noteAt(u.at, string(importmap.FieldRelated), CodeSelfEdge, "the entity names itself as related: "+u.name)
		default:
			if err := b.addEdge(u, relatedID, graph.EdgeRelated, seen); err != nil {
				return err
			}
		}
	}
	return nil
}

func (b *builder) relatedID(u *unit, name string) (string, bool) {
	ref, code := b.resolve(name, u.context)
	switch code {
	case CodeUnknownParent:
		b.p.noteAt(u.at, string(importmap.FieldRelated), CodeUnknownRelated,
			"the related entity is not in the file and not on the site: "+name)
		return "", false
	case CodeAmbiguousParent:
		b.p.noteAt(u.at, string(importmap.FieldRelated), CodeAmbiguousEntity,
			"more than one entity is named "+name+"; name a related entity that only one carries")
		return "", false
	}
	if ref.kind == refUnit {
		return b.entityID(ref.unit), true
	}
	return ref.site, true
}

func (b *builder) addEdge(from *unit, toID string, kind graph.EdgeKind, seen map[string]struct{}) error {
	edge, err := graph.NewEdge(graph.Edge{
		ID: id.New(), SiteID: b.state.siteID, FromEntityID: from.id, ToEntityID: toID,
		Kind: kind, Weight: 1, Source: graph.SourceImport, Status: graph.StatusApproved, CreatedAt: b.now,
	})
	if err != nil {
		return err
	}
	view := PreviewEdge{
		Sheet: b.p.sheetAt(from.at), From: b.nameOf(from.id), To: b.nameOf(toID), Kind: string(kind), Action: string(ActionCreate),
	}
	if _, known := seen[edgeKey(edge)]; known {
		view.Action = string(ActionSkip)
	} else {
		seen[edgeKey(edge)] = struct{}{}
		b.p.edges = append(b.p.edges, edge)
	}
	b.p.report.Edges = append(b.p.report.Edges, view)
	return nil
}

func (b *builder) checkAcyclic() error {
	entities := make([]graph.Entity, 0, len(b.state.entities)+len(b.resolved))
	for _, at := range slices.Sorted(maps.Keys(b.resolved)) {
		entities = append(entities, b.resolved[at])
	}
	for i := range b.state.entities {
		if _, replaced := b.resolved[b.state.entities[i].ID]; !replaced {
			entities = append(entities, b.state.entities[i])
		}
	}

	g, err := graph.New(entities, slices.Concat(b.state.edges, b.p.edges))
	if err != nil {
		return err
	}
	if cycleErr := g.ValidateAcyclic(); cycleErr != nil {
		b.p.noteAt(b.p.whole(), string(importmap.FieldParentEntity), CodeCycle, cycleErr.Error())
	}
	return nil
}
