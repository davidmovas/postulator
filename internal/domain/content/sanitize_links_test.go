package content_test

import (
	"testing"

	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
)

func allowanceFor(policy field.LinkPolicy, absolute bool) *content.LinkAllowance {
	lc := content.LinkContext{
		PageID: "page-self", PageURL: "/espresso/", Site: pagemap.NewSite("https://shop.example.com"), Absolute: absolute,
		Targets: []content.LinkTarget{
			{EntityID: "coffee", PageID: "page-coffee", URL: "/coffee/", Anchors: []string{"coffee"}, Relation: content.RelationUp, Required: true},
			{EntityID: "tea", PageID: "page-tea", URL: "/tea/", Anchors: []string{"tea"}, Relation: content.RelationSibling},
		},
	}
	return content.NewLinkAllowance(lc, policy)
}

func TestSanitizeKeepsTheLinksThePolicyAllows(t *testing.T) {
	t.Parallel()

	const fragment = `<p>Read about <a href="/coffee/" class="x" target="_blank">coffee</a>, then <a href="/coffee/">coffee</a> again, ` +
		`<a href="/tea/">a cup</a>, <a href="https://elsewhere.example/">elsewhere</a>, <a href="/espresso/">espresso</a> and <a href="/unknown/">this</a>.</p>`
	allowed := policy(field.LinkRules{MaxLinks: 5, MaxPerTarget: 1})

	cases := []struct {
		name      string
		allowance *content.LinkAllowance
		want      string
		stripped  int
	}{
		{
			name: "a planned target with its anchor, once", allowance: allowanceFor(allowed, false),
			want: `<p>Read about <a href="/coffee/">coffee</a>, then coffee again, a cup, elsewhere, espresso and this.</p>`, stripped: 5,
		},
		{
			name: "written in the site's absolute form", allowance: allowanceFor(allowed, true),
			want: `<p>Read about <a href="https://shop.example.com/coffee/">coffee</a>, then coffee again, a cup, elsewhere, espresso and this.</p>`, stripped: 5,
		},
		{
			name: "a field without links keeps none", allowance: allowanceFor(field.NoLinks(), false),
			want: `<p>Read about coffee, then coffee again, a cup, elsewhere, espresso and this.</p>`, stripped: 6,
		},
		{
			name: "no allowance keeps none", allowance: nil,
			want: `<p>Read about coffee, then coffee again, a cup, elsewhere, espresso and this.</p>`, stripped: 6,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, stripped := content.Sanitize(fragment, tc.allowance)
			if got != tc.want {
				t.Errorf("Sanitize =\n%s\nwant\n%s", got, tc.want)
			}
			if stripped.Links != tc.stripped {
				t.Errorf("stripped %d links, want %d", stripped.Links, tc.stripped)
			}
		})
	}
}

func TestTheWritersLinksCountAgainstThePagesBudget(t *testing.T) {
	t.Parallel()

	allowance := allowanceFor(policy(field.LinkRules{MaxLinks: 1, MaxPerTarget: 1}), false)
	first, _ := content.Sanitize(`<p><a href="/coffee/">coffee</a></p>`, allowance)
	second, stripped := content.Sanitize(`<p><a href="/tea/">tea</a></p>`, allowance)
	if first != `<p><a href="/coffee/">coffee</a></p>` || second != `<p>tea</p>` || stripped.Links != 1 {
		t.Errorf("sections = %q, %q; want the page's one link kept across sections and the next one stripped", first, second)
	}
}

func TestSanitizeStripsTheModelsOwnMarks(t *testing.T) {
	t.Parallel()

	got, stripped := content.Sanitize(`<p class="lead" id="intro" data-source="model" style="color:red">Text <em class="x">here</em>.</p>`, nil)
	if got != `<p>Text <em>here</em>.</p>` {
		t.Errorf("Sanitize = %q, want the model's classes, ids, data and styles gone", got)
	}
	if stripped.Attributes != 5 {
		t.Errorf("stripped %d attributes, want 5", stripped.Attributes)
	}
}

func TestAWriterLinkCountsAsPlaced(t *testing.T) {
	t.Parallel()

	allowance := allowanceFor(policy(field.LinkRules{MaxLinks: 5, MaxPerTarget: 1}), false)
	kept, _ := content.Sanitize(`<p>Read about <a href="/coffee/">coffee</a> first.</p>`, allowance)
	doc, err := content.Parse(kept)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	result := content.InsertLinks(doc, allowance.Context(), policy(field.LinkRules{MaxLinks: 5, MaxPerTarget: 1}))
	if len(result.Placed) == 0 || result.Placed[0].Target.PageID != "page-coffee" || result.Decisions[0].Outcome != content.OutcomeAlreadyLinked {
		t.Errorf("insert = %+v, want the writer's link counted as placed", result)
	}
}
