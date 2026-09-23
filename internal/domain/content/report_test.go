package content_test

import (
	"testing"

	"github.com/davidmovas/postulator/internal/domain/content"
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
		name      string
		body      string
		primary   string
		secondary []string
		codes     []string
	}{
		{
			name: "a compliant body",
			body: "<h1>Coffee guide</h1><p>This coffee guide explains the basics of brewing at home today.</p>" +
				"<h2>Beans</h2><p>Pick a roast that suits your grinder and your palate.</p>" +
				"<h2>Brewing</h2><p>Use water just off the boil for the best extraction.</p>",
			primary:   "coffee",
			secondary: []string{"roast"},
			codes:     []string{},
		},
		{
			name: "the primary keyword is missing from the h1 and the lead",
			body: "<h1>Guide</h1><p>This explains the basics of brewing at home today for everyone.</p>" +
				"<h2>Beans</h2><p>Pick a roast that suits your grinder and your palate.</p>" +
				"<h2>Brewing</h2><p>Use water just off the boil for the best extraction here.</p>",
			primary: "coffee",
			codes:   []string{content.CodePrimaryMissingInH1, content.CodePrimaryMissingInLead},
		},
		{
			name: "a required section is missing",
			body: "<h1>Coffee guide</h1><p>This coffee guide explains the basics of brewing at home today.</p>" +
				"<h2>Beans</h2><p>Pick a roast that suits your grinder and your palate for sure.</p>",
			primary: "coffee",
			codes:   []string{content.CodeSectionMissing},
		},
		{
			name:    "a short body with an empty heading",
			body:    "<h1>Coffee</h1><h2></h2><p>Coffee.</p><h2>Beans</h2><h2>Brewing</h2>",
			primary: "coffee",
			codes:   []string{content.CodeEmptyHeading, content.CodeWordCount, content.CodeKeywordDensity},
		},
		{
			name: "a secondary keyword is missing",
			body: "<h1>Coffee guide</h1><p>This coffee guide explains the basics of brewing at home today.</p>" +
				"<h2>Beans</h2><p>Pick a bean that suits your grinder and your palate for sure.</p>" +
				"<h2>Brewing</h2><p>Use water just off the boil for the best extraction here.</p>",
			primary:   "coffee",
			secondary: []string{"roast", "  "},
			codes:     []string{content.CodeSecondaryMissing},
		},
		{
			name:    "the keyword is repeated too often",
			body:    "<h1>Coffee</h1><p>Coffee coffee coffee coffee.</p><h2>Beans</h2><h2>Brewing</h2>",
			primary: "coffee",
			codes:   []string{content.CodeKeywordDensity, content.CodeWordCount},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			report := content.Structure(mustParse(t, tc.body), tc.primary, tc.secondary, spec)
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

func TestStructureOnAnEmptyBody(t *testing.T) {
	t.Parallel()

	report := content.Structure(mustParse(t, "<p>Nothing at all.</p>"), "", nil, template.TemplateSpec{})
	if !hasCode(report, content.CodeNoHeadings) || len(report.Items) != 1 {
		t.Fatalf("findings = %v", codesOf(report))
	}
	if !report.HasErrors() || report.Score != 0.75 {
		t.Fatalf("report = %+v", report)
	}

	long := content.Structure(mustParse(t, "<h1>x</h1><p>one two three</p>"), "", nil,
		template.TemplateSpec{Length: template.Length{Max: 2}})
	if !hasCode(long, content.CodeWordCount) {
		t.Fatalf("findings = %v", codesOf(long))
	}
}
