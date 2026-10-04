package imports

import (
	"strings"
	"unicode"

	"github.com/davidmovas/postulator/internal/domain/importmap"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
)

func slugOf(name string) string {
	return strings.ReplaceAll(importmap.Words(name), " ", "-")
}

func titleFrom(path string) string {
	words := strings.FieldsFunc(pagemap.Slug(path), func(r rune) bool { return r == '-' || r == '_' })
	for i, word := range words {
		runes := []rune(word)
		runes[0] = unicode.ToUpper(runes[0])
		words[i] = string(runes)
	}
	return strings.Join(words, " ")
}

func fill(current, next string) string {
	if strings.TrimSpace(current) != "" {
		return current
	}
	return strings.TrimSpace(next)
}
