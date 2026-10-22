package block

import "slices"

func Restored(body string, written []string, moved []Moved, uncovered []string) []string {
	ours := make(map[string]bool, len(written)+len(moved)+len(uncovered))
	for _, hash := range written {
		ours[hash] = true
	}
	for _, move := range moved {
		if slices.Contains(written, move.From) {
			ours[move.To] = true
		}
	}
	for _, hash := range uncovered {
		ours[hash] = true
	}
	out := make([]string, 0)
	for _, held := range Parse(body).Blocks {
		if ours[held.Hash] && !slices.Contains(out, held.Hash) {
			out = append(out, held.Hash)
		}
	}
	return out
}
