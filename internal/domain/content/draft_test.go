package content_test

import (
	stderrors "errors"
	"slices"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func guideBrief(page pagemap.Page) content.Brief {
	spec := template.TemplateSpec{
		Sections: []template.Section{
			{Heading: "Overview", Required: true, KeywordRules: template.SectionKeywordRules{PrimaryInHeading: true}},
			{Heading: "Brewing", Required: true},
			{Heading: "Extras"},
		},
		KeywordRules: template.KeywordRules{PrimaryInTitle: true, PrimaryInH1: true, PrimaryInFirstParagraph: true},
	}
	entity := graph.Entity{Name: "Espresso", PrimaryKeyword: "espresso"}
	lc := content.LinkContext{Targets: []content.LinkTarget{
		{URL: "/coffee/", Anchors: []string{"coffee"}, Relation: content.RelationUp, Required: true},
		{URL: "/coffee/filter/", Anchors: []string{"filter coffee"}, Relation: content.RelationSibling},
	}}
	return content.NewBrief(spec, template.LinkRules{ParentLinkWithinParagraphs: 2}, page, entity, lc)
}

func reasonOf(t *testing.T, err error) string {
	t.Helper()
	var kernel *errors.Error
	if !stderrors.As(err, &kernel) {
		t.Fatalf("error %v is not a kernel error", err)
	}
	reason, ok := kernel.Details["reason"].(string)
	if !ok {
		return ""
	}
	return reason
}

func codesIn(findings []content.Finding) []string {
	out := make([]string, 0, len(findings))
	for _, finding := range findings {
		out = append(out, finding.Code)
	}
	return out
}

func TestNewBriefTakesThePlanFirstAndListsWhatThePageOwes(t *testing.T) {
	t.Parallel()

	brief := guideBrief(pagemap.Page{Title: "Espresso at home", H1: "Pull a better espresso"})
	if !brief.PlannedTitle || !brief.PlannedH1 || brief.Title != "Espresso at home" || brief.H1 != "Pull a better espresso" {
		t.Fatalf("brief = %+v, want the plan's title and h1", brief)
	}
	if len(brief.Sections) != 3 || brief.Sections[0].Slot != 1 || brief.Sections[2].Slot != 3 || !brief.Sections[0].PrimaryInHeading {
		t.Fatalf("sections = %+v", brief.Sections)
	}
	if got := brief.PhraseTexts(); len(got) != 3 || got[0] != "espresso" || got[1] != "coffee" || got[2] != "filter coffee" {
		t.Fatalf("phrases = %v, want the lead keyword, the parent anchor and the sibling anchor", got)
	}
	if !brief.Phrases[0].Lead || brief.Phrases[1].Lead || brief.Phrases[2].Lead {
		t.Fatalf("only the keyword opens the page: %+v", brief.Phrases)
	}
	if brief.Phrases[0].Within != 0 || brief.Phrases[1].Within != 2 || brief.Phrases[2].Within != 0 {
		t.Fatalf("phrases = %+v, want the parent anchor within the first two paragraphs and the others unbounded", brief.Phrases)
	}
	if got := brief.RequiredHeadings(); len(got) != 2 || got[1] != "Brewing" {
		t.Fatalf("required headings = %v", got)
	}

	bare := guideBrief(pagemap.Page{})
	if bare.PlannedTitle || bare.PlannedH1 {
		t.Fatalf("a page without a plan claims one: %+v", bare)
	}
}

func TestNewBriefOwesEveryLinkTheBudgetAllows(t *testing.T) {
	t.Parallel()

	lc := content.LinkContext{Targets: []content.LinkTarget{
		{URL: "/coffee/", Anchors: []string{"coffee"}, Relation: content.RelationUp, Required: true},
		{URL: "/coffee/espresso/ristretto/", Anchors: []string{"ristretto"}, Relation: content.RelationDown},
		{URL: "/coffee/espresso/lungo/", Relation: content.RelationDown},
		{URL: "/coffee/filter/", Anchors: []string{"filter coffee"}, Relation: content.RelationSibling},
	}}
	entity := graph.Entity{Name: "Espresso", PrimaryKeyword: "espresso"}

	cases := []struct {
		name     string
		rules    template.LinkRules
		phrases  []string
		children []string
	}{
		{
			name:    "every target in running text",
			rules:   template.LinkRules{ParentLinkWithinParagraphs: 2},
			phrases: []string{"coffee", "ristretto", "filter coffee"},
		},
		{
			name:     "the children in a section of their own",
			rules:    template.LinkRules{ParentLinkWithinParagraphs: 2, ChildrenSection: true},
			phrases:  []string{"coffee", "filter coffee"},
			children: []string{"ristretto"},
		},
		{
			name:    "a budget that ends before the sibling",
			rules:   template.LinkRules{ParentLinkWithinParagraphs: 2, MaxLinks: 3},
			phrases: []string{"coffee", "ristretto"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			brief := content.NewBrief(template.TemplateSpec{}, tc.rules, pagemap.Page{}, entity, lc)
			if got := brief.PhraseTexts(); !slices.Equal(got, tc.phrases) {
				t.Fatalf("phrases = %v, want %v", got, tc.phrases)
			}
			if !slices.Equal(brief.Children, tc.children) {
				t.Fatalf("children = %v, want %v", brief.Children, tc.children)
			}
			for _, phrase := range brief.Phrases {
				if phrase.Why == "" || !strings.Contains(phrase.Why, "/coffee/") {
					t.Fatalf("phrase %+v, want it to say which page it links to", phrase)
				}
			}
		})
	}
}

func TestAssembleTakesTheHeadingsOfTheBrief(t *testing.T) {
	t.Parallel()

	answer := content.DraftAnswer{
		Title: "Espresso guide", H1: "Espresso at home",
		Sections: []content.AnswerSection{
			{Slot: 2, Heading: "How to brew it", HTML: "<p>Grind fine and pull for thirty seconds.</p>"},
			{Slot: 1, Heading: "An overview of espresso", HTML: "<p>Espresso is coffee under pressure.</p>"},
			{Slot: 0, Heading: "Children", HTML: "<p>Read about filter coffee next.</p>"},
		},
		Summary: "A guide.",
	}

	draft, doc, err := content.Assemble(answer, guideBrief(pagemap.Page{}))
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}
	headings := make([]string, 0, 3)
	for _, section := range draft.Sections {
		headings = append(headings, section.Heading)
	}
	if strings.Join(headings, "|") != "An overview of espresso|Brewing|Children" {
		t.Fatalf("headings = %v, want the keyword heading the model wrote, the brief's heading, then the extra", headings)
	}
	if len(draft.Findings) != 0 {
		t.Fatalf("findings = %v, want none", codesIn(draft.Findings))
	}
	body, err := doc.Render()
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.HasPrefix(body, "<h1>Espresso at home</h1><h2>An overview of espresso</h2>") {
		t.Fatalf("body = %q", body)
	}
}

