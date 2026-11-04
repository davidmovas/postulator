package content_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/template"
)

const (
	whoPeptides = "https://www.who.int/peptides"
	whoWater    = "https://www.who.int/water"
	cdcLabels   = "https://www.cdc.gov/labels"
)

func researchedPage() *content.Research {
	return &content.Research{
		Text: "- Store peptides frozen. ([who.int](" + whoPeptides + "?utm_source=openai)) [who.int]\n" +
			"- Use sterile water. ([who.int](" + whoWater + "?utm_source=openai)) [who.int]\n" +
			"- Label every vial. ([cdc.gov](" + cdcLabels + "?utm_source=openai)) [cdc.gov]\n" +
			"- Volume is mass over concentration. [calculator]",
		Sources: []content.Source{
			{URL: whoPeptides + "?utm_source=openai"}, {URL: whoWater + "?utm_source=openai"},
			{URL: cdcLabels + "?utm_source=openai"}, {URL: "https://www.nih.gov/uncited"},
		},
	}
}

func sourcesFor(rules template.SourceLinks) content.Sources {
	return content.SourcesOf(researchedPage(), rules)
}

func TestSanitizeKeepsTheLinksToTheResearchsSourcesItsTemplateAllows(t *testing.T) {
	t.Parallel()

	const fragment = `<p><a href="` + whoPeptides + `?utm_source=openai" target="_blank" rel="nofollow">Frozen storage</a> keeps them, ` +
		`<a href="` + whoWater + `">sterile water</a> dissolves them, <a href="` + cdcLabels + `">labels</a> name them, ` +
		`<a href="` + whoPeptides + `">once more</a>, <a href="https://www.nih.gov/uncited">uncited</a>, ` +
		`<a href="https://elsewhere.example/">elsewhere</a>.</p>`

	cases := []struct {
		name     string
		rules    template.SourceLinks
		want     string
		stripped int
	}{
		{
			name: "a template that links no source", rules: template.SourceLinks{},
			want:     `<p>Frozen storage keeps them, sterile water dissolves them, labels name them, once more, uncited, elsewhere.</p>`,
			stripped: 6,
		},
		{
			name: "each cited address once, in the order written, without its tracking", rules: template.SourceLinks{Max: 5},
			want: `<p><a href="` + whoPeptides + `">Frozen storage</a> keeps them, <a href="` + whoWater + `">sterile water</a> dissolves them, ` +
				`<a href="` + cdcLabels + `">labels</a> name them, once more, uncited, elsewhere.</p>`,
			stripped: 3,
		},
		{
			name: "no more than the template's most", rules: template.SourceLinks{Max: 2},
			want: `<p><a href="` + whoPeptides + `">Frozen storage</a> keeps them, <a href="` + whoWater + `">sterile water</a> dissolves them, ` +
				`labels name them, once more, uncited, elsewhere.</p>`,
			stripped: 4,
		},
		{
			name: "no more than the template's most to one site", rules: template.SourceLinks{Max: 5, PerDomain: 1},
			want: `<p><a href="` + whoPeptides + `">Frozen storage</a> keeps them, sterile water dissolves them, ` +
				`<a href="` + cdcLabels + `">labels</a> name them, once more, uncited, elsewhere.</p>`,
			stripped: 4,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			allowance := allowanceFor(field.NoLinks(), false).WithSources(sourcesFor(tc.rules))
			got, stripped := content.Sanitize(fragment, allowance)
			if got != tc.want {
				t.Errorf("Sanitize =\n%s\nwant\n%s", got, tc.want)
			}
			if stripped.Links != tc.stripped {
				t.Errorf("stripped %d links, want %d", stripped.Links, tc.stripped)
			}
		})
	}
}

func TestSanitizeSaysWhyItDroppedALinkOffTheSite(t *testing.T) {
	t.Parallel()

	const fragment = `<p><a href="` + whoPeptides + `">frozen storage</a>, <a href="` + whoWater + `">sterile water</a>, ` +
		`<a href="` + whoPeptides + `">again</a>, <a href="https://www.nih.gov/uncited">uncited</a>, ` +
		`<a href="` + cdcLabels + `">` + cdcLabels + `</a>.</p>`
	allowance := allowanceFor(field.NoLinks(), false).WithSources(sourcesFor(template.SourceLinks{Max: 3, PerDomain: 1}))
	_, stripped := content.Sanitize(fragment, allowance)
	want := []content.DroppedSource{
		{Href: whoWater, Anchor: "sterile water", Reason: content.DroppedSite},
		{Href: whoPeptides, Anchor: "again", Reason: content.DroppedRepeat},
		{Href: "https://www.nih.gov/uncited", Anchor: "uncited", Reason: content.DroppedUncited},
		{Href: cdcLabels, Anchor: cdcLabels, Reason: content.DroppedWords},
	}
	if !slices.Equal(stripped.Sources, want) {
		t.Fatalf("dropped = %+v, want %+v", stripped.Sources, want)
	}

	full := allowanceFor(field.NoLinks(), false).WithSources(sourcesFor(template.SourceLinks{Max: 1}))
	if _, capped := content.Sanitize(`<p><a href="`+whoPeptides+`">frozen</a> <a href="`+cdcLabels+`">labels</a></p>`, full); len(capped.Sources) != 1 ||
		capped.Sources[0].Reason != content.DroppedMost {
		t.Errorf("dropped = %+v, want the second source over the template's most", capped.Sources)
	}
}

