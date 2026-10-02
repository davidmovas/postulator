package content_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/keyword"
	"github.com/davidmovas/postulator/internal/domain/template"
)

func codesOf(report content.Report) []string {
	out := make([]string, 0, len(report.Items))
	for _, item := range report.Items {
		out = append(out, item.Code)
	}
	return out
}

func detailsOf(report content.Report, code string) map[string]any {
	for _, item := range report.Items {
		if item.Code == code {
			return item.Details
		}
	}
	return nil
}

func hasCode(report content.Report, code string) bool {
	for _, item := range report.Items {
		if item.Code == code {
			return true
		}
	}
	return false
}

func TestComplianceOnHandWrittenBodies(t *testing.T) {
	t.Parallel()

	parent := target("/drinks/", []string{"drinks"}, content.RelationUp, true)
	child := target("/coffee/beans/", []string{"beans"}, content.RelationDown, false)
	lc := contextOf(parent, child)

	cases := []struct {
		name  string
		body  string
		codes []string
		score float64
	}{
		{
			name: "a compliant body scores one",
			body: `<p>Part of our <a href="/drinks/">drinks</a> range.</p>` +
				`<p>Read about <a href="/coffee/beans/">beans</a>.</p>`,
			codes: []string{},
			score: 1,
		},
		{
			name:  "a missing required target is an error",
			body:  `<p>Read about <a href="/coffee/beans/">beans</a>.</p>`,
			codes: []string{content.CodeTargetMissing},
			score: 0.75,
		},
		{
			name:  "a missing optional target is a warning",
			body:  `<p>Part of our <a href="/drinks/">drinks</a> range.</p>`,
			codes: []string{content.CodeTargetMissing},
			score: 0.95,
		},
		{
			name: "an external link is an error",
			body: `<p>Part of our <a href="/drinks/">drinks</a> range and ` +
				`<a href="https://example.com/">elsewhere</a>.</p>` +
				`<p>Read about <a href="/coffee/beans/">beans</a>.</p>`,
			codes: []string{content.CodeExternalLink},
			score: 0.75,
		},
		{
			name: "a self link is an error",
			body: `<p>Part of our <a href="/drinks/">drinks</a> range and <a href="/self/">this page</a>.</p>` +
				`<p>Read about <a href="/coffee/beans/">beans</a>.</p>`,
			codes: []string{content.CodeSelfLink},
			score: 0.75,
		},
		{
			name: "an internal link outside the graph is a warning",
			body: `<p>Part of our <a href="/drinks/">drinks</a> range and <a href="/legal/">the terms</a>.</p>` +
				`<p>Read about <a href="/coffee/beans/">beans</a>.</p>`,
			codes: []string{content.CodeUnknownInternal},
			score: 0.95,
		},
		{
			name: "an anchor outside the whitelist is a warning",
			body: `<p>Part of our <a href="/drinks/">beverages</a> range.</p>` +
				`<p>Read about <a href="/coffee/beans/">beans</a>.</p>`,
			codes: []string{content.CodeAnchorNotAllowed},
			score: 0.95,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			doc := mustParse(t, tc.body)
			report := content.Compliance(doc, lc, policy(template.LinkRules{}), "self")

			if len(report.Items) != len(tc.codes) {
				t.Fatalf("findings = %v, want %v", codesOf(report), tc.codes)
			}
			for _, code := range tc.codes {
				if !hasCode(report, code) {
					t.Fatalf("findings = %v, want %s", codesOf(report), code)
				}
			}
			if report.Score < tc.score-0.001 || report.Score > tc.score+0.001 {
				t.Fatalf("Score = %v, want %v", report.Score, tc.score)
			}
		})
	}
}

