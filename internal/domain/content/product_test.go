package content_test

import (
	"slices"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/keyword"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func productSpec() template.TemplateSpec {
	return template.TemplateSpec{
		Sections:     []template.Section{{Heading: "Overview", Required: true}},
		KeywordRules: template.KeywordRules{PrimaryInH1: true, PrimaryInFirstParagraph: true},
		Product: &template.Product{
			ShortDescription: template.ProductShortDescription{
				Enabled: true, Intent: "Say what it is", TargetWords: 40, PrimaryKeyword: true,
			},
			Specifications: []template.ProductSpecification{
				{Name: "Form", Intent: "As the name says"},
				{Name: "Size", Intent: "As the notes say"},
			},
		},
	}
}

func storeProductPage() pagemap.Page {
	return pagemap.Page{
		Path: "/product/bpc-157-liquid/", WPType: pagemap.WPProduct, Title: "Buy BPC-157 Liquid", H1: "BPC-157 Liquid Form",
		Observed: pagemap.Observed{Title: "BPC-157 Liquid &amp; Spray"},
	}
}

func productBrief(page pagemap.Page) content.Brief {
	entity := graph.Entity{Name: "BPC-157 Liquid", Keywords: keyword.Of("bpc 157 liquid")}
	return content.NewBrief(productSpec(), template.LinkRules{}, page, entity, content.LinkContext{})
}

func TestNewBriefWritesAProductUnderTheNameTheStoreGivesIt(t *testing.T) {
	t.Parallel()

	brief := productBrief(storeProductPage())
	if !brief.PlannedH1 || brief.H1 != "BPC-157 Liquid & Spray" {
		t.Fatalf("h1 = %q (planned %t), want the store's name as it reads", brief.H1, brief.PlannedH1)
	}
	if brief.Title != "Buy BPC-157 Liquid" {
		t.Errorf("title = %q, want the file's title", brief.Title)
	}
	if brief.Product == nil {
		t.Fatal("a product written from a template with product outputs carries no product brief")
	}
	if !brief.Product.ShortDescription || brief.Product.ShortWords != 40 || !brief.Product.ShortKeyword {
		t.Errorf("product brief = %+v", brief.Product)
	}
	names := make([]string, 0, len(brief.Product.Specifications))
	for _, specification := range brief.Product.Specifications {
		names = append(names, specification.Name)
	}
	if !slices.Equal(names, []string{"Form", "Size"}) {
		t.Errorf("specifications = %v", names)
	}
}

func TestNewBriefLeavesTheProductOutputsOffAPage(t *testing.T) {
	t.Parallel()

	page := storeProductPage()
	page.WPType = pagemap.WPPage
	brief := productBrief(page)
	if brief.Product != nil {
		t.Errorf("a page got a product brief: %+v", brief.Product)
	}
	if brief.H1 != "BPC-157 Liquid Form" {
		t.Errorf("h1 = %q, want the page's own plan", brief.H1)
	}

	unnamed := storeProductPage()
	unnamed.Observed.Title = ""
	if got := productBrief(unnamed).H1; got != "BPC-157 Liquid Form" {
		t.Errorf("h1 = %q, want the file's H1 for a product the store has not named yet", got)
	}
}

func productAnswer() content.ProductAnswer {
	return content.ProductAnswer{
		DraftAnswer: content.DraftAnswer{
			H1:       "BPC-157 Liquid & Spray",
			Sections: []content.AnswerSection{{Slot: 1, Heading: "Overview", HTML: "<p>bpc 157 liquid in a vial.</p>"}},
		},
		ShortDescription: `<p>A <strong>bpc 157 liquid</strong> <a href="/x/">for research</a>.</p>`,
		Specifications: []content.AnswerSpecification{
			{Name: "form", Value: " <b>Liquid</b> "},
			{Name: "Shade", Value: "Clear"},
		},
	}
}