func TestAssembleMatchesByHeadingThenByPositionAndWarnsOnAKeywordlessHeading(t *testing.T) {
	t.Parallel()

	answer := content.DraftAnswer{
		Title: "Espresso guide", H1: "Espresso at home",
		Sections: []content.AnswerSection{
			{Heading: "Introduction", HTML: "<p>Espresso is coffee under pressure.</p>"},
			{Heading: "Brewing basics", HTML: "<p>Grind fine.</p>"},
		},
	}

	draft, _, err := content.Assemble(answer, guideBrief(pagemap.Page{}))
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}
	if len(draft.Sections) != 2 || draft.Sections[0].Heading != "Overview" || draft.Sections[1].Heading != "Brewing" {
		t.Fatalf("sections = %+v, want Brewing matched by heading and Overview by position with the brief's names", draft.Sections)
	}
	if codes := codesIn(draft.Findings); len(codes) != 1 || codes[0] != content.CodeHeadingLacksKeyword {
		t.Fatalf("findings = %v, want the heading that lacks the keyword named once", codes)
	}
}

func TestAssembleTitlesAndOpensThePageFromThePlanOrTheKeyword(t *testing.T) {
	t.Parallel()

	sections := []content.AnswerSection{
		{Slot: 1, Heading: "Espresso overview", HTML: "<p>Espresso is coffee under pressure.</p>"},
		{Slot: 2, Heading: "Brewing", HTML: "<p>Grind fine.</p>"},
	}

	cases := []struct {
		name      string
		page      pagemap.Page
		answer    content.DraftAnswer
		wantTitle string
		wantH1    string
		codes     []string
	}{
		{
			name:      "the plan wins over the model",
			page:      pagemap.Page{Title: "Espresso at home", H1: "Pull a better espresso"},
			answer:    content.DraftAnswer{Title: "Coffee", H1: "Coffee", Sections: sections},
			wantTitle: "Espresso at home", wantH1: "Pull a better espresso", codes: []string{},
		},
		{
			name:      "a plan without the keyword is kept and named",
			page:      pagemap.Page{Title: "Our shop", H1: "Welcome"},
			answer:    content.DraftAnswer{Title: "Espresso", H1: "Espresso", Sections: sections},
			wantTitle: "Our shop", wantH1: "Welcome",
			codes: []string{content.CodePlanTitleLacksKeyword, content.CodePlanH1LacksKeyword},
		},
		{
			name:      "the model's words are kept when they carry the keyword",
			answer:    content.DraftAnswer{Title: "Espresso, explained", H1: "Espresso at home", Sections: sections},
			wantTitle: "Espresso, explained", wantH1: "Espresso at home", codes: []string{},
		},
		{
			name:      "a keywordless answer falls back to the keyword",
			answer:    content.DraftAnswer{Title: "Coffee", H1: "", Sections: sections},
			wantTitle: "Espresso", wantH1: "Espresso",
			codes: []string{content.CodeTitleFallback, content.CodeH1Fallback},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			draft, _, err := content.Assemble(tc.answer, guideBrief(tc.page))
			if err != nil {
				t.Fatalf("Assemble: %v", err)
			}
			if draft.Title != tc.wantTitle || draft.H1 != tc.wantH1 {
				t.Fatalf("title = %q, h1 = %q; want %q and %q", draft.Title, draft.H1, tc.wantTitle, tc.wantH1)
			}
			if codes := codesIn(draft.Findings); strings.Join(codes, ",") != strings.Join(tc.codes, ",") {
				t.Fatalf("findings = %v, want %v", codes, tc.codes)
			}
		})
	}
}

