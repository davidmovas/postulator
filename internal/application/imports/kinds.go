package imports

import (
	"strings"
	"unicode"

	"github.com/davidmovas/postulator/internal/domain/graph"
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

func kindWords(raw string) string {
	return strings.Join(strings.FieldsFunc(strings.ToLower(raw), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	}), " ")
}

func kindOf(raw string) (graph.Kind, bool) {
	words := kindWords(raw)
	if words == "" {
		return "", true
	}
	if kind := graph.Kind(words); kind.Valid() {
		return kind, true
	}
	kind, known := kindSynonyms[words]
	return kind, known
}
