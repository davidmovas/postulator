package content

import (
	"strings"

	"github.com/davidmovas/postulator/internal/domain/template"
)

type Sources struct {
	rules      template.SourceLinks
	cited      map[string]string
	researched bool
}

func SourcesOf(research *Research, rules template.SourceLinks) Sources {
	sources := Sources{rules: rules, cited: map[string]string{}}
	if research == nil {
		return sources
	}
	sources.researched = true
	for _, citation := range research.Citations() {
		if citation.URL != "" {
			sources.cited[sourceKeyOf(citation.URL)] = citation.URL
		}
	}
	return sources
}

func (s Sources) Rules() template.SourceLinks {
	return s.rules
}

func (s Sources) Cites() bool {
	return s.rules.Cites() && len(s.cited) > 0
}

func (s Sources) Found() int {
	return len(s.cited)
}

type SourceLink struct {
	Href   string `json:"href"`
	Anchor string `json:"anchor"`
}

func (s Sources) Linked(doc *Document, lc LinkContext) []SourceLink {
	out := []SourceLink{}
	for _, link := range doc.Links() {
		if lc.Resolve(link.Href).Class != ClassExternal {
			continue
		}
		if target, cited := s.lookup(link.Href); cited {
			out = append(out, SourceLink{Href: target, Anchor: strings.TrimSpace(link.Anchor)})
		}
	}
	return out
}

func (s Sources) lookup(href string) (string, bool) {
	target, cited := s.cited[sourceKeyOf(href)]
	return target, cited
}

type DropReason string

const (
	DroppedUncited DropReason = "uncited"
	DroppedWords   DropReason = "words"
	DroppedRepeat  DropReason = "repeat"
	DroppedMost    DropReason = "most"
	DroppedSite    DropReason = "site"
)

var dropPhrases = map[DropReason]string{
	DroppedUncited: "not cited by the research",
	DroppedWords:   "without words of its own",
	DroppedRepeat:  "already linked",
	DroppedMost:    "over the template's most links to sources",
	DroppedSite:    "over the template's most links to one site",
}

type DroppedSource struct {
	Href   string     `json:"href"`
	Anchor string     `json:"anchor"`
	Reason DropReason `json:"reason"`
}

func (d DroppedSource) said() string {
	return sourceSite(d.Href) + " " + dropPhrases[d.Reason]
}

type sourceCount struct {
	placed map[string]bool
	sites  map[string]int
	total  int
}

func newSourceCount() sourceCount {
	return sourceCount{placed: map[string]bool{}, sites: map[string]int{}}
}

func (c *sourceCount) admits(target string, rules template.SourceLinks) bool {
	return c.refusal(target, rules) == ""
}

func (c *sourceCount) refusal(target string, rules template.SourceLinks) DropReason {
	switch {
	case c.placed[target]:
		return DroppedRepeat
	case c.total >= rules.Max:
		return DroppedMost
	case rules.PerDomain > 0 && c.sites[sourceSite(target)] >= rules.PerDomain:
		return DroppedSite
	default:
		return ""
	}
}

func (c *sourceCount) add(target string) {
	c.placed[target] = true
	c.sites[sourceSite(target)]++
	c.total++
}

func bareAddress(anchor string) bool {
	text := strings.ToLower(strings.TrimSpace(anchor))
	return text == "" || strings.HasPrefix(text, "http://") || strings.HasPrefix(text, "https://") || strings.HasPrefix(text, "www.")
}