func TestUnpublishedTargetsAreNamed(t *testing.T) {
	t.Parallel()

	parent := target("/drinks/", []string{"drinks"}, content.RelationUp, true)
	child := target("/coffee/beans/", []string{"beans"}, content.RelationDown, false)
	sibling := target("/coffee/filter/", []string{"filter coffee"}, content.RelationSibling, false)
	lc := contextOf(parent, child, sibling)
	doc := mustParse(t, `<p>Part of our <a href="/drinks/">drinks</a> range.</p>`+
		`<p>Read about <a href="/coffee/beans/">beans</a>.</p>`)

	cases := []struct {
		name string
		live map[string]bool
		want []string
	}{
		{name: "every linked page is on the site", live: map[string]bool{parent.PageID: true, child.PageID: true}},
		{name: "a linked child is not", live: map[string]bool{parent.PageID: true}, want: []string{child.URL}},
		{name: "nothing is", live: map[string]bool{}, want: []string{parent.URL, child.URL}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			found := content.Unpublished(doc, lc, tc.live, "self")
			if len(found) != len(tc.want) {
				t.Fatalf("findings = %+v, want %v", found, tc.want)
			}
			for i, finding := range found {
				if finding.Code != content.CodeTargetNotPublished || finding.Severity != content.SeverityWarn {
					t.Fatalf("finding = %+v, want a target_not_published warning", finding)
				}
				if finding.Details["targetPageId"] != "p-"+tc.want[i] || finding.Details["pageId"] != "self" {
					t.Fatalf("details = %+v, want the page and the target named", finding.Details)
				}
				if !strings.Contains(finding.Message, tc.want[i]) {
					t.Fatalf("message = %q, want %s named", finding.Message, tc.want[i])
				}
			}
		})
	}
}

func TestCompliancePolicySwitchesAndCaps(t *testing.T) {
	t.Parallel()

	lc := contextOf(target("/drinks/", []string{"drinks"}, content.RelationUp, true))
	body := `<p>Our <a href="/drinks/">drinks</a>, <a href="/self/">this page</a> and ` +
		`<a href="https://example.com/">elsewhere</a>.</p>`
	doc := mustParse(t, body)

	permissive := template.LinkPolicy{Rules: template.LinkRules{MaxLinks: 2}}
	report := content.Compliance(doc, lc, permissive, "self")
	if hasCode(report, content.CodeSelfLink) || hasCode(report, content.CodeExternalLink) {
		t.Fatalf("a permissive policy allows both: %v", codesOf(report))
	}
	if report.HasErrors() {
		t.Fatalf("a permissive policy reports no error: %v", codesOf(report))
	}

	strict := content.Compliance(doc, lc, policy(template.LinkRules{}), "self")
	if !strict.HasErrors() {
		t.Fatalf("a strict policy reports errors: %v", codesOf(strict))
	}
}

func TestTheLinkCapCountsGraphLinksOnBothSides(t *testing.T) {
	t.Parallel()

	lc := contextOf(
		target("/a/", []string{"alpha"}, content.RelationDown, false),
		target("/b/", []string{"bravo"}, content.RelationDown, false),
		target("/c/", []string{"charlie"}, content.RelationDown, false),
	)
	rules := template.LinkRules{MaxLinks: 2}

	cases := []struct {
		name    string
		body    string
		graph   int
		capped  bool
		outcome content.Outcome
	}{
		{
			name: "off-graph links do not spend the budget",
			body: `<p>An <a href="/a/">alpha</a>, <a href="/legal/">the terms</a>, ` +
				`<a href="https://example.com/">elsewhere</a> and <a href="#top">the top</a>. ` +
				`A charlie waits.</p>`,
			graph: 1, outcome: content.OutcomeInserted,
		},
		{
			name:  "a spent budget stops the next target",
			body:  `<p>An <a href="/a/">alpha</a> and a <a href="/b/">bravo</a>. A charlie waits.</p>`,
			graph: 2, outcome: content.OutcomeCapReached,
		},
		{
			name: "a body over the cap is reported",
			body: `<p>An <a href="/a/">alpha</a>, a <a href="/b/">bravo</a> and a ` +
				`<a href="/c/">charlie</a>.</p>`,
			graph: 3, capped: true, outcome: content.OutcomeAlreadyLinked,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			report := content.Compliance(mustParse(t, tc.body), lc, policy(rules), "self")
			if hasCode(report, content.CodeTooManyLinks) != tc.capped {
				t.Fatalf("the cap finding = %v, want capped=%t", codesOf(report), tc.capped)
			}
			if tc.capped {
				details := detailsOf(report, content.CodeTooManyLinks)
				if details["links"] != tc.graph || details["counted"] != content.CountedGraphLinks {
					t.Fatalf("the cap names %+v, want %d %s", details, tc.graph, content.CountedGraphLinks)
				}
			}

			result := content.InsertLinks(mustParse(t, tc.body), lc, policy(rules))
			last := result.Decisions[len(result.Decisions)-1]
			if last.Target.URL != "/c/" || last.Outcome != tc.outcome {
				t.Fatalf("the last decision = %+v, want /c/ %s", last, tc.outcome)
			}
		})
	}
}