func TestTheStrippedFindingNamesTheDroppedSourcesAndWhy(t *testing.T) {
	t.Parallel()

	draft, _, err := content.Assemble(content.DraftAnswer{
		Title: "T", H1: "H",
		Sections: []content.AnswerSection{{Slot: 1, Heading: "One", HTML: `<p><a href="` + whoPeptides + `">frozen storage</a> and ` +
			`<a href="` + whoWater + `">sterile water</a>.</p>`}},
	}, content.Brief{
		Sections: []content.BriefSection{{Slot: 1, Heading: "One", Required: true}},
		Links:    allowanceFor(field.NoLinks(), false).WithSources(sourcesFor(template.SourceLinks{Max: 3, PerDomain: 1})),
	})
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}
	var found *content.Finding
	for i := range draft.Findings {
		if draft.Findings[i].Code == content.CodeMarkupStripped {
			found = &draft.Findings[i]
		}
	}
	if found == nil || !strings.Contains(found.Message, "who.int over the template's most links to one site") {
		t.Fatalf("the finding = %+v, want it to say which source went and why", found)
	}
}

func TestALinkToASourceNeedsWordsOfItsOwn(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		fragment string
		want     string
	}{
		{name: "the bare address", fragment: `<p>See <a href="` + whoPeptides + `">` + whoPeptides + `</a>.</p>`, want: `<p>See ` + whoPeptides + `.</p>`},
		{name: "an address without its scheme", fragment: `<p>See <a href="` + whoPeptides + `">www.who.int/peptides</a>.</p>`, want: `<p>See www.who.int/peptides.</p>`},
		{name: "no words at all", fragment: `<p>See <a href="` + whoPeptides + `"> </a>.</p>`, want: `<p>See  .</p>`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			allowance := allowanceFor(field.NoLinks(), false).WithSources(sourcesFor(template.SourceLinks{Max: 3}))
			if got, _ := content.Sanitize(tc.fragment, allowance); got != tc.want {
				t.Errorf("Sanitize = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestTheLinksToSourcesSpendNothingOfTheSitesLinkBudget(t *testing.T) {
	t.Parallel()

	allowance := allowanceFor(policy(field.LinkRules{MaxLinks: 1, MaxPerTarget: 1}), false).WithSources(sourcesFor(template.SourceLinks{Max: 2}))
	got, stripped := content.Sanitize(`<p><a href="`+whoPeptides+`">Frozen storage</a> and <a href="/coffee/">coffee</a> and `+
		`<a href="`+cdcLabels+`">labels</a> and <a href="/tea/">tea</a>.</p>`, allowance)
	want := `<p><a href="` + whoPeptides + `">Frozen storage</a> and <a href="/coffee/">coffee</a> and <a href="` + cdcLabels + `">labels</a> and tea.</p>`
	if got != want || stripped.Links != 1 {
		t.Errorf("Sanitize = %q (stripped %d), want %q with only the second site link stripped", got, stripped.Links, want)
	}
}

func TestTheSourcesAPageMayLinkAreTheAddressesItsFactsCite(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		research *content.Research
		rules    template.SourceLinks
		cites    bool
		found    int
	}{
		{name: "a researched page under a template that links sources", research: researchedPage(), rules: template.SourceLinks{Max: 2}, cites: true, found: 3},
		{name: "a template that links none", research: researchedPage(), rules: template.SourceLinks{}, found: 3},
		{name: "a page nobody researched", rules: template.SourceLinks{Max: 2}},
		{name: "a research that cited nothing", research: &content.Research{Text: "- A fact. [who.int]"}, rules: template.SourceLinks{Max: 2}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			sources := content.SourcesOf(tc.research, tc.rules)
			if sources.Cites() != tc.cites || sources.Found() != tc.found {
				t.Errorf("Cites() = %t, Found() = %d; want %t, %d", sources.Cites(), sources.Found(), tc.cites, tc.found)
			}
		})
	}
}
