package block

import (
	"slices"
	"strings"
)

const (
	keptSeparator = "\n\n"
	htmlOpen      = "<!-- wp:html -->\n"
	htmlClose     = "\n<!-- /wp:html -->"
)

func Keep(current string, written []string, ours string) (raw string, kept bool) {
	body := Parse(current)
	client := clientBlocks(body, written)
	if len(client) == 0 {
		return ours, false
	}
	if body.Kind == Blocks {
		ours = htmlOpen + ours + htmlClose
	}
	at := insertionPoint(client)
	parts := make([]string, 0, 3)
	if before := strings.TrimRight(joinRaw(client[:at]), " \t\r\n"); before != "" {
		parts = append(parts, before)
	}
	parts = append(parts, ours)
	if after := strings.TrimLeft(joinRaw(client[at:]), " \t\r\n"); after != "" {
		parts = append(parts, after)
	}
	return strings.Join(parts, keptSeparator), true
}

func Client(current string, written []string) string {
	return joinRaw(clientBlocks(Parse(current), written))
}

func HasClient(current string, written []string) bool {
	return len(clientBlocks(Parse(current), written)) > 0
}

func Ours(stored, client string) []string {
	theirs := make(map[string]struct{})
	for _, held := range Parse(client).Blocks {
		theirs[held.Hash] = struct{}{}
	}
	out := make([]string, 0)
	for _, held := range Parse(stored).Blocks {
		if _, kept := theirs[held.Hash]; !kept && !slices.Contains(out, held.Hash) {
			out = append(out, held.Hash)
		}
	}
	return out
}

func clientBlocks(body Body, written []string) []Block {
	out := make([]Block, 0, len(body.Blocks))
	for i := range body.Blocks {
		if !slices.Contains(written, body.Blocks[i].Hash) && strings.TrimSpace(body.Blocks[i].Raw) != "" {
			out = append(out, body.Blocks[i])
		}
	}
	return out
}

func insertionPoint(client []Block) int {
	for i := len(client) - 1; i >= 0; i-- {
		if client[i].Text() != "" {
			return i + 1
		}
	}
	for i := range client {
		if client[i].Dynamic || client[i].Kind == Shortcode {
			return i
		}
	}
	return len(client)
}

func joinRaw(blocks []Block) string {
	var builder strings.Builder
	for i := range blocks {
		builder.WriteString(blocks[i].Raw)
	}
	return builder.String()
}
