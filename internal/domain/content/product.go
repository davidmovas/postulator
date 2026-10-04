package content

import (
	stdhtml "html"
	"strings"

	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/template"
)

const (
	CodePrimaryMissingInName             = "primary_missing_in_name"
	CodePrimaryMissingInShortDescription = "primary_missing_in_short_description"
	CodeSpecificationMissing             = "specification_missing"
)

type AnswerSpecification struct {
	Name  string `json:"name" description:"The specification name, exactly as the brief gives it"`
	Value string `json:"value" description:"Its value as plain text; empty when neither the name nor the page notes state it"`
}

type ProductAnswer struct {
	DraftAnswer
	ShortDescription string                `json:"shortDescription" description:"The short description the store shows beside the price, as HTML paragraphs"`
	Specifications   []AnswerSpecification `json:"specifications" description:"One value for every specification the brief lists"`
}

type Specification struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type ProductDraft struct {
	ShortDescription string          `json:"shortDescription"`
	Specifications   []Specification `json:"specifications"`
}

type BriefSpecification struct {
	Name   string `json:"name"`
	Intent string `json:"intent"`
}

type BriefProduct struct {
	ShortIntent      string               `json:"shortIntent"`
	Specifications   []BriefSpecification `json:"specifications"`
	ShortWords       int                  `json:"shortWords"`
	ShortDescription bool                 `json:"shortDescription"`
	ShortKeyword     bool                 `json:"shortKeyword"`
}

func StoreName(page pagemap.Page) string {
	return stdhtml.UnescapeString(strings.TrimSpace(page.Observed.Title))
}

func productBrief(product template.Product) *BriefProduct {
	brief := &BriefProduct{
		ShortDescription: product.ShortDescription.Enabled,
		ShortIntent:      strings.TrimSpace(product.ShortDescription.Intent),
		ShortWords:       product.ShortDescription.TargetWords,
		ShortKeyword:     product.ShortDescription.PrimaryKeyword,
		Specifications:   make([]BriefSpecification, 0, len(product.Specifications)),
	}
	for _, specification := range product.Specifications {
		brief.Specifications = append(brief.Specifications, BriefSpecification{
			Name: strings.TrimSpace(specification.Name), Intent: strings.TrimSpace(specification.Intent),
		})
	}
	return brief
}

func AssembleProduct(answer ProductAnswer, brief Brief) (ContentDraft, *Document, error) {
	draft, doc, err := Assemble(answer.DraftAnswer, brief)
	if err != nil || brief.Product == nil {
		return draft, doc, err
	}

	product := ProductDraft{Specifications: make([]Specification, 0, len(brief.Product.Specifications))}
	if brief.Product.ShortDescription {
		short, stripped := Sanitize(answer.ShortDescription)
		if strings.TrimSpace(short) == "" {
			return ContentDraft{}, nil, incomplete("the model left the short description empty", nil)
		}
		product.ShortDescription = strings.TrimSpace(short)
		draft.Findings = append(draft.Findings, strippedFindings("the short description", stripped)...)
	}

	for _, wanted := range brief.Product.Specifications {
		value := specificationValue(answer.Specifications, wanted.Name)
		if value == "" {
			draft.Findings = append(draft.Findings, Finding{
				Severity: SeverityWarn, Code: CodeSpecificationMissing,
				Message: "the writer stated no value for " + wanted.Name + ", so the product does not get it",
				Details: map[string]any{"specification": wanted.Name},
			})
			continue
		}
		product.Specifications = append(product.Specifications, Specification{Name: wanted.Name, Value: value})
	}
	draft.Product = &product
	return draft, doc, nil
}

func specificationValue(answered []AnswerSpecification, name string) string {
	for _, specification := range answered {
		if !strings.EqualFold(strings.TrimSpace(specification.Name), name) {
			continue
		}
		return plainText(specification.Value)
	}
	return ""
}

func plainText(value string) string {
	doc, err := Parse(value)
	if err != nil {
		return strings.TrimSpace(value)
	}
	return strings.Join(strings.Fields(doc.Text()), " ")
}

func ProductFindings(draft ProductDraft, product template.Product, primary string) []Finding {
	out := make([]Finding, 0)
	short := product.ShortDescription
	if !short.Enabled || !short.PrimaryKeyword || primary == "" {
		return out
	}
	if !carries(plainText(draft.ShortDescription), primary) {
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