func TestComplianceFloorsTheScore(t *testing.T) {
	t.Parallel()

	targets := make([]content.LinkTarget, 0, 6)
	for _, path := range []string{"/a/", "/b/", "/c/", "/d/", "/e/", "/f/"} {
		targets = append(targets, target(path, []string{"anchor"}, content.RelationUp, true))
	}

	report := content.Compliance(mustParse(t, "<p>Nothing.</p>"), contextOf(targets...),
		policy(template.LinkRules{}), "self")
	if report.Score != 0 {
		t.Fatalf("Score = %v, want the floor", report.Score)
	}
}

func TestStructureChecksTheTemplateRules(t *testing.T) {
	t.Parallel()

	spec := template.TemplateSpec{
		Sections: []template.Section{
			{Heading: "Beans", Required: true},
			{Heading: "Brewing", Required: true},
			{Heading: "Extras"},
		},
		Length:       template.Length{Min: 10, Max: 40},
		KeywordRules: template.KeywordRules{PrimaryInH1: true, PrimaryInFirstParagraph: true, MaxDensity: 0.2},
	}

	cases := []struct {
		name     string
		body     string
		keywords keyword.List
		codes    []string
	}{
		{
			name: "a compliant body",
			body: "<h1>Coffee guide</h1><p>This coffee guide explains the basics of brewing at home today.</p>" +
				"<h2>Beans</h2><p>Pick a roast that suits your grinder and your palate.</p>" +
				"<h2>Brewing</h2><p>Use water just off the boil for the best extraction.</p>",
			keywords: keyword.Of("coffee", "roast"),
			codes:    []string{},
		},
		{
			name: "the main keyword is nowhere on the page",
			body: "<h1>Guide</h1><p>This explains the basics of brewing at home today for everyone.</p>" +
				"<h2>Beans</h2><p>Pick a roast that suits your grinder and your palate.</p>" +
				"<h2>Brewing</h2><p>Use water just off the boil for the best extraction here.</p>",
			keywords: keyword.Of("coffee"),
			codes:    []string{content.CodePrimaryMissingInH1, content.CodePrimaryMissingInLead, content.CodeKeywordsMissing},
		},
		{
			name: "the main keyword is in the body and not where the template wants it",
			body: "<h1>Guide</h1><p>This explains the basics of brewing at home today for everyone.</p>" +
				"<h2>Beans</h2><p>Pick a coffee that suits your grinder and your palate.</p>" +
				"<h2>Brewing</h2><p>Use water just off the boil for the best extraction here.</p>",
			keywords: keyword.Of("coffee"),
			codes:    []string{content.CodePrimaryMissingInH1, content.CodePrimaryMissingInLead},
		},
		{
			name: "a required section is missing",
			body: "<h1>Coffee guide</h1><p>This coffee guide explains the basics of brewing at home today.</p>" +
				"<h2>Beans</h2><p>Pick a roast that suits your grinder and your palate for sure.</p>",
			keywords: keyword.Of("coffee"),
			codes:    []string{content.CodeSectionMissing},
		},
		{
			name:     "a short body with an empty heading",
			body:     "<h1>Coffee</h1><h2></h2><p>Coffee.</p><h2>Beans</h2><h2>Brewing</h2>",
			keywords: keyword.Of("coffee"),
			codes:    []string{content.CodeEmptyHeading, content.CodeWordCount, content.CodeKeywordDensity},
		},
		{
			name: "the keywords the body does not use are one finding",
			body: "<h1>Coffee guide</h1><p>This coffee guide explains the basics of brewing at home today.</p>" +
				"<h2>Beans</h2><p>Pick a bean that suits your grinder and your palate for sure.</p>" +
				"<h2>Brewing</h2><p>Use water just off the boil for the best extraction here.</p>",
			keywords: keyword.Of("coffee", "roast", "grinder", "arabica beans"),
			codes:    []string{content.CodeKeywordsMissing},
		},
		{
			name:     "the keyword is repeated too often",
			body:     "<h1>Coffee</h1><p>Coffee coffee coffee coffee.</p><h2>Beans</h2><h2>Brewing</h2>",
			keywords: keyword.Of("coffee"),
			codes:    []string{content.CodeKeywordDensity, content.CodeWordCount},
		},
		{
			name: "a page without keywords raises no keyword finding",
			body: "<h1>Guide</h1><p>This explains the basics of brewing at home today for everyone.</p>" +
				"<h2>Beans</h2><p>Pick a roast that suits your grinder and your palate.</p>" +
				"<h2>Brewing</h2><p>Use water just off the boil for the best extraction here.</p>",
			keywords: keyword.Of(),
			codes:    []string{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			report := content.Structure(mustParse(t, tc.body), tc.keywords, spec)
			if len(report.Items) != len(tc.codes) {
				t.Fatalf("findings = %v, want %v", codesOf(report), tc.codes)
			}
			for _, code := range tc.codes {
				if !hasCode(report, code) {
					t.Fatalf("findings = %v, want %s", codesOf(report), code)
				}
			}
		})
	}
}

