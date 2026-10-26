package content

import (
	"slices"
	"strings"
)

type Must struct {
	PageID string
	Anchor string
}

func Insist(lc LinkContext, must []Must) (insisted LinkContext, unplanned []string) {
	insisted = lc
	insisted.Targets = make([]LinkTarget, len(lc.Targets))
	for i := range lc.Targets {
		insisted.Targets[i] = lc.Targets[i]
		insisted.Targets[i].Anchors = slices.Clone(lc.Targets[i].Anchors)
	}
	unplanned = make([]string, 0)
	for _, link := range must {
		at := slices.IndexFunc(insisted.Targets, func(target LinkTarget) bool { return target.PageID == link.PageID })
		if at < 0 {
			unplanned = append(unplanned, link.PageID)
			continue
		}
		target := &insisted.Targets[at]
		target.Required = true
		if anchor := strings.TrimSpace(link.Anchor); anchor != "" {
			target.Anchors = append([]string{anchor}, slices.DeleteFunc(target.Anchors, func(held string) bool { return held == anchor })...)
		}
	}
	return insisted, unplanned
}
