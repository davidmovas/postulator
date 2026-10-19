package contextpack

import (
	"strings"

	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/render"
	"github.com/davidmovas/postulator/internal/domain/site"
)

func LinkForm(setting site.URLForm, links []pagemap.PageLink, renderForms []string) site.URLForm {
	if setting == site.URLFormAbsolute || setting == site.URLFormRelative {
		return setting
	}
	absolute, relative := 0, 0
	for i := range links {
		link := links[i]
		if link.Type != pagemap.LinkContextual || link.Source != pagemap.SourceRaw || link.Ours || link.ToPageID == nil {
			continue
		}
		if absoluteHref(link.ToURL) {
			absolute++
		} else {
			relative++
		}
	}
	if absolute+relative == 0 {
		for _, form := range renderForms {
			switch form {
			case render.URLFormAbsolute:
				absolute++
			case render.URLFormRelative:
				relative++
			}
		}
	}
	if absolute > relative {
		return site.URLFormAbsolute
	}
	return site.URLFormRelative
}

func absoluteHref(href string) bool {
	lower := strings.ToLower(strings.TrimSpace(href))
	return strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") || strings.HasPrefix(lower, "//")
}
