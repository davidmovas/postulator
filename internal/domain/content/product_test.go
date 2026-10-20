package content_test

import (
	"slices"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/keyword"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/template"
)

func productSpec() template.TemplateSpec {
	return template.TemplateSpec{
		Sections:     []template.Section{{Heading: "Overview", Required: true}},
		KeywordRules: template.KeywordRules{PrimaryInH1: true, PrimaryInFirstParagraph: true},
		Product: &template.Product{
			ShortDescription: template.ProductShortDescription{
				Enabled: true, Intent: "Say what it is", TargetWords: 40, PrimaryKeyword: true,
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
	return content.NewBrief(productSpec(), field.LinkRules{}, page, entity, content.LinkContext{})
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
}

func TestNewBriefKeepsAPagesOwnH1(t *testing.T) {
	t.Parallel()

	page := storeProductPage()
	page.WPType = pagemap.WPPage
	if got := productBrief(page).H1; got != "BPC-157 Liquid Form" {
		t.Errorf("h1 = %q, want the page's own plan", got)
	}

	unnamed := storeProductPage()
	unnamed.Observed.Title = ""
	if got := productBrief(unnamed).H1; got != "BPC-157 Liquid Form" {
		t.Errorf("h1 = %q, want the file's H1 for a product the store has not named yet", got)
	}
}

func TestTheTemplateBriefsTheShortDescription(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		product *template.Product
		writes  bool
		rules   []string
	}{
		{name: "no product outputs", product: nil},
		{name: "switched off", product: &template.Product{ShortDescription: template.ProductShortDescription{Intent: "Say what it is"}}},
		{
			name:    "every rule",
			product: &template.Product{ShortDescription: template.ProductShortDescription{Enabled: true, Intent: " Say what it is ", TargetWords: 40, PrimaryKeyword: true}},
			writes:  true,
			rules:   []string{"The template asks: Say what it is", "About 40 words", "It carries the primary keyword"},
		},
		{
			name:    "the intent alone",
			product: &template.Product{ShortDescription: template.ProductShortDescription{Enabled: true}},
			writes:  true,
			rules:   []string{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := content.WritesShortDescription(tc.product); got != tc.writes {
				t.Fatalf("writes = %t, want %t", got, tc.writes)
			}
			if !tc.writes {
				return
			}
			if got := content.ShortDescriptionRules(*tc.product); !slices.Equal(got, tc.rules) {
				t.Errorf("rules = %q, want %q", got, tc.rules)
			}
		})
	}
}

func TestTheShortDescriptionIsWrittenAsEscapedParagraphs(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		plain string
		want  string
	}{
		{name: "one paragraph", plain: " A bpc 157 liquid for research. ", want: "<p>A bpc 157 liquid for research.</p>"},
		{name: "two paragraphs", plain: "First.\n\n  Second.\n", want: "<p>First.</p>\n<p>Second.</p>"},
		{name: "a line break is a space", plain: "First line\nsecond  line", want: "<p>First line second line</p>"},
		{name: "markup is text", plain: `Fits <b>5 & 7</b> "kits"`, want: `<p>Fits &lt;b&gt;5 &amp; 7&lt;/b&gt; "kits"</p>`},
		{name: "nothing", plain: " \n\n ", want: ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := content.PlainParagraphs(tc.plain); got != tc.want {
				t.Errorf("PlainParagraphs(%q) = %q, want %q", tc.plain, got, tc.want)
			}
		})
	}
}

func TestShortDescriptionFindingsGradeTheShortDescription(t *testing.T) {
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
			found := content.ShortDescriptionFindings(tc.short, product, "bpc 157 liquid")
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
