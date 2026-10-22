package block

import (
	"slices"
	"strings"

	"github.com/davidmovas/postulator/internal/domain/content"
)

type pair struct {
	left, right int
}

func align(left, right []string) []pair {
	lengths := make([][]int, len(left)+1)
	for i := range lengths {
		lengths[i] = make([]int, len(right)+1)
	}
	for i := len(left) - 1; i >= 0; i-- {
		for j := len(right) - 1; j >= 0; j-- {
			if left[i] == right[j] {
				lengths[i][j] = lengths[i+1][j+1] + 1
				continue
			}
			lengths[i][j] = max(lengths[i+1][j], lengths[i][j+1])
		}
	}
	out := make([]pair, 0, lengths[0][0])
	for i, j := 0, 0; i < len(left) && j < len(right); {
		switch {
		case left[i] == right[j]:
			out = append(out, pair{left: i, right: j})
			i++
			j++
		case lengths[i+1][j] >= lengths[i][j+1]:
			i++
		default:
			j++
		}
	}
	return out
}

func renders(blocks []Block) []string {
	out := make([]string, 0, len(blocks))
	for i := range blocks {
		out = append(out, rendered(blocks[i].Raw))
	}
	return out
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

func Align(composed, stored string, ours []string) []int {
	left, right := Parse(composed).Blocks, Parse(stored).Blocks
	out := make([]int, len(left))
	for i := range out {
		out[i] = -1
	}
	client := make([]int, 0, len(left))
	theirs := make([]string, 0, len(left))
	for i := range left {
		if !slices.Contains(ours, left[i].Hash) {
			client = append(client, i)
			theirs = append(theirs, rendered(left[i].Raw))
		}
	}
	for _, at := range align(theirs, renders(right)) {
		out[client[at.left]] = at.right
	}
	pairGaps(out, left, right)
	return out
}

func pairGaps(out []int, left, right []Block) {
	prevLeft, prevRight := -1, -1
	for i := 0; i <= len(out); i++ {
		if i < len(out) && out[i] < 0 {
			continue
		}
		nextRight := len(right)
		if i < len(out) {
			nextRight = out[i]
		}
		for _, at := range align(texts(left[prevLeft+1:i]), texts(right[prevRight+1:nextRight])) {
			out[prevLeft+1+at.left] = prevRight + 1 + at.right
		}
		prevLeft, prevRight = i, nextRight
	}
}

func texts(blocks []Block) []string {
	out := make([]string, 0, len(blocks))
	for i := range blocks {
		text := blocks[i].Text()
		if text == "" {
			text = "\x00" + rendered(blocks[i].Raw)
		}
		out = append(out, text)
	}
	return out
}

func Lineage(from, to string, ours []string) []Moved {
	left, right := Parse(from).Blocks, Parse(to).Blocks
	out := make([]Moved, 0)
	for i, at := range Align(from, to, ours) {
		if at >= 0 && left[i].Hash != right[at].Hash {
			out = append(out, Moved{From: left[i].Hash, To: right[at].Hash})
		}
	}
	return out
}

func SameRender(left, right string) bool {
	a, b := renders(Parse(left).Blocks), renders(Parse(right).Blocks)
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
