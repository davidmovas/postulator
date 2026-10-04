package graph

import "strings"

func Key(text string) string {
	folded := []byte(strings.TrimSpace(text))
	for i, b := range folded {
		if 'A' <= b && b <= 'Z' {
			folded[i] = b + 'a' - 'A'
		}
	}
	return string(folded)
}
