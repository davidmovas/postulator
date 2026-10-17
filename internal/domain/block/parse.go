package block

import (
	"encoding/json"
	"regexp"
	"strings"
)

type span struct {
	start      int
	end        int
	kind       BlockKind
	name       string
	attributes json.RawMessage
	dynamic    bool
}

var shortcodePattern = regexp.MustCompile(`(?s)^\s*\[([a-zA-Z_][\w-]*)(?:\s[^\]]*)?\](?:.*?\[/([a-zA-Z_][\w-]*)\])?\s*$`)

var shortcodeTagPattern = regexp.MustCompile(`\[([a-zA-Z_][\w-]*)(?:\s[^\]]*)?/?\]`)

func Parse(raw string) Body {
	if strings.Contains(raw, "<!-- wp:") {
		return finish(Blocks, raw, parseBlocks(raw))
	}
	return finish(Classic, raw, parseClassic(raw))
}

func finish(kind Kind, raw string, spans []span) Body {
	if len(spans) == 0 {
		if raw == "" {
			return Body{Kind: kind, Blocks: []Block{}}
		}
		spans = []span{{start: 0, end: len(raw), kind: Paragraph}}
	}
	spans[0].start = 0
	for i := range spans {
		if i+1 < len(spans) {
			spans[i].end = spans[i+1].start
		} else {
			spans[i].end = len(raw)
		}
	}

	blocks := make([]Block, 0, len(spans))
	for i, s := range spans {
		block := Block{
			Index: i, Kind: s.kind, Name: s.name, Raw: raw[s.start:s.end],
			Attributes: s.attributes, Dynamic: s.dynamic,
		}
		block.Hash = Hash(block.Raw)
		if !block.Dynamic && (block.Kind == Foreign || block.Kind == WP) && block.Text() == "" {
			block.Dynamic = true
		}
		blocks = append(blocks, block)
	}
	return Body{Kind: kind, Blocks: blocks}
}

func shortcodeTags(raw string) []string {
	matches := shortcodeTagPattern.FindAllStringSubmatch(raw, -1)
	out := make([]string, 0, len(matches))
	for _, match := range matches {
		out = append(out, match[1])
	}
	return out
}
