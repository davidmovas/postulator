package imports

import (
	"strings"
	"unicode"

	"github.com/davidmovas/postulator/internal/domain/importmap"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
)

func key(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

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

func union(into, more []string) []string {
	seen := make(map[string]struct{}, len(into)+len(more))
	out := make([]string, 0, len(into)+len(more))
	for _, values := range [][]string{into, more} {
		for _, value := range values {
			trimmed := strings.TrimSpace(value)
			if trimmed == "" {
				continue
			}
			if _, dup := seen[key(trimmed)]; dup {
				continue
			}
			seen[key(trimmed)] = struct{}{}
			out = append(out, trimmed)
		}
	}
	return out
}

func fill(current, next string) string {
	if strings.TrimSpace(current) != "" {
		return current
	}
	return strings.TrimSpace(next)
}
