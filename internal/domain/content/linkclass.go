package content

import (
	"strings"

	"github.com/davidmovas/postulator/internal/domain/pagemap"
)

type LinkClass string

const (
	ClassGraph           LinkClass = "graph"
	ClassSelf            LinkClass = "self"
	ClassExternal        LinkClass = "external"
	ClassUnknownInternal LinkClass = "unknown_internal"
)

func (c LinkClass) OffGraph() bool {
	return c != ClassGraph
}

type Resolution struct {
	Path         string
	Target       LinkTarget
	Class        LinkClass
	SameDocument bool
}

func (c LinkContext) Resolve(href string) Resolution {
	path, kind := c.Site.Resolve(href)
	switch kind {
	case pagemap.LinkExternal:
		return Resolution{Class: ClassExternal}
	case pagemap.LinkSameDocument:
		return Resolution{Class: ClassSelf, SameDocument: true}
	case pagemap.LinkUnresolved:
		return Resolution{Class: ClassUnknownInternal}
	}

	if own, ok := canonicalPath(c.PageURL); ok && own == path {
		return Resolution{Path: path, Class: ClassSelf}
	}
	for _, target := range c.Targets {
		if wanted, ok := canonicalPath(target.URL); ok && wanted == path {
			return Resolution{Path: path, Target: target, Class: ClassGraph}
		}
	}
	return Resolution{Path: path, Class: ClassUnknownInternal}
}

func (c LinkContext) ClassifyLink(link pagemap.PageLink) LinkClass {
	if link.ToPageID != nil {
		if *link.ToPageID == c.PageID {
			return ClassSelf
		}
		if _, ok := c.ByPageID(*link.ToPageID); ok {
			return ClassGraph
		}
		return ClassUnknownInternal
	}
	return c.Resolve(link.ToURL).Class
}

func canonicalPath(raw string) (string, bool) {
	path, kind := pagemap.Site{}.Resolve(raw)
	if kind != pagemap.LinkPath {
		return "", false
	}
	return path, true
}

func AnchorAllowed(target LinkTarget, anchor string) bool {
	trimmed := strings.TrimSpace(anchor)
	for _, allowed := range target.Anchors {
		if strings.EqualFold(allowed, trimmed) {
			return true
		}
	}
	return false
}
