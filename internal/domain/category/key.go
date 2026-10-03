package category

import (
	"html"
	"strings"
	"unicode"
)

func Key(name string) string {
	return strings.Map(fold, strings.Join(strings.Fields(html.UnescapeString(name)), " "))
}

func fold(r rune) rune {
	least := r
	for at := unicode.SimpleFold(r); at != r; at = unicode.SimpleFold(at) {
		least = min(least, at)
	}
	lower := unicode.ToLower(least)
	for at := unicode.SimpleFold(least); at != least; at = unicode.SimpleFold(at) {
		if at == lower {
			return lower
		}
	}
	return least
}
