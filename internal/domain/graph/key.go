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

func Distinct(lists ...[]string) []string {
	out := make([]string, 0)
	seen := make(map[string]struct{})
	for _, list := range lists {
		for _, text := range list {
			trimmed := strings.TrimSpace(text)
			if _, dup := seen[Key(trimmed)]; dup || trimmed == "" {
				continue
			}
			seen[Key(trimmed)] = struct{}{}
			out = append(out, trimmed)
		}
	}
	return out
}
