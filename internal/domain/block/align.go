package block

import (
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
