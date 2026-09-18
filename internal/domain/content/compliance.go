package content

import (
	"strings"

	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/template"
)

type Severity string

const (
	SeverityInfo  Severity = "info"
	SeverityWarn  Severity = "warn"
	SeverityError Severity = "error"
)

type Finding struct {
	Severity Severity       `json:"severity"`
	Code     string         `json:"code"`
	Message  string         `json:"message"`
	Details  map[string]any `json:"details,omitempty"`
}

type Report struct {
	Items []Finding `json:"items"`
	Score float64   `json:"score"`
}

func (r Report) HasErrors() bool {
	for _, item := range r.Items {
		if item.Severity == SeverityError {
			return true
		}
	}
	return false
}

const (
	CodeTargetMissing    = "target_missing"
	CodeExternalLink     = "external_link"
	CodeSelfLink         = "self_link"
	CodeUnknownInternal  = "unknown_internal_link"
	CodeAnchorNotAllowed = "anchor_not_allowed"
	CodeTooManyLinks     = "too_many_links"

	errorPenalty = 0.25
	warnPenalty  = 0.05
)

type LinkClass string

const (
	ClassGraph           LinkClass = "graph"
	ClassSelf            LinkClass = "self"
	ClassExternal        LinkClass = "external"
	ClassUnknownInternal LinkClass = "unknown_internal"
)

func Compliance(doc *Document, lc LinkContext, policy template.LinkPolicy, pageID string) Report {
	report := Report{Items: make([]Finding, 0)}

	for _, target := range lc.Targets {
		if len(existingFor(doc, target)) > 0 {
			continue
		}
		severity := SeverityWarn
		if target.Required {
			severity = SeverityError
		}
		report.Items = append(report.Items, Finding{
			Severity: severity, Code: CodeTargetMissing,
			Message: "the page does not link to " + target.URL,
			Details: map[string]any{"pageId": pageID, "targetPageId": target.PageID, "relation": string(target.Relation)},
		})
	}

	links := doc.Links()
	for _, link := range links {
		href := strings.TrimSpace(link.Href)
		switch classify(href, lc) {
		case ClassGraph:
			target, _ := lc.ByURL(href)
			if !allowedAnchor(target, link.Anchor) {
				report.Items = append(report.Items, Finding{
					Severity: SeverityWarn, Code: CodeAnchorNotAllowed,
					Message: "the anchor " + link.Anchor + " is not one of the anchors of " + href,
					Details: map[string]any{"href": href, "anchor": link.Anchor},
				})
			}
		case ClassSelf:
			if policy.ForbidSelf {
				report.Items = append(report.Items, Finding{
					Severity: SeverityError, Code: CodeSelfLink,
					Message: "the page links to itself",
					Details: map[string]any{"href": href},
				})
			}
		case ClassExternal:
			if policy.ForbidExternal {
				report.Items = append(report.Items, Finding{
					Severity: SeverityError, Code: CodeExternalLink,
					Message: "the page links outside the site",
					Details: map[string]any{"href": href},
				})
			}
		case ClassUnknownInternal:
			report.Items = append(report.Items, Finding{
				Severity: SeverityWarn, Code: CodeUnknownInternal,
				Message: "the page links to " + href + ", which the entity graph does not sanction",
				Details: map[string]any{"href": href},
			})
		}
	}

	if policy.Rules.MaxLinks > 0 && len(links) > policy.Rules.MaxLinks {
		report.Items = append(report.Items, Finding{
			Severity: SeverityWarn, Code: CodeTooManyLinks,
			Message: "the page carries more links than the policy allows",
			Details: map[string]any{"links": len(links), "maxLinks": policy.Rules.MaxLinks},
		})
	}

	report.Score = scoreOf(report.Items)
	return report
}

func classify(href string, lc LinkContext) LinkClass {
	switch {
	case href == "":
		return ClassUnknownInternal
	case lc.PageURL != "" && href == lc.PageURL:
		return ClassSelf
	}
	if _, ok := lc.ByURL(href); ok {
		return ClassGraph
	}
	if _, internal := pagemap.InternalPath(href, ""); !internal {
		return ClassExternal
	}
	return ClassUnknownInternal
}

func allowedAnchor(target LinkTarget, anchor string) bool {
	trimmed := strings.TrimSpace(anchor)
	for _, allowed := range target.Anchors {
		if strings.EqualFold(allowed, trimmed) {
			return true
		}
	}
	return false
}

func scoreOf(items []Finding) float64 {
	score := 1.0
	for _, item := range items {
		switch item.Severity {
		case SeverityError:
			score -= errorPenalty
		case SeverityWarn:
			score -= warnPenalty
		case SeverityInfo:
		}
	}
	return max(score, 0)
}
