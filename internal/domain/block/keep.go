package block

import (
	"slices"
	"strings"
)

const keptSeparator = "\n\n"

func Keep(current string, written []string, ours string) (raw string, kept bool) {
	client := clientBlocks(Parse(current), written)
	if len(client) == 0 {
		return ours, false
	}
	at := insertionPoint(client)
	parts := make([]string, 0, 3)
	if before := strings.TrimRight(joinRaw(client[:at]), " \t\r\n"); before != "" {
		parts = append(parts, before)
	}
	parts = append(parts, ours)
	if after := joinRaw(client[at:]); after != "" {
		parts = append(parts, after)
	}
	return strings.Join(parts, keptSeparator), true
}

func Client(current string, written []string) string {
	return joinRaw(clientBlocks(Parse(current), written))
}

func ClientText(current string, written []string) bool {
	for _, held := range clientBlocks(Parse(current), written) {
		if held.Text() != "" {
			return true
		}
	}
	return false
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
