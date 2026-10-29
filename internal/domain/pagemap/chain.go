package pagemap

import (
	"html"
	"strings"
)

type ChainBreak string

const (
	ChainWhole       ChainBreak = ""
	ChainNoTrail     ChainBreak = "no_breadcrumbs"
	ChainUnknownLink ChainBreak = "unknown_link"
	ChainAmbiguous   ChainBreak = "ambiguous_name"
)

type ChainLink struct {
	Name   string
	PageID string
	Break  ChainBreak
}

type Chain struct {
	Links []ChainLink
	Label string
	Break ChainBreak
}

func CrumbName(raw string) string {
	return strings.Join(strings.Fields(html.UnescapeString(raw)), " ")
}

func TopicChain(page Page, evidence ParentSignals, index Index) Chain {
	if page.Path == RootPath {
		return Chain{Links: []ChainLink{}, Break: ChainNoTrail}
	}
	var named, broken, rooted *Chain
	for _, ranked := range rankTrails(evidence.Trails, index) {
		chain := chainOf(page, ranked.trail, evidence, index)
		switch {
		case len(chain.Links) == 0:
			if rooted == nil {
				rooted = &chain
			}
		case chain.Break != ChainWhole:
			if broken == nil {
				broken = &chain
			}
		case chain.Links[len(chain.Links)-1].PageID != "":
			return chain
		case named == nil:
			named = &chain
		}
	}
	for _, found := range []*Chain{named, broken, rooted} {
		if found != nil {
			return *found
		}
	}
	return Chain{Links: []ChainLink{}, Break: ChainNoTrail}
}

func chainOf(page Page, t Trail, evidence ParentSignals, index Index) Chain {
	chain := Chain{Links: make([]ChainLink, 0, len(t.Crumbs))}
	for _, crumb := range t.Crumbs {
		if crumb.Path == page.Path {
			chain.Label = CrumbName(crumb.Name)
			break
		}
		if crumb.Path == RootPath {
			continue
		}
		if technical(crumb.Path, index) {
			continue
		}
		link := ChainLink{Name: CrumbName(crumb.Name)}
		match := matchCrumb(page, crumb, evidence.Landings, index)
		switch {
		case match.found && (match.page.Path == RootPath || match.page.TechnicalReason != TechnicalNone):
			continue
		case match.found:
			link.PageID = match.page.ID
		case len(match.candidates) > 0:
			link.Break = ChainAmbiguous
		case match.liveLink:
			link.Break = ChainUnknownLink
		case link.Name == "":
			continue
		}
		chain.Links = append(chain.Links, link)
	}
	if len(chain.Links) > 0 {
		chain.Break = chain.Links[len(chain.Links)-1].Break
	}
	return chain
}

func technical(path string, index Index) bool {
	if path == "" {
		return false
	}
	resolved, found := index.ByPath(path)
	return found && resolved.TechnicalReason != TechnicalNone
}
