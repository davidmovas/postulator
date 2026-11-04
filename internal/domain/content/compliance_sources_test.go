package content_test

import (
	"slices"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/template"
)

func TestComplianceGradesTheLinksToTheResearchsSources(t *testing.T) {
	t.Parallel()

	lc := contextOf(target("/drinks/", []string{"drinks"}, content.RelationUp, false))
	whoLink := `<a href="` + whoPeptides + `">frozen storage</a>`
	waterLink := `<a href="` + whoWater + `">sterile water</a>`
	cdcLink := `<a href="` + cdcLabels + `">labels</a>`

	cases := []struct {
		name    string
		body    string
		sources content.Sources
		codes   []string
		details map[string]any
	}{
		{
			name: "cited sources within the template's rules", body: `<p>` + whoLink + ` and ` + cdcLink + `.</p>`,
			sources: sourcesFor(template.SourceLinks{Max: 2, PerDomain: 1, Min: 2}), codes: []string{"target_missing"},
		},
		{
			name: "an address the research did not cite", body: `<p>See <a href="https://example.com/">elsewhere</a>.</p>`,
			sources: sourcesFor(template.SourceLinks{Max: 2}), codes: []string{"target_missing", content.CodeExternalLink},
		},
		{
			name: "a cited source under a template that links none", body: `<p>` + whoLink + `.</p>`,
			sources: sourcesFor(template.SourceLinks{}), codes: []string{"target_missing", content.CodeExternalLink},
		},
		{
			name: "more sources than the template's most", body: `<p>` + whoLink + `, ` + waterLink + ` and ` + cdcLink + `.</p>`,
			sources: sourcesFor(template.SourceLinks{Max: 2}), codes: []string{"target_missing", content.CodeTooManySourceLinks},
			details: map[string]any{"linked": 3, "max": 2, "perDomain": 0},
		},
		{
			name: "more links to one site than the template allows", body: `<p>` + whoLink + ` and ` + waterLink + `.</p>`,
			sources: sourcesFor(template.SourceLinks{Max: 3, PerDomain: 1}), codes: []string{"target_missing", content.CodeTooManySourceLinks},
			details: map[string]any{"linked": 2, "max": 3, "perDomain": 1},
		},
		{
			name: "fewer sources than the template asks of a researched page", body: `<p>` + whoLink + `.</p>`,
			sources: sourcesFor(template.SourceLinks{Max: 3, Min: 2}), codes: []string{"target_missing", content.CodeTooFewSourceLinks},
			details: map[string]any{"linked": 1, "min": 2, "found": 3},
		},
		{
			name: "a page nobody researched owes no source", body: `<p>Nothing to cite.</p>`,
			sources: content.SourcesOf(nil, template.SourceLinks{Max: 3, Min: 2}), codes: []string{"target_missing"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			report := content.Compliance(mustParse(t, tc.body), lc, field.NoLinks(), tc.sources, "self")
			if got := codesOf(report); !slices.Equal(got, tc.codes) {
				t.Fatalf("codes = %v, want %v", got, tc.codes)
			}
			if tc.details == nil {
				return
			}
			got := detailsOf(report, tc.codes[len(tc.codes)-1])
			for key, want := range tc.details {
				if got[key] != want {
					t.Errorf("details[%s] = %v, want %v (%v)", key, got[key], want, got)
				}
			}
		})
	}
}

func TestTheLinksToSourcesAreListedWithTheirWords(t *testing.T) {
	t.Parallel()

	lc := contextOf(target("/drinks/", []string{"drinks"}, content.RelationUp, false))
	body := `<p><a href="` + whoPeptides + `?utm_source=openai">frozen storage</a>, <a href="/drinks/">drinks</a>, ` +
		`<a href="https://example.com/">elsewhere</a> and <a href="` + cdcLabels + `">labels</a>.</p>`

	got := sourcesFor(template.SourceLinks{Max: 3}).Linked(mustParse(t, body), lc)
	want := []content.SourceLink{{Href: whoPeptides, Anchor: "frozen storage"}, {Href: cdcLabels, Anchor: "labels"}}
	if !slices.Equal(got, want) {
		t.Errorf("Linked = %+v, want %+v", got, want)
	}
	if none := content.SourcesOf(nil, template.SourceLinks{Max: 3}).Linked(mustParse(t, body), lc); len(none) != 0 {
		t.Errorf("a page nobody researched lists %+v", none)
	}
}

func TestTheClientsOwnExternalLinksNeverCountAsSources(t *testing.T) {
	t.Parallel()

	lc := contextOf(target("/drinks/", []string{"drinks"}, content.RelationUp, false))
	theirs := `<p>His supplier is <a href="` + whoPeptides + `">frozen storage</a> and <a href="https://example.com/">elsewhere</a>.</p>`
	ours := `<p>Espresso is brewed under pressure.</p>`

	report := content.ComplianceOf(mustParse(t, theirs+ours), mustParse(t, ours), lc, field.NoLinks(),
		sourcesFor(template.SourceLinks{Max: 2, Min: 1}), "self")
	if hasCode(report, content.CodeExternalLink) || !hasCode(report, content.CodeTooFewSourceLinks) {
		t.Errorf("codes = %v, want his links ungraded and uncounted", codesOf(report))
	}
}
