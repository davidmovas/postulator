package imports

import (
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/importmap"
)

var kindSynonyms = map[string]graph.Kind{
	"commercial taxonomy": graph.KindCategory,
	"taxonomy":            graph.KindCategory,
	"compound":            graph.KindProduct,
	"compound product":    graph.KindProduct,
	"product compound":    graph.KindProduct,
	"product owner":       graph.KindProduct,
	"geo":                 graph.KindCustom,
	"geographic":          graph.KindCustom,
	"geographic entity":   graph.KindCustom,
	"root":                graph.KindHub,
	"root entity":         graph.KindHub,
	"pillar":              graph.KindHub,
}

func kindOf(raw string) (graph.Kind, bool) {
	words := importmap.Words(raw)
	if words == "" {
		return "", true
	}
	if kind := graph.Kind(words); kind.Valid() {
		return kind, true
	}
	kind, known := kindSynonyms[words]
	return kind, known
}
