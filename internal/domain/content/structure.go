package content

import (
	"strings"

	"golang.org/x/net/html"

	"github.com/davidmovas/postulator/internal/domain/template"
)

const (
	CodePrimaryMissingInH1   = "primary_missing_in_h1"
	CodePrimaryMissingInLead = "primary_missing_in_first_paragraph"
	CodeSecondaryMissing     = "secondary_keyword_missing"
	CodeKeywordDensity       = "keyword_density_too_high"
	CodeWordCount            = "word_count_out_of_range"
	CodeSectionMissing       = "section_missing"
	CodeEmptyHeading         = "empty_heading"
	CodeNoHeadings           = "no_headings"
)

func Structure(doc *Document, primary string, secondary []string, spec template.TemplateSpec) Report {
	report := Report{Items: make([]Finding, 0)}
	words := doc.Words()

	report.Items = append(report.Items, keywordFindings(doc, primary, secondary, spec, words)...)
	report.Items = append(report.Items, headingFindings(doc, spec)...)
	report.Items = append(report.Items, lengthFindings(spec, len(words))...)

	report.Score = scoreOf(report.Items)
	return report
}

func keywordFindings(doc *Document, primary string, secondary []string, spec template.TemplateSpec, words []string) []Finding {
	out := make([]Finding, 0)
	primary = strings.TrimSpace(primary)
	if primary == "" {
		return out
	}

	if spec.KeywordRules.PrimaryInH1 {
		heading := firstOf(doc.elements("h1"))
		if _, _, found := findFold(heading, primary); !found {
			out = append(out, Finding{
				Severity: SeverityError, Code: CodePrimaryMissingInH1,
				Message: "the primary keyword does not appear in the h1",
				Details: map[string]any{"primaryKeyword": primary},
			})
		}
	}
	if spec.KeywordRules.PrimaryInFirstParagraph {
		lead := firstOf(doc.Paragraphs())
		if _, _, found := findFold(lead, primary); !found {
			out = append(out, Finding{
				Severity: SeverityError, Code: CodePrimaryMissingInLead,
				Message: "the primary keyword does not appear in the first paragraph",
				Details: map[string]any{"primaryKeyword": primary},
			})
		}
	}

	text := doc.Text()
	for _, keyword := range secondary {
		trimmed := strings.TrimSpace(keyword)
		if trimmed == "" {
			continue
		}
		if _, _, found := findFold(text, trimmed); !found {
			out = append(out, Finding{
				Severity: SeverityWarn, Code: CodeSecondaryMissing,
				Message: "the secondary keyword " + trimmed + " does not appear in the body",
				Details: map[string]any{"keyword": trimmed},
			})
		}
	}

	if spec.KeywordRules.MaxDensity > 0 && len(words) > 0 {
		density := float64(occurrences(text, primary)*len(strings.Fields(primary))) / float64(len(words))
		if density > spec.KeywordRules.MaxDensity {
			out = append(out, Finding{
				Severity: SeverityWarn, Code: CodeKeywordDensity,
				Message: "the primary keyword is repeated more often than the template allows",
				Details: map[string]any{"density": density, "maxDensity": spec.KeywordRules.MaxDensity},
			})
		}
	}
	return out
}

func headingFindings(doc *Document, spec template.TemplateSpec) []Finding {
	out := make([]Finding, 0)
	headings := doc.Headings()

	if len(headings) == 0 {
		return append(out, Finding{
			Severity: SeverityError, Code: CodeNoHeadings,
			Message: "the body carries no heading",
		})
	}

	texts := make([]string, 0, len(headings))
	for _, heading := range headings {
		text := TextOf(heading)
		if text == "" {
			out = append(out, Finding{
				Severity: SeverityWarn, Code: CodeEmptyHeading,
				Message: "the body carries an empty " + heading.Data,
			})
			continue
		}
		texts = append(texts, text)
	}

	for _, section := range spec.Sections {
		if !section.Required {
			continue
		}
		if !containsFold(texts, section.Heading) {
			out = append(out, Finding{
				Severity: SeverityError, Code: CodeSectionMissing,
				Message: "the required section " + section.Heading + " is missing",
				Details: map[string]any{"heading": section.Heading},
			})
		}
	}
	return out
}

func lengthFindings(spec template.TemplateSpec, words int) []Finding {
	out := make([]Finding, 0)
	if spec.Length.Min > 0 && words < spec.Length.Min {
		out = append(out, Finding{
			Severity: SeverityWarn, Code: CodeWordCount,
			Message: "the body is shorter than the template allows",
			Details: map[string]any{"words": words, "min": spec.Length.Min},
		})
	}
	if spec.Length.Max > 0 && words > spec.Length.Max {
		out = append(out, Finding{
			Severity: SeverityWarn, Code: CodeWordCount,
			Message: "the body is longer than the template allows",
			Details: map[string]any{"words": words, "max": spec.Length.Max},
		})
	}
	return out
}

func firstOf(nodes []*html.Node) string {
	if len(nodes) == 0 {
		return ""
	}
	return TextOf(nodes[0])
}

func containsFold(values []string, needle string) bool {
	trimmed := strings.TrimSpace(needle)
	if trimmed == "" {
		return true
	}
	for _, value := range values {
		if _, _, found := findFold(value, trimmed); found {
			return true
		}
	}
	return false
}

func occurrences(haystack, needle string) int {
	if needle == "" {
		return 0
	}

	count := 0
	rest := haystack
	for {
		_, end, found := findFold(rest, needle)
		if !found {
			return count
		}
		count++
		rest = rest[end:]
	}
}
