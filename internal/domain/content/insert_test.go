package content_test

import (
	"math/rand/v2"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/template"
)

func policy(rules template.LinkRules) template.LinkPolicy {
	return template.LinkPolicy{Rules: rules, ForbidExternal: true, ForbidSelf: true, AnchorStrategy: template.AnchorPreferUser}
}

func target(url string, anchors []string, relation content.Relation, required bool) content.LinkTarget {
	return content.LinkTarget{
		EntityID: "e-" + url, PageID: "p-" + url, URL: url, Anchors: anchors,
		Relation: relation, Required: required, Weight: 1, Depth: 1,
	}
}

func contextOf(targets ...content.LinkTarget) content.LinkContext {
	return content.LinkContext{PageID: "self", PageURL: "/self/", EntityID: "self", Targets: targets}
}

func TestInsertLinksGolden(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		body     string
		context  content.LinkContext
		rules    template.LinkRules
		want     string
		placed   int
		missing  int
		outcomes []content.Outcome
	}{
		{
			name:    "the first plain occurrence wins",
			body:    "<p>Coffee is good. Coffee is better.</p>",
			context: contextOf(target("/coffee/", []string{"Coffee"}, content.RelationDown, false)),
			want:    `<p><a href="/coffee/">Coffee</a> is good. Coffee is better.</p>`,
			placed:  1, outcomes: []content.Outcome{content.OutcomeInserted},
		},
		{
			name:    "the casing of the body survives",
			body:    "<p>We roast COFFEE daily.</p>",
			context: contextOf(target("/coffee/", []string{"coffee"}, content.RelationDown, false)),
			want:    `<p>We roast <a href="/coffee/">COFFEE</a> daily.</p>`,
			placed:  1, outcomes: []content.Outcome{content.OutcomeInserted},
		},
		{
			name:    "a utf-8 anchor matches without case",
			body:    "<p>Ein GRÜNER Kaffee schmeckt gut.</p>",
			context: contextOf(target("/gruen/", []string{"grüner Kaffee"}, content.RelationDown, false)),
			want:    `<p>Ein <a href="/gruen/">GRÜNER Kaffee</a> schmeckt gut.</p>`,
			placed:  1, outcomes: []content.Outcome{content.OutcomeInserted},
		},
		{
			name:    "a heading is a forbidden zone",
			body:    "<h2>Coffee</h2><p>We sell coffee here.</p>",
			context: contextOf(target("/coffee/", []string{"coffee"}, content.RelationDown, false)),
			want:    `<h2>Coffee</h2><p>We sell <a href="/coffee/">coffee</a> here.</p>`,
			placed:  1, outcomes: []content.Outcome{content.OutcomeInserted},
		},
		{
			name:    "code and existing links are forbidden zones",
			body:    `<p><code>coffee</code> and <a href="/other/">coffee</a></p>`,
			context: contextOf(target("/coffee/", []string{"coffee"}, content.RelationDown, false)),
			want:    `<p><code>coffee</code> and <a href="/other/">coffee</a></p>`,
			placed:  0, missing: 1, outcomes: []content.Outcome{content.OutcomeAnchorNotFound},
		},
		{
			name:    "an anchor inside an inline tag is still linkable",
			body:    "<p>A <strong>coffee</strong> story about coffee beans.</p>",
			context: contextOf(target("/coffee/", []string{"coffee"}, content.RelationDown, false)),
			want:    `<p>A <strong><a href="/coffee/">coffee</a></strong> story about coffee beans.</p>`,
			placed:  1, outcomes: []content.Outcome{content.OutcomeInserted},
		},
		{
			name:    "an existing link to the target counts as placed",
			body:    `<p>Read our <a href="/coffee/">coffee</a> guide. More coffee here.</p>`,
			context: contextOf(target("/coffee/", []string{"coffee"}, content.RelationDown, false)),
			want:    `<p>Read our <a href="/coffee/">coffee</a> guide. More coffee here.</p>`,
			placed:  1, outcomes: []content.Outcome{content.OutcomeAlreadyLinked},
		},
		{
			name: "the link budget stops the second target",
			body: "<p>We sell coffee and tea.</p>",
			context: contextOf(
				target("/coffee/", []string{"coffee"}, content.RelationDown, false),
				target("/tea/", []string{"tea"}, content.RelationDown, true),
			),
			rules:  template.LinkRules{MaxLinks: 1},
			want:   `<p>We sell <a href="/coffee/">coffee</a> and tea.</p>`,
			placed: 1, missing: 1,
			outcomes: []content.Outcome{content.OutcomeInserted, content.OutcomeCapReached},
		},
		{
			name:    "max per target places the anchor twice",
			body:    "<p>Coffee here, coffee there, coffee everywhere.</p>",
			context: contextOf(target("/coffee/", []string{"coffee"}, content.RelationDown, false)),
			rules:   template.LinkRules{MaxPerTarget: 2},
			want: `<p><a href="/coffee/">Coffee</a> here, <a href="/coffee/">coffee</a> there, ` +
				`coffee everywhere.</p>`,
			placed: 2, outcomes: []content.Outcome{content.OutcomeInserted, content.OutcomeInserted},
		},
		{
			name: "a parent link may only land in the opening paragraphs",
			body: "<p>Nothing here.</p><p>Nothing either.</p><p>All about coffee.</p>",
			context: contextOf(
				target("/coffee/", []string{"coffee"}, content.RelationUp, true),
			),
			rules:  template.LinkRules{ParentLinkWithinParagraphs: 2},
			want:   "<p>Nothing here.</p><p>Nothing either.</p><p>All about coffee.</p>",
			placed: 0, missing: 1,
			outcomes: []content.Outcome{content.OutcomePositionRule},
		},
		{
			name: "a parent link inside the opening paragraphs is placed",
			body: "<p>All about coffee.</p><p>Nothing here.</p>",
			context: contextOf(
				target("/coffee/", []string{"coffee"}, content.RelationUp, true),
			),
			rules:  template.LinkRules{ParentLinkWithinParagraphs: 2},
			want:   `<p>All about <a href="/coffee/">coffee</a>.</p><p>Nothing here.</p>`,
			placed: 1, outcomes: []content.Outcome{content.OutcomeInserted},
		},
		{
			name:    "a target with no anchor in the body is missing",
			body:    "<p>Only tea here.</p>",
			context: contextOf(target("/coffee/", []string{"coffee"}, content.RelationUp, true)),
			want:    "<p>Only tea here.</p>",
			placed:  0, missing: 1,
			outcomes: []content.Outcome{content.OutcomeAnchorNotFound},
		},
		{
			name:    "the second anchor of a target is tried",
			body:    "<p>We love espresso.</p>",
			context: contextOf(target("/coffee/", []string{"coffee", "espresso"}, content.RelationDown, false)),
			want:    `<p>We love <a href="/coffee/">espresso</a>.</p>`,
			placed:  1, outcomes: []content.Outcome{content.OutcomeInserted},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			doc := mustParse(t, tc.body)
			result := content.InsertLinks(doc, tc.context, policy(tc.rules))

			if doc.HTML() != tc.want {
				t.Fatalf("HTML =\n%s\nwant\n%s", doc.HTML(), tc.want)
			}
			if len(result.Placed) != tc.placed {
				t.Errorf("Placed = %d, want %d (%+v)", len(result.Placed), tc.placed, result.Placed)
			}
			if len(result.Missing) != tc.missing {
				t.Errorf("Missing = %d, want %d", len(result.Missing), tc.missing)
			}
			if len(result.Decisions) != len(tc.outcomes) {
				t.Fatalf("Decisions = %+v, want %v", result.Decisions, tc.outcomes)
			}
			for i, outcome := range tc.outcomes {
				if result.Decisions[i].Outcome != outcome {
					t.Errorf("decision %d = %q, want %q", i, result.Decisions[i].Outcome, outcome)
				}
				if result.Decisions[i].Detail == "" {
					t.Errorf("decision %d carries no detail", i)
				}
			}
		})
	}
}

