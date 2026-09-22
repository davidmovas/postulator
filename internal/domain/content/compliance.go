package content

import (
	"strconv"
	"strings"

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

	CountedGraphLinks = "graph_links"

	errorPenalty = 0.25
	warnPenalty  = 0.05
)

func Compliance(doc *Document, lc LinkContext, policy template.LinkPolicy, pageID string) Report {
	report := Report{Items: make([]Finding, 0)}

	for _, target := range lc.Targets {
		if len(existingFor(doc, lc, target)) > 0 {
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

	graphLinks := 0
	for _, link := range doc.Links() {
		href := strings.TrimSpace(link.Href)
		resolution := lc.Resolve(href)
		if resolution.SameDocument {
			continue
		}

		switch resolution.Class {
		case ClassGraph:
			graphLinks++
			if !AnchorAllowed(resolution.Target, link.Anchor) {
				report.Items = append(report.Items, Finding{
					Severity: SeverityWarn, Code: CodeAnchorNotAllowed,
					Message: "the anchor " + link.Anchor + " is not one of the anchors of " + resolution.Target.URL,
					Details: map[string]any{"href": href, "path": resolution.Path, "anchor": link.Anchor},
				})
			}
		case ClassSelf:
			if policy.ForbidSelf {
				report.Items = append(report.Items, Finding{
					Severity: SeverityError, Code: CodeSelfLink,
					Message: "the page links to itself",
					Details: map[string]any{"href": href, "path": resolution.Path},
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
				Details: map[string]any{"href": href, "path": resolution.Path},
			})
		}
	}

	if policy.Rules.MaxLinks > 0 && graphLinks > policy.Rules.MaxLinks {
		report.Items = append(report.Items, Finding{
			Severity: SeverityWarn, Code: CodeTooManyLinks,
			Message: "the page carries " + strconv.Itoa(graphLinks) + " graph links, more than the " +
				strconv.Itoa(policy.Rules.MaxLinks) + " its rules allow",
			Details: map[string]any{
				"links": graphLinks, "counted": CountedGraphLinks, "maxLinks": policy.Rules.MaxLinks,
			},
		})
	}

	report.Score = scoreOf(report.Items)
	return report
}

func ScoreOf(items []Finding) float64 {
	return scoreOf(items)
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
