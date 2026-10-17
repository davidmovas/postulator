package block

import (
	"regexp"
	"slices"
	"strings"

	"golang.org/x/net/html"

	"github.com/davidmovas/postulator/internal/domain/content"
)

var voidElements = []string{"area", "base", "br", "col", "embed", "hr", "img", "input", "link", "meta", "source", "track", "wbr"}

var inlineBreaks = []string{"br", "wbr"}

var blankLine = regexp.MustCompile(`\r?\n[ \t]*\r?\n`)

type classicParser struct {
	raw      string
	spans    []span
	stack    []string
	runStart int
	pos      int
}

func parseClassic(raw string) []span {
	p := &classicParser{raw: raw, spans: make([]span, 0), runStart: -1}
	tokenizer := html.NewTokenizer(strings.NewReader(raw))
	blockStart := 0

	for {
		tokenType := tokenizer.Next()
		if tokenType == html.ErrorToken {
			break
		}
		size := len(tokenizer.Raw())
		name, _ := tokenizer.TagName()
		tag := string(name)

		switch tokenType {
		case html.StartTagToken, html.SelfClosingTagToken:
			closed := tokenType == html.SelfClosingTagToken || slices.Contains(voidElements, tag)
			if len(p.stack) > 0 && p.stack[len(p.stack)-1] == "p" && content.IsBlockElement(tag) && !slices.Contains(inlineBreaks, tag) {
				p.stack = p.stack[:0]
				p.spans = append(p.spans, elementSpan(blockStart, p.pos, "p"))
			}
			switch {
			case len(p.stack) > 0:
				if !closed {
					p.stack = append(p.stack, tag)
				}
			case content.IsBlockElement(tag) && !slices.Contains(inlineBreaks, tag), slices.Contains(voidElements, tag) && !slices.Contains(inlineBreaks, tag):
				p.flushRun(p.pos)
				blockStart = p.pos
				if closed {
					p.spans = append(p.spans, elementSpan(blockStart, p.pos+size, tag))
				} else {
					p.stack = append(p.stack, tag)
				}
			default:
				p.openRun()
			}
		case html.EndTagToken:
			switch index := lastIndex(p.stack, tag); {
			case index >= 0:
				p.stack = p.stack[:index]
				if len(p.stack) == 0 {
					p.spans = append(p.spans, elementSpan(blockStart, p.pos+size, tag))
				}
			case len(p.stack) == 0:
				p.openRun()
			}
		default:
			if len(p.stack) == 0 {
				p.openRun()
			}
		}
		p.pos += size
	}
	if len(p.stack) > 0 {
		p.spans = append(p.spans, elementSpan(blockStart, len(raw), p.stack[0]))
	}
	p.flushRun(len(raw))
	return p.spans
}

func (p *classicParser) openRun() {
	if p.runStart < 0 {
		p.runStart = p.pos
	}
}

func (p *classicParser) flushRun(end int) {
	if p.runStart < 0 {
		return
	}
	start := p.runStart
	p.runStart = -1
	text := p.raw[start:end]
	offset := 0
	for _, cut := range append(blankLine.FindAllStringIndex(text, -1), []int{len(text), len(text)}) {
		segment := text[offset:cut[0]]
		if strings.TrimSpace(segment) != "" {
			p.spans = append(p.spans, textSpan(start+offset, start+cut[0], segment))
		}
		offset = cut[1]
	}
}

func textSpan(start, end int, segment string) span {
	if match := shortcodePattern.FindStringSubmatch(segment); len(match) == 3 && (match[2] == "" || match[2] == match[1]) {
		return span{start: start, end: end, kind: Shortcode, name: match[1], dynamic: true}
	}
	return span{start: start, end: end, kind: Paragraph}
}

func elementSpan(start, end int, tag string) span {
	s := span{start: start, end: end, name: tag}
	switch tag {
	case "p":
		s.kind = Paragraph
		s.name = ""
	case "h1", "h2", "h3", "h4", "h5", "h6":
		s.kind = Heading
	case "ul", "ol":
		s.kind = List
	case "blockquote":
		s.kind = Quote
	default:
		s.kind = Foreign
	}
	return s
}

func lastIndex(stack []string, tag string) int {
	for i, open := range slices.Backward(stack) {
		if open == tag {
			return i
		}
	}
	return -1
}
