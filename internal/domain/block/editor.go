package block

import (
	"encoding/json"
	"regexp"
	"strings"
)

var delimiterPattern = regexp.MustCompile(`(?s)<!--\s+(/)?wp:([a-z][a-z0-9_-]*/)?([a-z][a-z0-9_-]*)\s+(\{.*?\}\s+)?(/)?-->`)

func parseBlocks(raw string) []span {
	spans := make([]span, 0)
	depth := 0
	var open span
	cursor := 0

	freeform := func(from, to int) {
		if strings.TrimSpace(raw[from:to]) == "" {
			return
		}
		if match := shortcodePattern.FindStringSubmatch(raw[from:to]); match != nil {
			spans = append(spans, span{start: from, end: to, kind: Shortcode, name: match[1], dynamic: true})
			return
		}
		spans = append(spans, span{start: from, end: to, kind: HTML})
	}

	for _, match := range delimiterPattern.FindAllStringSubmatchIndex(raw, -1) {
		start, end := match[0], match[1]
		closing := match[2] >= 0
		selfClosing := match[10] >= 0
		name := blockName(raw, match)

		switch {
		case closing && depth > 0:
			depth--
			if depth == 0 {
				open.end = end
				spans = append(spans, open)
				cursor = end
			}
		case closing:
		case selfClosing && depth == 0:
			freeform(cursor, start)
			spans = append(spans, span{
				start: start, end: end, kind: kindOf(name), name: name,
				attributes: attributesOf(raw, match), dynamic: true,
			})
			cursor = end
		case selfClosing:
		case depth == 0:
			freeform(cursor, start)
			open = span{
				start: start, kind: kindOf(name), name: name,
				attributes: attributesOf(raw, match), dynamic: name == "core/shortcode",
			}
			depth = 1
		default:
			depth++
		}
	}
	if depth > 0 {
		open.end = len(raw)
		spans = append(spans, open)
		cursor = len(raw)
	}
	freeform(cursor, len(raw))
	return spans
}

func blockName(raw string, match []int) string {
	namespace := "core/"
	if match[4] >= 0 {
		namespace = raw[match[4]:match[5]]
	}
	return namespace + raw[match[6]:match[7]]
}

func attributesOf(raw string, match []int) json.RawMessage {
	if match[8] < 0 {
		return nil
	}
	return json.RawMessage(strings.TrimSpace(raw[match[8]:match[9]]))
}

func kindOf(name string) BlockKind {
	switch name {
	case "core/paragraph":
		return Paragraph
	case "core/heading":
		return Heading
	case "core/list":
		return List
	case "core/quote":
		return Quote
	case "core/html":
		return HTML
	case "core/shortcode":
		return Shortcode
	default:
		return WP
	}
}