func TestAssembleProductKeepsTheBriefsSpecificationsAndSaysWhichCameBackEmpty(t *testing.T) {
	t.Parallel()

	brief := productBrief(storeProductPage())
	draft, doc, err := content.AssembleProduct(productAnswer(), brief)
	if err != nil {
		t.Fatalf("AssembleProduct: %v", err)
	}
	if doc == nil || draft.Product == nil {
		t.Fatalf("draft = %+v", draft)
	}
	if draft.Product.ShortDescription != `<p>A <strong>bpc 157 liquid</strong> for research.</p>` {
		t.Errorf("short description = %q, want it sanitized like a section", draft.Product.ShortDescription)
	}
	want := []content.Specification{{Name: "Form", Value: "Liquid"}}
	if !slices.Equal(draft.Product.Specifications, want) {
		t.Errorf("specifications = %+v, want %+v", draft.Product.Specifications, want)
	}
	if !slices.Contains(codesIn(draft.Findings), content.CodeSpecificationMissing) {
		t.Errorf("findings = %v, want the empty Size named", codesIn(draft.Findings))
	}
}

func TestAssembleProductAsksAgainForAShortDescriptionTheTemplateWants(t *testing.T) {
	t.Parallel()

	answer := productAnswer()
	answer.ShortDescription = "  "
	_, _, err := content.AssembleProduct(answer, productBrief(storeProductPage()))
	if !errors.IsCode(err, errors.External) || reasonOf(t, err) != content.ReasonIncompleteAnswer {
		t.Fatalf("err = %v, want an incomplete answer the engine tries again", err)
	}

	page := storeProductPage()
	page.WPType = pagemap.WPPage
	draft, _, err := content.AssembleProduct(answer, productBrief(page))
	if err != nil || draft.Product != nil {
		t.Errorf("a page's draft = %+v, %v; it carries no product outputs", draft.Product, err)
	}
}

func TestProductFindingsGradeTheShortDescription(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		short string
		rule  bool
		want  []string
	}{
		{name: "the keyword is there", short: "<p>Our bpc 157 liquid.</p>", rule: true, want: []string{}},
		{name: "the keyword is missing", short: "<p>Our vial.</p>", rule: true, want: []string{content.CodePrimaryMissingInShortDescription}},
		{name: "the rule is off", short: "<p>Our vial.</p>", want: []string{}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			product := template.Product{ShortDescription: template.ProductShortDescription{Enabled: true, PrimaryKeyword: tc.rule}}
			found := content.ProductFindings(content.ProductDraft{ShortDescription: tc.short}, product, "bpc 157 liquid")
			if got := codesIn(found); !slices.Equal(got, tc.want) {
				t.Errorf("codes = %v, want %v", got, tc.want)
			}
			for _, finding := range found {
				if finding.Severity != content.SeverityError {
					t.Errorf("finding %s is %s, want an error like the other keyword rules", finding.Code, finding.Severity)
				}
			}
		})
	}
}

func TestForStoreTurnsTheH1RuleIntoAWarningAboutTheName(t *testing.T) {
	t.Parallel()

	doc, err := content.Parse("<h1>Vial</h1><h2>Overview</h2><p>bpc 157 liquid.</p>")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	report := content.Structure(doc, keyword.Of("bpc 157 liquid"), productSpec())
	if !slices.Contains(codesIn(report.Items), content.CodePrimaryMissingInH1) {
		t.Fatalf("codes = %v, want the H1 rule broken to start with", codesIn(report.Items))
	}
	report.Items = append(report.Items, content.Finding{Severity: content.SeverityWarn, Code: content.CodePlanH1LacksKeyword})

	graded := content.ForStore(report)
	if slices.Contains(codesIn(graded.Items), content.CodePrimaryMissingInH1) {
		t.Fatalf("codes = %v, want no H1 error on a product", codesIn(graded.Items))
	}
	if slices.Contains(codesIn(graded.Items), content.CodePlanH1LacksKeyword) {
		t.Errorf("codes = %v, want the planned H1 finding folded into the one about the name", codesIn(graded.Items))
	}
	for _, finding := range graded.Items {
		if finding.Code == content.CodePrimaryMissingInName && finding.Severity != content.SeverityWarn {
			t.Errorf("the name finding is %s, want a warning", finding.Severity)
		}
	}
	if !slices.Contains(codesIn(graded.Items), content.CodePrimaryMissingInName) {
		t.Errorf("codes = %v, want the name named", codesIn(graded.Items))
	}
	if graded.Score < report.Score {
		t.Errorf("score = %v, want no worse than %v once the error is a warning", graded.Score, report.Score)
	}
}