func TestInsertLinksIsIdempotent(t *testing.T) {
	t.Parallel()

	words := []string{"coffee", "tea", "beans", "roast", "milk", "sugar", "water", "cup"}
	source := rand.New(rand.NewPCG(7, 11))

	for run := range 200 {
		body := randomBody(source, words)
		targets := randomTargets(source, words)
		rules := template.LinkRules{
			MaxLinks:                   source.IntN(4),
			MaxPerTarget:               1 + source.IntN(2),
			ParentLinkWithinParagraphs: source.IntN(3),
		}

		doc := mustParse(t, body)
		first := content.InsertLinks(doc, contextOf(targets...), policy(rules))
		once := doc.HTML()

		second := content.InsertLinks(doc, contextOf(targets...), policy(rules))
		if doc.HTML() != once {
			t.Fatalf("run %d: a second pass changed the body\nbefore %s\nafter  %s", run, once, doc.HTML())
		}
		if len(second.Placed) < len(first.Placed) {
			t.Fatalf("run %d: the second pass lost placements: %d then %d", run, len(first.Placed), len(second.Placed))
		}
		for _, decision := range second.Decisions {
			if decision.Outcome == content.OutcomeInserted {
				t.Fatalf("run %d: the second pass inserted %s again", run, decision.Target.URL)
			}
		}
	}
}

