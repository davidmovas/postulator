package content

import (
	stdhtml "html"
	"regexp"
	"strconv"
	"strings"

	"github.com/davidmovas/postulator/internal/domain/template"
)

const (
	CodePrimaryMissingInName             = "primary_missing_in_name"
	CodePrimaryMissingInShortDescription = "primary_missing_in_short_description"
)

var (
	blankLines  = regexp.MustCompile(`\n[ \t\r]*\n`)
	textEscaper = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")
)

func StoreName(name string) string {
	return stdhtml.UnescapeString(strings.TrimSpace(name))
}

func WritesShortDescription(product *template.Product) bool {
	return product != nil && product.ShortDescription.Enabled
}

func ShortDescriptionRules(product template.Product) []string {
	short := product.ShortDescription
	rules := make([]string, 0, 3)
	if intent := strings.TrimSpace(short.Intent); intent != "" {
		rules = append(rules, "The template asks: "+intent)
	}
	if short.TargetWords > 0 {
		rules = append(rules, "About "+strconv.Itoa(short.TargetWords)+" words")
	}
	if short.PrimaryKeyword {
		rules = append(rules, "It carries the primary keyword")
	}
	return rules
}

func PlainParagraphs(plain string) string {
	paragraphs := make([]string, 0)
	for _, chunk := range blankLines.Split(strings.ReplaceAll(plain, "\r\n", "\n"), -1) {
		text := strings.Join(strings.Fields(chunk), " ")
		if text != "" {
			paragraphs = append(paragraphs, "<p>"+textEscaper.Replace(text)+"</p>")
		}
	}
	return strings.Join(paragraphs, "\n")
}

func plainText(value string) string {
	doc, err := Parse(value)
	if err != nil {
		return strings.TrimSpace(value)
	}
	return strings.Join(strings.Fields(doc.Text()), " ")
}

func ShortDescriptionFindings(short string, product template.Product, primary string) []Finding {
	out := make([]Finding, 0)
	rule := product.ShortDescription
	if !rule.Enabled || !rule.PrimaryKeyword || primary == "" {
		return out
	}
	if !carries(plainText(short), primary) {
		out = append(out, Finding{
			Severity: SeverityError, Code: CodePrimaryMissingInShortDescription,
			Message: "the primary keyword does not appear in the short description",
			Details: map[string]any{"primaryKeyword": primary},
		})
	}
	return out
}

func ForStore(report Report) Report {
	graded := Report{Items: make([]Finding, 0, len(report.Items))}
	for _, finding := range report.Items {
		if finding.Code == CodePlanH1LacksKeyword {
			continue
		}
		if finding.Code == CodePrimaryMissingInH1 {
			finding = Finding{
				Severity: SeverityWarn, Code: CodePrimaryMissingInName,
				Message: "the primary keyword is not in the product's name, which the store owns; " +
					"rename the product in WooCommerce for the name to carry it",
				Details: finding.Details,
			}
		}
		graded.Items = append(graded.Items, finding)
	}
	graded.Score = scoreOf(graded.Items)
	return graded
}
