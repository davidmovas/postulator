package block

import (
	"slices"
	"strings"

	"github.com/davidmovas/postulator/internal/domain/content"
)

func Splice(before, after string, written []string) (raw string, ours []string) {
	old := Parse(before).Blocks
	next := Parse(after).Blocks
	prefix := 0
	for prefix < len(old) && prefix < len(next) && same(old[prefix].Raw, next[prefix].Raw) {
		prefix++
	}
	suffix := 0
	for suffix < len(old)-prefix && suffix < len(next)-prefix &&
		same(old[len(old)-1-suffix].Raw, next[len(next)-1-suffix].Raw) {
		suffix++
	}
	replaced := old[prefix : len(old)-suffix]
	inserted := next[prefix : len(next)-suffix]

	var builder strings.Builder
	ours = make([]string, 0, len(written)+len(inserted))
	for i := range old[:prefix] {
		builder.WriteString(old[i].Raw)
	}
	if prefix > 0 && prefix < len(next) && trailing(old[prefix-1].Raw) == "" {
		builder.WriteString(trailing(next[prefix-1].Raw))
	}
	for _, kept := range [][]Block{inserted, old[len(old)-suffix:]} {
		for i := range kept {
			builder.WriteString(kept[i].Raw)
		}
	}
	for _, kept := range [][]Block{old[:prefix], old[len(old)-suffix:]} {
		for i := range kept {
			if slices.Contains(written, kept[i].Hash) && !slices.Contains(ours, kept[i].Hash) {
				ours = append(ours, kept[i].Hash)
			}
		}
	}
	if allOurs(replaced, written) {
		for i := range inserted {
			if !slices.Contains(ours, inserted[i].Hash) {
				ours = append(ours, inserted[i].Hash)
			}
		}
	}
	return builder.String(), ours
}

func allOurs(blocks []Block, written []string) bool {
	for i := range blocks {
		if !slices.Contains(written, blocks[i].Hash) {
			return false
		}
	}
	return true
}

func trailing(raw string) string {
	return raw[len(strings.TrimRight(raw, " \t\r\n")):]
}

func same(a, b string) bool {
	return rendered(a) == rendered(b)
}

func rendered(raw string) string {
	doc, err := content.Parse(raw)
	if err != nil {
		return strings.TrimSpace(raw)
	}
	out, err := doc.Render()
	if err != nil {
		return strings.TrimSpace(raw)
	}
	return strings.TrimSpace(strings.ReplaceAll(out, "\r\n", "\n"))
}