func randomBody(source *rand.Rand, words []string) string {
	var builder strings.Builder
	builder.WriteString("<h1>" + words[source.IntN(len(words))] + "</h1>")

	for range 1 + source.IntN(4) {
		builder.WriteString("<p>")
		for range 3 + source.IntN(8) {
			word := words[source.IntN(len(words))]
			switch source.IntN(6) {
			case 0:
				builder.WriteString("<strong>" + word + "</strong> ")
			case 1:
				builder.WriteString("<code>" + word + "</code> ")
			default:
				builder.WriteString(word + " ")
			}
		}
		builder.WriteString("</p>")
	}
	return builder.String()
}

func randomTargets(source *rand.Rand, words []string) []content.LinkTarget {
	count := 1 + source.IntN(3)

	out := make([]content.LinkTarget, 0, count)
	for i := range count {
		word := words[source.IntN(len(words))]
		relation := content.RelationDown
		if i == 0 {
			relation = content.RelationUp
		}
		out = append(out, target("/"+word+"-"+string(rune('a'+i))+"/", []string{word}, relation, relation == content.RelationUp))
	}
	return out
}

func TestInsertLinksLeavesADocumentWithNoTargetsAlone(t *testing.T) {
	t.Parallel()

	doc := mustParse(t, "<p>Nothing to link.</p>")
	result := content.InsertLinks(doc, contextOf(), policy(template.LinkRules{}))

	if doc.HTML() != "<p>Nothing to link.</p>" {
		t.Fatalf("HTML = %q", doc.HTML())
	}
	if len(result.Placed) != 0 || len(result.Missing) != 0 || len(result.Decisions) != 0 {
		t.Fatalf("InsertLinks = %+v", result)
	}
}

func TestInsertLinksLeavesNonProseZonesAlone(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		body string
		want string
	}{
		{
			name: "a script body is never spliced",
			body: `<script>var label = "coffee";</script><p>Nothing else.</p>`,
			want: `<script>var label = "coffee";</script><p>Nothing else.</p>`,
		},
		{
			name: "a stylesheet is never spliced",
			body: `<style>.coffee { color: red; }</style><p>Nothing else.</p>`,
			want: `<style>.coffee { color: red; }</style><p>Nothing else.</p>`,
		},
		{
			name: "a textarea value is never spliced",
			body: `<textarea>coffee</textarea><p>Nothing else.</p>`,
			want: `<textarea>coffee</textarea><p>Nothing else.</p>`,
		},
		{
			name: "a noscript fallback is never spliced",
			body: `<noscript>coffee</noscript><p>Nothing else.</p>`,
			want: `<noscript>coffee</noscript><p>Nothing else.</p>`,
		},
		{
			name: "prose beside a script is still linked",
			body: `<script>var label = "coffee";</script><p>We roast coffee.</p>`,
			want: `<script>var label = "coffee";</script><p>We roast <a href="/coffee/">coffee</a>.</p>`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			doc := mustParse(t, tc.body)
			content.InsertLinks(doc, contextOf(target("/coffee/", []string{"coffee"}, content.RelationDown, false)),
				policy(template.LinkRules{MaxLinks: 5, MaxPerTarget: 1}))

			if doc.HTML() != tc.want {
				t.Fatalf("HTML =\n%s\nwant\n%s", doc.HTML(), tc.want)
			}
		})
	}
}
