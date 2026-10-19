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

func titleOf(page pagemap.Page) string {
	for _, held := range []string{page.Observed.Title, page.Title, page.Observed.H1, page.H1} {
		if held != "" {
			return held
		}
	}
	return page.Path
}

func h1Of(page pagemap.Page) string {
	for _, held := range []string{page.Observed.H1, page.H1, page.Observed.Title, page.Title} {
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
