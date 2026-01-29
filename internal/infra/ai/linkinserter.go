package ai

import (
	"fmt"
	"regexp"
	"strings"
)

type forbiddenZone struct {
	start int
	end   int
}

func applyLinkPlacements(content string, placements []LinkPlacement) (string, int) {
	if len(placements) == 0 {
		return content, 0
	}

	applied := 0

	for _, p := range placements {
		if p.AnchorText == "" || p.TargetURL == "" {
			continue
		}

		zones := computeForbiddenZones(content)

		idx := findAnchorText(content, p.AnchorText, zones)
		if idx == -1 {
			continue
		}

		originalText := content[idx : idx+len(p.AnchorText)]
		replacement := fmt.Sprintf(`<a href="%s">%s</a>`, p.TargetURL, originalText)
		content = content[:idx] + replacement + content[idx+len(p.AnchorText):]
		applied++
	}

	return content, applied
}

func findAnchorText(content, anchorText string, zones []forbiddenZone) int {
	contentLower := strings.ToLower(content)
	searchLower := strings.ToLower(anchorText)
	searchLen := len(searchLower)

	offset := 0
	for {
		idx := strings.Index(contentLower[offset:], searchLower)
		if idx == -1 {
			return -1
		}

		absIdx := offset + idx

		if !isInForbiddenZone(absIdx, absIdx+searchLen, zones) {
			return absIdx
		}

		offset = absIdx + 1
		if offset >= len(contentLower) {
			return -1
		}
	}
}

func computeForbiddenZones(content string) []forbiddenZone {
	var zones []forbiddenZone

	pairedTags := []string{"a", "h1", "h2", "h3", "h4", "h5", "h6"}

	for _, tag := range pairedTags {
		pattern := fmt.Sprintf(`(?is)<%s[\s>].*?</%s\s*>`, tag, tag)
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringIndex(content, -1)
		for _, m := range matches {
			zones = append(zones, forbiddenZone{start: m[0], end: m[1]})
		}
	}

	tagRe := regexp.MustCompile(`<[^>]+>`)
	tagMatches := tagRe.FindAllStringIndex(content, -1)
	for _, m := range tagMatches {
		zones = append(zones, forbiddenZone{start: m[0], end: m[1]})
	}

	return zones
}

func isInForbiddenZone(start, end int, zones []forbiddenZone) bool {
	for _, z := range zones {
		if start < z.end && end > z.start {
			return true
		}
	}
	return false
}
