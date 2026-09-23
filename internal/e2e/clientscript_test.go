//go:build e2e

package e2e_test

import (
	"strconv"
	"strings"
	"sync"

	"github.com/davidmovas/postulator/internal/adapters/llm/fake"
	"github.com/davidmovas/postulator/internal/application/graph"
	port "github.com/davidmovas/postulator/internal/application/llm"
	"github.com/davidmovas/postulator/internal/runtime/steps"
)

const (
	markerSections = "SECTIONS"
	markerPhrases  = "REQUIRED PHRASES"
	markerChildren = "CHILDREN"
	noneEntry      = "none"
)

var promptMarkers = []string{markerSections, markerPhrases, markerChildren}

type clientScript struct {
	mu      sync.Mutex
	related []relatedPair
}

type relatedPair struct {
	from   string
	to     string
	reason string
}

func (c *clientScript) relate(pairs ...relatedPair) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.related = pairs
}

func (c *clientScript) replies() []fake.Reply {
	return []fake.Reply{
		{Step: steps.NameGenerateBody, Make: bodyFromPrompt},
		{Step: steps.NameGenerateMeta, Make: metaFromPrompt},
		{Step: steps.NameRepairLinks, Make: sentenceFromPrompt},
		{Step: steps.NameJudge, Text: guideJudge},
		{Step: graph.NameProposeRelated, Make: c.relatedFromPrompt},
	}
}

func (c *clientScript) relatedFromPrompt(port.Request) string {
	c.mu.Lock()
	defer c.mu.Unlock()

	edges := make([]string, 0, len(c.related))
	for _, pair := range c.related {
		edges = append(edges, `{"from":`+quoted(pair.from)+`,"to":`+quoted(pair.to)+
			`,"weight":0.8,"reason":`+quoted(pair.reason)+`}`)
	}
	return `{"edges":[` + strings.Join(edges, ",") + `]}`
}

func quoted(text string) string {
	return strconv.Quote(text)
}

func lineAfter(prompt, label string) string {
	for line := range strings.SplitSeq(prompt, "\n") {
		trimmed := strings.TrimSpace(line)
		if after, found := strings.CutPrefix(trimmed, label); found {
			return strings.TrimSpace(after)
		}
	}
	return ""
}

func listUnder(prompt, marker string) []string {
	out := make([]string, 0, 8)
	reading := false
	for line := range strings.SplitSeq(prompt, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == marker {
			reading = true
			continue
		}
		if !reading {
			continue
		}
		if isMarker(trimmed) {
			return out
		}
		entry, found := strings.CutPrefix(trimmed, "- ")
		if !found {
			continue
		}
		if entry = strings.TrimSpace(entry); entry != "" && entry != noneEntry {
			out = append(out, entry)
		}
	}
	return out
}

func isMarker(line string) bool {
	for _, marker := range promptMarkers {
		if line == marker {
			return true
		}
	}
	return false
}

func headingOf(entry string) string {
	heading := entry
	for _, cut := range []string{" (required)", ", about ", ": "} {
		if index := strings.Index(heading, cut); index >= 0 {
			heading = heading[:index]
		}
	}
	return strings.TrimSpace(heading)
}

func sentences(phrases []string, lead string) string {
	out := make([]string, 0, len(phrases)+1)
	if lead != "" {
		out = append(out, lead)
	}
	for _, phrase := range phrases {
		out = append(out, "Read "+phrase+" for the detail this page leaves out")
	}
	return strings.Join(out, ". ") + "."
}

func filler(topic string, times int) string {
	body := make([]string, 0, times)
	for i := range times {
		body = append(body, "The shop checks every claim on this page about "+topic+
			" against the workshop notes before it goes out, pass "+strconv.Itoa(i+1))
	}
	return strings.Join(body, ". ") + "."
}

func bodyFromPrompt(req port.Request) string {
	prompt := req.Messages[len(req.Messages)-1].Text

	title := lineAfter(prompt, "Title: ")
	h1 := lineAfter(prompt, "H1: ")
	primary := lineAfter(prompt, "Primary keyword: ")
	headings := listUnder(prompt, markerSections)
	phrases := listUnder(prompt, markerPhrases)
	children := listUnder(prompt, markerChildren)

	lead := "This page is the shop's own account of " + primary +
		", written for a reader who has not bought one before"

	draft := draftHead(title, h1, primary)
	draft.WriteString(`,"sections":[`)
	for i, entry := range headings {
		heading := headingOf(entry)
		if i > 0 {
			draft.WriteString(",")
		}
		paragraphs := "<p>" + sentences(phrases, lead) + "</p><p>" + filler(primary, 6) + "</p>"
		if i > 0 {
			paragraphs = "<p>" + filler(heading, 8) + "</p>"
		}
		if i == len(headings)-1 && len(children) > 0 {
			paragraphs += "<p>" + sentences(children, "") + "</p>"
		}
		draft.WriteString(`{"heading":` + quoted(heading) + `,"html":` + quoted(paragraphs) + `}`)
	}
	draft.WriteString(`],"summary":` + quoted("What the shop knows about "+primary+".") + `}`)
	return draft.String()
}

func draftHead(title, h1, primary string) *strings.Builder {
	out := &strings.Builder{}
	if !strings.Contains(strings.ToLower(title), strings.ToLower(primary)) {
		title = title + ": " + primary
	}
	if !strings.Contains(strings.ToLower(h1), strings.ToLower(primary)) {
		h1 = h1 + ": " + primary
	}
	out.WriteString(`{"title":` + quoted(title) + `,"h1":` + quoted(h1))
	return out
}

func metaFromPrompt(req port.Request) string {
	prompt := req.Messages[len(req.Messages)-1].Text

	primary := lineAfter(prompt, "Primary keyword: ")
	canonical := strings.TrimSuffix(lineAfter(prompt, "Canonical: "), ".")
	title := lineAfter(prompt, "The title follows this shape exactly: ")
	if title == "" {
		title = primary
	}

	description := "What the shop knows about " + primary + " and what to read next."
	if len(description) > 155 {
		description = description[:155]
	}
	return `{"title":` + quoted(title) + `,"description":` + quoted(description) +
		`,"canonical":` + quoted(canonical) +
		`,"ogTitle":` + quoted(primary) + `,"ogDescription":` + quoted(description) + `}`
}

func sentenceFromPrompt(req port.Request) string {
	prompt := req.Messages[len(req.Messages)-1].Text

	phrase := ""
	reading := false
	for line := range strings.SplitSeq(prompt, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "PHRASE TO INCLUDE" {
			reading = true
			continue
		}
		if reading && trimmed != "" {
			phrase = trimmed
			break
		}
	}
	return `{"sentence":` + quoted("The shop keeps a page on "+phrase+" for readers who want the whole story.") + `}`
}