func TestStructureNamesTheMissingKeywordsMostImportantFirst(t *testing.T) {
	t.Parallel()

	body := "<h1>Coffee guide</h1><p>This coffee guide explains the basics of brewing at home today.</p>" +
		"<h2>Beans</h2><p>Pick a bean that suits your grinder and your palate for sure.</p>"
	keywords := keyword.New([]keyword.Keyword{
		{Text: "arabica beans"}, {Text: "coffee", Volume: new(9000)}, {Text: "roast", Volume: new(40)},
		{Text: "grinder", Volume: new(700)}, {Text: "espresso", Volume: new(2000)},
	})

	report := content.Structure(mustParse(t, body), keywords, template.TemplateSpec{})
	missing, ok := detailsOf(report, content.CodeKeywordsMissing)["keywords"].([]string)
	if !ok || !slices.Equal(missing, []string{"espresso", "roast", "arabica beans"}) {
		t.Fatalf("missing keywords = %v, want them in the order of the list", detailsOf(report, content.CodeKeywordsMissing))
	}

	var finding content.Finding
	for _, item := range report.Items {
		if item.Code == content.CodeKeywordsMissing {
			finding = item
		}
	}
	if finding.Severity != content.SeverityWarn {
		t.Fatalf("severity = %q, want a warning", finding.Severity)
	}
	if want := "the body does not use 3 of the 5 keywords of the page: espresso, roast, arabica beans"; finding.Message != want {
		t.Fatalf("message = %q, want %q", finding.Message, want)
	}
}

func TestStructureAsksOnlyForTheKeywordsTheTemplateRequires(t *testing.T) {
	t.Parallel()

	body := "<h1>Coffee guide</h1><p>This coffee guide explains the basics of brewing at home today.</p>" +
		"<h2>Beans</h2><p>Pick a bean that suits your grinder and your palate for sure.</p>"
	keywords := keyword.Of("coffee", "espresso", "grinder", "roast")

	cases := []struct {
		name     string
		required *int
		missing  []string
	}{
		{name: "a template that does not say asks for them all", required: nil, missing: []string{"espresso", "roast"}},
		{name: "the first two", required: new(2), missing: []string{"espresso"}},
		{name: "the first alone, which the body uses", required: new(1), missing: nil},
		{name: "none", required: new(0), missing: nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			spec := template.TemplateSpec{KeywordRules: template.KeywordRules{RequiredKeywords: tc.required}}
			report := content.Structure(mustParse(t, body), keywords, spec)
			if tc.missing == nil {
				if hasCode(report, content.CodeKeywordsMissing) {
					t.Fatalf("findings = %v, want no missing keyword", codesOf(report))
				}
				return
			}
			missing, ok := detailsOf(report, content.CodeKeywordsMissing)["keywords"].([]string)
			if !ok || !slices.Equal(missing, tc.missing) {
				t.Fatalf("missing = %v, want %v", detailsOf(report, content.CodeKeywordsMissing), tc.missing)
			}
		})
	}
}

func TestStructureOnAnEmptyBody(t *testing.T) {
	t.Parallel()

	report := content.Structure(mustParse(t, "<p>Nothing at all.</p>"), nil, template.TemplateSpec{})
	if !hasCode(report, content.CodeNoHeadings) || len(report.Items) != 1 {
		t.Fatalf("findings = %v", codesOf(report))
	}
	if !report.HasErrors() || report.Score != 0.75 {
		t.Fatalf("report = %+v", report)
	}

	long := content.Structure(mustParse(t, "<h1>x</h1><p>one two three</p>"), nil,
		template.TemplateSpec{Length: template.Length{Max: 2}})
	if !hasCode(long, content.CodeWordCount) {
		t.Fatalf("findings = %v", codesOf(long))
	}
}