func TestAssembleAsksForAnotherAnswerWhenARequiredSectionIsMissingOrEmpty(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		answer content.DraftAnswer
	}{
		{name: "no sections", answer: content.DraftAnswer{Title: "t", H1: "h"}},
		{
			name: "a required section is missing",
			answer: content.DraftAnswer{Title: "t", H1: "h", Sections: []content.AnswerSection{
				{Slot: 1, Heading: "Overview", HTML: "<p>Espresso.</p>"},
			}},
		},
		{
			name: "a required section is empty",
			answer: content.DraftAnswer{Title: "t", H1: "h", Sections: []content.AnswerSection{
				{Slot: 1, Heading: "Overview", HTML: "<p>Espresso.</p>"},
				{Slot: 2, Heading: "Brewing", HTML: "<a href='/x/'></a>"},
			}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, _, err := content.Assemble(tc.answer, guideBrief(pagemap.Page{}))
			if !errors.IsCode(err, errors.External) || reasonOf(t, err) != content.ReasonIncompleteAnswer {
				t.Fatalf("Assemble = %v, want EXTERNAL with the incomplete_answer reason so the writer is asked again", err)
			}
		})
	}
}

func TestAssembleDropsAnEmptyOptionalSectionAndStripsForbiddenMarkup(t *testing.T) {
	t.Parallel()

	answer := content.DraftAnswer{
		Title: "Espresso guide", H1: "Espresso at home",
		Sections: []content.AnswerSection{
			{Slot: 1, Heading: "Overview", HTML: `<p>Espresso is <a href="https://example.com">coffee</a> under pressure.</p><script>alert(1)</script>`},
			{Slot: 2, Heading: "Brewing", HTML: `<h2>Grind</h2><p onclick="x()">Grind fine.</p>`},
			{Slot: 3, Heading: "Extras", HTML: "   "},
		},
	}

	draft, doc, err := content.Assemble(answer, guideBrief(pagemap.Page{}))
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}
	if len(draft.Sections) != 2 {
		t.Fatalf("sections = %+v, want the empty optional one dropped", draft.Sections)
	}
	body, err := doc.Render()
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	for _, forbidden := range []string{"<a ", "<script", "onclick", "<h2>Grind"} {
		if strings.Contains(body, forbidden) {
			t.Errorf("body still carries %q: %s", forbidden, body)
		}
	}
	if !strings.Contains(body, "coffee under pressure") || !strings.Contains(body, "<h3>Grind</h3>") {
		t.Errorf("body lost the text or the demoted heading: %s", body)
	}
	codes := codesIn(draft.Findings)
	if strings.Count(strings.Join(codes, ","), content.CodeMarkupStripped) != 2 || !strings.Contains(strings.Join(codes, ","), content.CodeSectionEmpty) {
		t.Fatalf("findings = %v, want two stripped sections and one empty one", codes)
	}
}

func TestRenderRefusesAStoredDraftWithoutAnH1OrSections(t *testing.T) {
	t.Parallel()

	if _, err := content.Render(content.ContentDraft{Title: "t", Sections: []content.DraftSection{{Heading: "s", HTML: "<p>x</p>"}}}); !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("Render without an h1 = %v", err)
	}
	if _, err := content.Render(content.ContentDraft{Title: "t", H1: "h"}); !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("Render without sections = %v", err)
	}
}
