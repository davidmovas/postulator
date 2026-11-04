package content

import (
	"strconv"
	"strings"

	"github.com/davidmovas/postulator/internal/domain/field"
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
	CodeTargetMissing      = "target_missing"
	CodeTargetNotPublished = "target_not_published"
	CodeExternalLink       = "external_link"
	CodeSelfLink           = "self_link"
	CodeUnknownInternal    = "unknown_internal_link"
	CodeAnchorNotAllowed   = "anchor_not_allowed"
	CodeTooManyLinks       = "too_many_links"
	CodeTooManySourceLinks = "too_many_source_links"
	CodeTooFewSourceLinks  = "too_few_source_links"

	CountedGraphLinks = "graph_links"

	errorPenalty = 0.25
	warnPenalty  = 0.05
)

func Compliance(doc *Document, lc LinkContext, policy field.LinkPolicy, sources Sources, pageID string) Report {
	return ComplianceOf(doc, doc, lc, policy, sources, pageID)
}

func ComplianceOf(owed, judged *Document, lc LinkContext, policy field.LinkPolicy, sources Sources, pageID string) Report {
	report := Report{Items: make([]Finding, 0)}

	for _, target := range lc.Targets {
		if len(existingFor(owed, lc, target)) > 0 {
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
	cited := newSourceCount()
	crowded := false
	for _, link := range judged.Links() {
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
			source, known := sources.lookup(href)
			if !known || !sources.rules.Cites() {
				report.Items = append(report.Items, Finding{
					Severity: SeverityError, Code: CodeExternalLink,
					Message: "the page links outside the site to an address its research did not cite",
					Details: map[string]any{"href": href},
				})
				break
			}
			crowded = crowded || !cited.admits(source, sources.rules)
			cited.add(source)
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
	report.Items = append(report.Items, sourceFindings(sources, cited, crowded)...)

	report.Score = scoreOf(report.Items)
	return report
}

func sourceFindings(sources Sources, cited sourceCount, crowded bool) []Finding {
	rules := sources.rules
	switch {
	case crowded:
		return []Finding{{
			Severity: SeverityWarn, Code: CodeTooManySourceLinks,
			Message: "the page carries " + strconv.Itoa(cited.total) + " links to its sources, more than its template allows",
			Details: map[string]any{"linked": cited.total, "max": rules.Max, "perDomain": rules.PerDomain},
		}}
	case sources.researched && cited.total < rules.Min:
		return []Finding{{
			Severity: SeverityWarn, Code: CodeTooFewSourceLinks,
			Message: "the page links " + strconv.Itoa(cited.total) + " of its sources, fewer than the " + strconv.Itoa(rules.Min) +
				" its template asks; the research cited " + strconv.Itoa(sources.Found()),
			Details: map[string]any{"linked": cited.total, "min": rules.Min, "found": sources.Found()},
		}}
	default:
		return nil
	}
}

func Unpublished(doc *Document, lc LinkContext, live map[string]bool, pageID string) []Finding {
	out := make([]Finding, 0)
	for _, target := range lc.Targets {
		if live[target.PageID] || len(existingFor(doc, lc, target)) == 0 {
			continue
		}
		out = append(out, Finding{
			Severity: SeverityWarn, Code: CodeTargetNotPublished,
			Message: "the page links to " + target.URL + ", which is not on the site yet, so the link leads " +
				"nowhere until that page is published",
			Details: map[string]any{"pageId": pageID, "targetPageId": target.PageID, "relation": string(target.Relation)},
		})
	}
	return out
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
