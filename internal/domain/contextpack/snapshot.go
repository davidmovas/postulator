package contextpack

import (
	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
	"github.com/davidmovas/postulator/internal/domain/render"
)

type Snapshot struct {
	Site     pagemap.Site
	Absolute bool
	Index    pagemap.Index
	Types    pagetype.Set
	Graph    graph.Graph
	Terms    map[string][]pagemap.Term
	Fields   map[string][]field.Value
	Bodies   map[string]block.Snapshot
	Written  map[string][]string
	Renders  map[string]render.Record
	Links    []pagemap.PageLink
	Findings []finding.Finding
}

func (s Snapshot) value(pageID string, key field.Key) (field.Value, bool) {
	values := s.Fields[pageID]
	for i := range values {
		if values[i].Key == key {
			return values[i], true
		}
	}
	return field.Value{}, false
}

func (s Snapshot) text(pageID string, key field.Key) string {
	value, found := s.value(pageID, key)
	if !found {
		return ""
	}
	return value.Current
}

func (s Snapshot) body(pageID string) (block.Body, bool) {
	snapshot, found := s.Bodies[pageID]
	if !found {
		return block.Body{}, false
	}
	return block.Parse(snapshot.Raw), true
}

func (s Snapshot) entity(page pagemap.Page) (graph.Entity, bool) {
	if page.EntityID == nil {
		return graph.Entity{}, false
	}
	return s.Graph.Entity(*page.EntityID)
}

func (s Snapshot) typeOf(page pagemap.Page) pagetype.Type {
	return s.Types.TypeOf(page)
}

func (s Snapshot) titleOf(page pagemap.Page) string {
	return shown(page, s.text(page.ID, field.Title), page.Title, s.text(page.ID, field.H1), page.H1)
}

func (s Snapshot) h1Of(page pagemap.Page) string {
	return shown(page, s.text(page.ID, field.H1), page.H1, s.text(page.ID, field.Title), page.Title)
}

func shown(page pagemap.Page, names ...string) string {
	for _, held := range names {
		if held != "" {
			return held
		}
	}
	return page.Path
}

func onSite(page pagemap.Page) bool {
	return page.WPID != nil
}

func live(page pagemap.Page) bool {
	return page.Status != pagemap.StatusArchived
}
