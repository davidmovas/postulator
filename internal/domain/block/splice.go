package block

import (
	"slices"
	"strings"
)

func Splice(before, after string, written []string) (raw string, ours []string) {
	old := Parse(before).Blocks
	next := Parse(after).Blocks
	matched := align(renders(old), renders(next))

	var builder strings.Builder
	ours = make([]string, 0, len(written)+len(next))
	claim := func(hash string) {
		if !slices.Contains(ours, hash) {
			ours = append(ours, hash)
		}
	}
	from, to := 0, 0
	gap := func(oldEnd, newEnd int) {
		replaced, inserted := old[from:oldEnd], next[to:newEnd]
		mine := gapOurs(replaced, inserted, written)
		for i := range inserted {
			builder.WriteString(inserted[i].Raw)
			if mine[i] {
				claim(inserted[i].Hash)
			}
		}
	}
	for _, at := range matched {
		gap(at.left, at.right)
		builder.WriteString(old[at.left].Raw)
		if trailing(old[at.left].Raw) == "" && at.right+1 < len(next) {
			builder.WriteString(trailing(next[at.right].Raw))
		}
		if slices.Contains(written, old[at.left].Hash) {
			claim(old[at.left].Hash)
		}
		from, to = at.left+1, at.right+1
	}
	gap(len(old), len(next))
	return builder.String(), ours
}

func gapOurs(replaced, inserted []Block, written []string) []bool {
	out := make([]bool, len(inserted))
	switch {
	case allOurs(replaced, written):
		for i := range out {
			out[i] = true
		}
	case len(replaced) == len(inserted):
		for i := range out {
			out[i] = slices.Contains(written, replaced[i].Hash)
		}
	}
	return out
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
