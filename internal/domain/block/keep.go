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

func Part(current string, written []string, ours string) string {
	body := Parse(current)
	if len(clientBlocks(body, written)) > 0 && body.Kind == Blocks {
		return HTMLBlock(ours)
	}
	return ours
}

func HTMLBlock(raw string) string {
	return htmlOpen + raw + htmlClose
}

func Keep(current string, written []string, ours string) (raw string, kept bool) {
	body := Parse(current)
	client := clientBlocks(body, written)
	if len(client) == 0 {
		return ours, false
	}
	ours = Part(current, written, ours)
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

func Rewrite(current string, written []string, ours string) (raw, theirs, part string, err error) {
	body := Parse(current)
	part = Part(current, written, ours)
	rewritable := func(held Block) bool { return slices.Contains(written, held.Hash) || held.Prose() }
	first := slices.IndexFunc(body.Blocks, rewritable)
	kept := make([]Block, 0, len(body.Blocks))
	for i := range body.Blocks {
		if !rewritable(body.Blocks[i]) && strings.TrimSpace(body.Blocks[i].Raw) != "" {
			kept = append(kept, body.Blocks[i])
		}
	}
	if first < 0 {
		raw, _ = Keep(current, written, ours)
		return raw, joinRaw(kept), part, nil
	}
	original := body.Blocks
	for i := range slices.Backward(original) {
		if i <= first || !rewritable(original[i]) {
			continue
		}
		if body, err = body.Delete(i); err != nil {
			return "", "", "", err
		}
	}
	if body, err = body.Replace(first, part); err != nil {
		return "", "", "", err
	}
	return body.Serialize(), joinRaw(kept), part, nil
}

func Client(current string, written []string) string {
	return joinRaw(clientBlocks(Parse(current), written))
}

func HasClient(current string, written []string) bool {
	return len(clientBlocks(Parse(current), written)) > 0
}

func Ours(stored, client, part string) []string {
	blocks := Parse(stored).Blocks
	out := make([]string, 0, len(blocks))
	if strings.TrimSpace(client) == "" {
		for i := range blocks {
			if !slices.Contains(out, blocks[i].Hash) {
				out = append(out, blocks[i].Hash)
			}
		}
		return out
	}
	theirs := make(map[int]struct{})
	for _, at := range align(renders(blocks), renders(Parse(client).Blocks)) {
		theirs[at.left] = struct{}{}
	}
	mine := renders(Parse(part).Blocks)
	storedRenders := renders(blocks)
	for i := range blocks {
		if _, held := theirs[i]; held {
			continue
		}
		at := slices.Index(mine, storedRenders[i])
		if at < 0 {
			continue
		}
		mine = slices.Delete(mine, at, at+1)
		if !slices.Contains(out, blocks[i].Hash) {
			out = append(out, blocks[i].Hash)
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
