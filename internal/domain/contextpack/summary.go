package contextpack

import (
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
)

type SummarySource string

const (
	FromMeta      SummarySource = "meta"
	FromExcerpt   SummarySource = "excerpt"
	FromBody      SummarySource = "body"
	FromStructure SummarySource = "structure"
)

type Summary struct {
	Text   string        `json:"text"`
	Source SummarySource `json:"source"`
}

const (
	summaryRunes = 240
	namesShown   = 3
)

func (s Snapshot) Summary(page pagemap.Page) Summary {
	if text := clean(s.text(page.ID, field.SEODescription)); text != "" {
		return Summary{Text: clip(text), Source: FromMeta}
	}
	excerpt := field.Excerpt
	if page.WPType == pagemap.WPProduct {
		excerpt = field.ProductShortDescription
	}
	if text := clean(s.text(page.ID, excerpt)); text != "" {
		return Summary{Text: clip(text), Source: FromExcerpt}
	}
	if text := s.firstSentence(page); text != "" {
		return Summary{Text: clip(text), Source: FromBody}
	}
	return Summary{Text: s.structure(page), Source: FromStructure}
}

func (s Snapshot) firstSentence(page pagemap.Page) string {
	body, found := s.body(page.ID)
	if !found {
		return ""
	}
	for _, text := range body.TextBlocks() {
		if sentence := sentenceOf(clean(text)); sentence != "" {
			return sentence
		}
	}
	return ""
}

func (s Snapshot) structure(page pagemap.Page) string {
	kind := s.typeOf(page).Name
	if kind == "" {
		kind = "Page"
	}
	var builder strings.Builder
	builder.WriteString(kind)
	builder.WriteString(" for ")
	builder.WriteString(s.h1Of(page))
	if parent, found := s.parent(page); found {
		builder.WriteString(" under ")
		builder.WriteString(s.titleOf(parent))
	}
	if products, _ := s.products(page); len(products) > 0 {
		builder.WriteString("; lists ")
		builder.WriteString(s.namesOf(products))
	}
	if children := s.children(page); len(children) > 0 {
		builder.WriteString("; its children are ")
		builder.WriteString(s.namesOf(children))
	}
	return builder.String()
}

func (s Snapshot) namesOf(pages []pagemap.Page) string {
	listed := pages[:min(len(pages), namesShown)]
	names := make([]string, 0, len(listed))
	for i := range listed {
		names = append(names, s.titleOf(listed[i]))
	}
	out := strings.Join(names, ", ")
	if rest := len(pages) - len(listed); rest > 0 {
		out += " and " + strconv.Itoa(rest) + " more"
	}
	return out
}

func clean(text string) string {
	if strings.ContainsAny(text, "<&") {
		if doc, err := content.Parse(text); err == nil {
			return doc.Text()
		}
	}
	return strings.Join(strings.Fields(text), " ")
}

func sentenceOf(text string) string {
	for at, r := range text {
		if r != '.' && r != '!' && r != '?' {
			continue
		}
		next := at + utf8.RuneLen(r)
		if next == len(text) || text[next] == ' ' {
			return text[:next]
		}
	}
	return text
}

func clip(text string) string {
	if utf8.RuneCountInString(text) <= summaryRunes {
		return text
	}
	runes := []rune(text)
	return strings.TrimSpace(string(runes[:summaryRunes])) + "…"
}
