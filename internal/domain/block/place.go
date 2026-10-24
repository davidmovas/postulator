package block

import (
	"encoding/json"
	"slices"
	"strings"
	"unicode/utf8"

	"golang.org/x/net/html"

	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const ReasonAnchorNotFound = "anchor_not_found"

const entityLimit = 40

var proseBlocks = []string{"core/paragraph", "core/heading", "core/list", "core/list-item", "core/quote"}

var proseContainers = []string{"core/group", "core/columns", "core/column"}

var proseElements = []string{
	"p", "h1", "h2", "h3", "h4", "h5", "h6", "ul", "ol", "li", "blockquote", "cite",
	"a", "strong", "em", "b", "i", "u", "s", "br", "span", "sub", "sup", "mark", "small", "code", "abbr", "q", "del", "ins",
}

type Link struct {
	Anchor string
	Href   string
	Nth    int
}

type byteSpan struct {
	start, end int
}

func (b Body) PlaceLink(i int, link Link) (Body, error) {
	if err := b.within(i); err != nil {
		return Body{}, err
	}
	held := b.Blocks[i]
	if held.Dynamic || held.Kind == Shortcode {
		return Body{}, notEditable(i, "the block is the site's to render, so no link is written into it")
	}
	if attributesHold(held.Raw, link.Anchor) {
		return Body{}, notEditable(i, "the block's attributes hold its text, so its markup cannot change without them")
	}
	found := anchorsIn(held.Raw, link.Anchor)
	if link.Nth < 0 || link.Nth >= len(found) {
		return Body{}, errors.New(errors.Invalid, "the anchor does not occur in the block's text").
			WithDetail("reason", ReasonAnchorNotFound).WithDetail("block", i).WithDetail("anchor", link.Anchor)
	}
	at := found[link.Nth]
	raw := held.Raw[:at.start] + `<a href="` + html.EscapeString(link.Href) + `">` +
		held.Raw[at.start:at.end] + "</a>" + held.Raw[at.end:]
	pieces := b.pieces()
	pieces[i] = split(raw)
	pieces[i].fresh = true
	return b.rebuild(pieces, i, i-1)
}

func notEditable(i int, message string) error {
	return errors.New(errors.Invalid, message).WithDetail("reason", ReasonBlockNotEditable).WithDetail("block", i)
}

func (b Block) Prose() bool {
	if b.Dynamic || b.Kind == Shortcode || b.Kind == Foreign || b.Text() == "" || shortcodeTagPattern.MatchString(b.Raw) {
		return false
	}
	container := slices.Contains(proseContainers, b.Name)
	for _, match := range delimiterPattern.FindAllStringSubmatchIndex(b.Raw, -1) {
		name := blockName(b.Raw, match)
		if !slices.Contains(proseBlocks, name) && !slices.Contains(proseContainers, name) {
			return false
		}
	}
	tokenizer := html.NewTokenizer(strings.NewReader(b.Raw))
	for {
		switch tokenizer.Next() {
		case html.ErrorToken:
			return true
		case html.StartTagToken, html.SelfClosingTagToken:
			name, _ := tokenizer.TagName()
			if tag := string(name); !slices.Contains(proseElements, tag) && (!container || tag != "div") {
				return false
			}
		default:
		}
	}
}

func attributesHold(raw, anchor string) bool {
	for _, match := range delimiterPattern.FindAllStringSubmatchIndex(raw, -1) {
		attributes := attributesOf(raw, match)
		if attributes == nil {
			continue
		}
		var decoded any
		if json.Unmarshal(attributes, &decoded) != nil || holds(decoded, anchor) {
			return true
		}
	}
	return false
}

func holds(value any, anchor string) bool {
	switch typed := value.(type) {
	case string:
		_, _, found := content.FindFold(typed, anchor)
		return found
	case []any:
		return slices.ContainsFunc(typed, func(item any) bool { return holds(item, anchor) })
	case map[string]any:
		for _, item := range typed {
			if holds(item, anchor) {
				return true
			}
		}
	}
	return false
}

func anchorsIn(raw, anchor string) []byteSpan {
	out := make([]byteSpan, 0, 1)
	tokenizer := html.NewTokenizer(strings.NewReader(raw))
	open := make([]string, 0, 8)
	position := 0
	for {
		kind := tokenizer.Next()
		if kind == html.ErrorToken {
			return out
		}
		size := len(tokenizer.Raw())
		switch kind {
		case html.StartTagToken:
			name, _ := tokenizer.TagName()
			if tag := string(name); !slices.Contains(voidElements, tag) {
				open = append(open, tag)
			}
		case html.EndTagToken:
			name, _ := tokenizer.TagName()
			if at := lastIndex(open, string(name)); at >= 0 {
				open = open[:at]
			}
		case html.TextToken:
			if !slices.ContainsFunc(open, content.ForbiddenZone) {
				out = append(out, occurrences(raw[position:position+size], position, anchor)...)
			}
		default:
		}
		position += size
	}
}

func occurrences(segment string, base int, anchor string) []byteSpan {
	text, at := decoded(segment)
	out := make([]byteSpan, 0, 1)
	for offset := 0; offset < len(text); {
		start, end, found := content.FindAnchor(text[offset:], anchor)
		if !found {
			return out
		}
		start, end = start+offset, end+offset
		offset = end
		if at[start] >= 0 && at[end] >= 0 {
			out = append(out, byteSpan{start: base + at[start], end: base + at[end]})
		}
	}
	return out
}

func decoded(segment string) (text string, at []int) {
	var builder strings.Builder
	at = make([]int, 0, len(segment)+1)
	for i := 0; i < len(segment); {
		size, unit, verbatim := unitAt(segment, i)
		for k := range len(unit) {
			switch {
			case k == 0:
				at = append(at, i)
			case verbatim:
				at = append(at, i+k)
			default:
				at = append(at, -1)
			}
		}
		builder.WriteString(unit)
		i += size
	}
	return builder.String(), append(at, len(segment))
}

func unitAt(segment string, i int) (size int, unit string, verbatim bool) {
	switch segment[i] {
	case '&':
		end := i + 1
		for end < len(segment) && end-i < entityLimit && entityRune(segment[end]) {
			end++
		}
		if end < len(segment) && segment[end] == ';' {
			end++
		}
		if piece := segment[i:end]; html.UnescapeString(piece) != piece {
			return end - i, html.UnescapeString(piece), false
		}
		return 1, "&", true
	case '\r':
		if i+1 < len(segment) && segment[i+1] == '\n' {
			return 2, "\n", false
		}
		return 1, "\n", false
	default:
		_, size = utf8.DecodeRuneInString(segment[i:])
		return size, segment[i : i+size], true
	}
}

func entityRune(c byte) bool {
	return c == '#' || (c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}
