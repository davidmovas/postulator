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

func (c LinkContext) ClassifyLink(link pagemap.PageLink, host string) LinkClass {
	if link.ToPageID != nil {
		switch {
		case *link.ToPageID == c.PageID:
			return ClassSelf
		default:
			if _, ok := c.ByPageID(*link.ToPageID); ok {
				return ClassGraph
			}
			return ClassUnknownInternal
		}
	}

	path, internal := pagemap.InternalPath(link.ToURL, host)
	switch {
	case !internal:
		return ClassExternal
	case path != "" && path == c.PageURL:
		return ClassSelf
	}
	if _, ok := c.ByURL(path); ok {
		return ClassGraph
	}
	return ClassUnknownInternal
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
