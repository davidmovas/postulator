package pagetype_test

import (
	"slices"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
)

func upgraded() []pagetype.Type {
	return append(legacy("s1"), fresh("s1")[:4]...)
}

type pageOption func(*pagetype.Facts)

func aPage(path string, level int, options ...pageOption) pagetype.Facts {
	f := pagetype.Facts{PageID: path, Path: path, Slug: slugOf(path), WPType: pagemap.WPPage, Rendered: true, Level: level, Words: 120}
	for _, option := range options {
		option(&f)
	}
	return f
}

func slugOf(path string) string {
	segments := []rune(path)
	end := len(segments) - 1
	start := end - 1
	for start > 0 && segments[start] != '/' {
		start--
	}
	return string(segments[start+1 : end])
}

func wp(kind pagemap.WPType) pageOption { return func(f *pagetype.Facts) { f.WPType = kind } }
func technical(reason pagemap.TechnicalReason) pageOption {
	return func(f *pagetype.Facts) { f.Technical = reason }
}
func front() pageOption {
	return func(f *pagetype.Facts) { f.FrontPage = true }
}
func unrendered() pageOption {
	return func(f *pagetype.Facts) { f.Rendered = false }
}
func lists(products int) pageOption {
	return func(f *pagetype.Facts) { f.Listing, f.Products = true, products }
}
func schema(kinds ...string) pageOption {
	return func(f *pagetype.Facts) { f.Schema = kinds }
}
func long(words, h2 int) pageOption {
	return func(f *pagetype.Facts) { f.Words, f.H2 = words, h2 }
}
func under(ancestors ...string) pageOption {
	return func(f *pagetype.Facts) { f.Ancestors = ancestors }
}
func typed(typeID string, source pagemap.TypeSource, reason pagemap.TypeReason, ruleID string) pageOption {
	return func(f *pagetype.Facts) {
		f.Current = pagetype.Assignment{TypeID: typeID, Source: source, Reason: reason, RuleID: ruleID}
	}
}

func classifier(t *testing.T, types []pagetype.Type) (pagetype.Classifier, []pagetype.Rule) {
	t.Helper()
	set := pagetype.NewSet(types, nil)
	rules := pagetype.DefaultRules(set, "s1", stamp, ids())
	return pagetype.NewClassifier(rules, set), rules
}

func byPage(outcomes []pagetype.Outcome) map[string]pagetype.Outcome {
	out := make(map[string]pagetype.Outcome, len(outcomes))
	for i := range outcomes {
		out[outcomes[i].PageID] = outcomes[i]
	}
	return out
}

func TestTheDefaultRulesTypeEveryKindOfPage(t *testing.T) {
	t.Parallel()

	c, rules := classifier(t, fresh("s1"))
	ruleAt := func(position int) string { return rules[position-1].ID }
	pages := []pagetype.Facts{
		aPage("/", 0, front(), lists(4)),
		aPage("/cart/", 1, technical(pagemap.TechnicalCart)),
		aPage("/hidden-product/", 2, wp(pagemap.WPProduct), technical(pagemap.TechnicalNoindex)),
		aPage("/product/bpc/", 2, wp(pagemap.WPProduct)),
		aPage("/product-category/peptides/", 1, wp(pagemap.WPProductCategory)),
		aPage("/shop/", 1, technical(pagemap.TechnicalShop)),
		aPage("/peptides/", 1, lists(6)),
		aPage("/healing-peptides/", 2, lists(3), under("/peptides/")),
		aPage("/hello-world/", 1, wp(pagemap.WPPost), schema("Article")),
		aPage("/storage/", 1, schema("BlogPosting", "WebPage")),
		aPage("/lab-guides/", 1),
		aPage("/how-to-reconstitute-bpc/", 1),
		aPage("/reconstitution/", 2, under("/lab-guides/")),
		aPage("/our-company/", 1, schema("AboutPage")),
		aPage("/about-us/", 1),
		aPage("/peptide-storage-explained/", 1, long(1400, 3)),
		aPage("/listing-notes/", 3, long(1400, 3), lists(2)),
		aPage("/sample-page/", 1),
		aPage("/short-with-sections/", 1, long(300, 4)),
	}
	want := map[string]struct {
		typeID string
		rule   int
	}{
		"/":                           {"s1-landing", 5},
		"/cart/":                      {"s1-technical", 1},
		"/hidden-product/":            {"s1-product", 2},
		"/product/bpc/":               {"s1-product", 2},
		"/product-category/peptides/": {"s1-store_category", 3},
		"/shop/":                      {"s1-catalog", 4},
		"/peptides/":                  {"s1-landing", 6},
		"/healing-peptides/":          {"s1-category", 7},
		"/hello-world/":               {"s1-guide", 8},
		"/storage/":                   {"s1-guide", 9},
		"/lab-guides/":                {"s1-guide", 10},
		"/how-to-reconstitute-bpc/":   {"s1-guide", 10},
		"/reconstitution/":            {"s1-guide", 11},
		"/our-company/":               {"s1-info", 12},
		"/about-us/":                  {"s1-info", 13},
		"/peptide-storage-explained/": {"s1-guide", 14},
		"/listing-notes/":             {"s1-category", 7},
		"/sample-page/":               {"s1-info", 0},
		"/short-with-sections/":       {"s1-info", 0},
	}
	got := byPage(c.Classify(pages, pagetype.ModeWalk))
	for path, expected := range want {
		outcome := got[path]
		wantRule, wantReason := "", pagemap.TypeByUnmatched
		if expected.rule > 0 {
			wantRule, wantReason = ruleAt(expected.rule), pagemap.TypeByRule
		}
		if outcome.TypeID != expected.typeID || outcome.RuleID != wantRule || outcome.Reason != wantReason || outcome.Waiting || !outcome.Changed() {
			t.Errorf("%s = %s by %q (%s, waiting %t), want %s by rule %d", path, outcome.TypeID, outcome.RuleID, outcome.Reason,
				outcome.Waiting, expected.typeID, expected.rule)
		}
	}
}

func TestAPageWithoutARenderWaitsAtTheFirstRuleThatNeedsOne(t *testing.T) {
	t.Parallel()

	c, rules := classifier(t, fresh("s1"))
	pages := []pagetype.Facts{
		aPage("/product/new/", 2, wp(pagemap.WPProduct), unrendered()),
		aPage("/new/", 1, unrendered()),
		aPage("/a/b/c/", 3, unrendered()),
		aPage("/new/child/", 2, under("/new/")),
		aPage("/about-us/", 1, unrendered()),
	}
	got := byPage(c.Classify(pages, pagetype.ModeWalk))
	if product := got["/product/new/"]; product.Waiting || product.TypeID != "s1-product" {
		t.Errorf("an unrendered product = %+v, want it typed by its identity", product)
	}
	for _, path := range []string{"/new/", "/a/b/c/", "/about-us/"} {
		if outcome := got[path]; !outcome.Waiting || outcome.Changed() || outcome.TypeID != "" {
			t.Errorf("%s = %+v, want it waiting for its render", path, outcome)
		}
	}
	if child := got["/new/child/"]; !child.Waiting {
		t.Errorf("a page under a page that waits = %+v, want it waiting when a rule asks for the page above's type", child)
	}

	explained := c.Explain(rules[5], "/new/", pages, pagetype.ModeWalk)
	if explained.Verdict != pagetype.VerdictWaiting || !slices.Equal(explained.Conditions, []pagetype.ConditionResult{
		{Condition: pagetype.ConditionLevel, Verdict: pagetype.VerdictMet},
		{Condition: pagetype.ConditionListing, Verdict: pagetype.VerdictWaiting},
	}) {
		t.Errorf("Explain = %+v, want the level met and the listing waiting for the render", explained)
	}
}

func TestTheFirstRuleThatMatchesWinsAndADisabledOneIsSkipped(t *testing.T) {
	t.Parallel()

	set := pagetype.NewSet(fresh("s1"), nil)
	rules := pagetype.DefaultRules(set, "s1", stamp, ids())
	post := aPage("/hello-world/", 1, wp(pagemap.WPPost), schema("Article"))

	got := byPage(pagetype.NewClassifier(rules, set).Classify([]pagetype.Facts{post}, pagetype.ModeWalk))
	if got["/hello-world/"].RuleID != rules[7].ID {
		t.Fatalf("a post with an Article schema = %+v, want the blog-post rule that comes first", got["/hello-world/"])
	}
	rules[7].Enabled = false
	got = byPage(pagetype.NewClassifier(rules, set).Classify([]pagetype.Facts{post}, pagetype.ModeWalk))
	if got["/hello-world/"].RuleID != rules[8].ID {
		t.Fatalf("with the blog-post rule off = %+v, want the schema rule after it", got["/hello-world/"])
	}

	rules[7].Enabled = true
	rules[7].Position, rules[8].Position = rules[8].Position, rules[7].Position
	got = byPage(pagetype.NewClassifier(rules, set).Classify([]pagetype.Facts{post}, pagetype.ModeWalk))
	if got["/hello-world/"].RuleID != rules[8].ID {
		t.Fatalf("with the two rules swapped = %+v, want the one now first", got["/hello-world/"])
	}
}

func TestARuleNeverGivesAProductOrATermARoleItCannotHold(t *testing.T) {
	t.Parallel()

	set := pagetype.NewSet(fresh("s1"), nil)
	rules := pagetype.DefaultRules(set, "s1", stamp, ids())
	products := pagetype.Rule{
		ID: "items", SiteID: "s1", Kind: pagetype.KindRule, Name: "Items", Enabled: true, TypeID: "s1-product",
		Conditions: pagetype.Conditions{URLs: []pagetype.URLMatch{{Mode: pagetype.URLPrefix, Value: "/items/"}}},
	}
	guides := pagetype.Rule{
		ID: "everything", SiteID: "s1", Kind: pagetype.KindRule, Name: "Everything under shop", Enabled: true, TypeID: "s1-guide",
		Conditions: pagetype.Conditions{URLs: []pagetype.URLMatch{{Mode: pagetype.URLGlob, Value: "/**"}}},
	}
	c := pagetype.NewClassifier(append([]pagetype.Rule{products, guides}, rules...), set)
	got := byPage(c.Classify([]pagetype.Facts{
		aPage("/items/mug/", 2),
		aPage("/items/cup/", 2, wp(pagemap.WPProduct)),
		aPage("/product-category/cups/", 1, wp(pagemap.WPProductCategory)),
	}, pagetype.ModeWalk))
	if page := got["/items/mug/"]; page.TypeID != "s1-guide" || page.RuleID != "everything" {
		t.Errorf("a page a product rule matched = %+v, want it passed on to the next rule", page)
	}
	if product := got["/items/cup/"]; product.TypeID != "s1-product" || product.RuleID != "items" {
		t.Errorf("a product = %+v, want the product rule", product)
	}
	if term := got["/product-category/cups/"]; term.TypeID != "s1-store_category" || term.RuleID != rules[2].ID {
		t.Errorf("a term = %+v, want the store category rule, never the catch-all guide", term)
	}
}

func TestAPersonsTypeIsKeptAndATypeFromBefore26MovesOnlyInThePreview(t *testing.T) {
	t.Parallel()

	c, rules := classifier(t, upgraded())
	pages := []pagetype.Facts{
		aPage("/about-us/", 1, typed("s1-catalog", pagemap.TypeSourceManual, pagemap.TypeByUser, "")),
		aPage("/imported/", 1, typed("s1-hub", pagemap.TypeSourceManual, pagemap.TypeByImport, "")),
		aPage("/lab-guides/", 1, typed("s1-entity", pagemap.TypeSourceAuto, pagemap.TypeByDefault, "")),
		aPage("/reconstitution/", 2, under("/lab-guides/"), typed("s1-guide", pagemap.TypeSourceAuto, pagemap.TypeByRule, rules[10].ID)),
		aPage("/new-page/", 2, under("/lab-guides/")),
		aPage("/learn/", 1, typed("s1-info", pagemap.TypeSourceAuto, pagemap.TypeByUnmatched, "")),
	}

	walk := byPage(c.Classify(pages, pagetype.ModeWalk))
	for _, path := range []string{"/about-us/", "/imported/", "/lab-guides/"} {
		if outcome := walk[path]; outcome.Eligible || outcome.Changed() {
			t.Errorf("the walk moved %s: %+v", path, outcome)
		}
	}
	if learn := walk["/learn/"]; !learn.Changed() || learn.TypeID != "s1-guide" || learn.RuleID != rules[9].ID {
		t.Errorf("a page the rules placed = %+v, want the walk to keep it in line with the rules", learn)
	}
	if moved := walk["/reconstitution/"]; moved.TypeID != "s1-info" || moved.Reason != pagemap.TypeByUnmatched {
		t.Errorf("under an Entity page the walk keeps = %+v, want it no longer under a guide", moved)
	}
	if fresh := walk["/new-page/"]; fresh.TypeID != "s1-info" {
		t.Errorf("a new page under an Entity page = %+v", fresh)
	}

	preview := byPage(c.Classify(pages, pagetype.ModePreview))
	if guides := preview["/lab-guides/"]; !guides.Retyped() || guides.TypeID != "s1-guide" {
		t.Errorf("the preview of a page typed before 2.6 = %+v, want it moved to Guide", guides)
	}
	for _, path := range []string{"/reconstitution/", "/new-page/"} {
		outcome := preview[path]
		if outcome.TypeID != "s1-guide" || !slices.Equal(outcome.Depends, []string{"/lab-guides/"}) {
			t.Errorf("the preview of %s = %+v, want Guide under the guide page it depends on", path, outcome)
		}
	}
	for _, path := range []string{"/about-us/", "/imported/"} {
		if outcome := preview[path]; outcome.Eligible || outcome.Changed() {
			t.Errorf("the preview moved %s, a person's or an import's choice: %+v", path, outcome)
		}
	}
}

func TestAnAddressIsMatchedByItsStartAWordInItOrAPattern(t *testing.T) {
	t.Parallel()

	set := pagetype.NewSet(fresh("s1"), nil)
	cases := []struct {
		match pagetype.URLMatch
		path  string
		want  bool
	}{
		{pagetype.URLMatch{Mode: pagetype.URLPrefix, Value: "/blog/"}, "/blog/", true},
		{pagetype.URLMatch{Mode: pagetype.URLPrefix, Value: "/blog/"}, "/blog/first/", true},
		{pagetype.URLMatch{Mode: pagetype.URLPrefix, Value: "/blog/"}, "/blogger/", false},
		{pagetype.URLMatch{Mode: pagetype.URLPrefix, Value: "/blog"}, "/blog/first/", true},
		{pagetype.URLMatch{Mode: pagetype.URLSegment, Value: "guides"}, "/lab-guides/", true},
		{pagetype.URLMatch{Mode: pagetype.URLSegment, Value: "guide"}, "/lab-guides/", false},
		{pagetype.URLMatch{Mode: pagetype.URLSegment, Value: "how-to"}, "/peptides/how-to-store/", true},
		{pagetype.URLMatch{Mode: pagetype.URLSegment, Value: "how-to"}, "/to-how/", false},
		{pagetype.URLMatch{Mode: pagetype.URLSegment, Value: "learn"}, "/Learn/", true},
		{pagetype.URLMatch{Mode: pagetype.URLGlob, Value: "/blog/*/"}, "/blog/first/", true},
		{pagetype.URLMatch{Mode: pagetype.URLGlob, Value: "/blog/*/"}, "/blog/first/second/", false},
		{pagetype.URLMatch{Mode: pagetype.URLGlob, Value: "/blog/**"}, "/blog/first/second/", true},
		{pagetype.URLMatch{Mode: pagetype.URLGlob, Value: "/blog/**"}, "/blog/", true},
		{pagetype.URLMatch{Mode: pagetype.URLGlob, Value: "/**/*-guide/"}, "/peptides/bpc-157-guide/", true},
		{pagetype.URLMatch{Mode: pagetype.URLGlob, Value: "/**/*-guide/"}, "/bpc-157-guides/", false},
		{pagetype.URLMatch{Mode: pagetype.URLGlob, Value: "/*a*b*/"}, "/xaybz/", true},
		{pagetype.URLMatch{Mode: pagetype.URLGlob, Value: "/*a*b*/"}, "/xbya/", false},
	}
	for _, tc := range cases {
		rule := aRule("s1-guide", pagetype.Conditions{URLs: []pagetype.URLMatch{tc.match}})
		got := byPage(pagetype.NewClassifier([]pagetype.Rule{rule}, set).Classify([]pagetype.Facts{aPage(tc.path, 1)}, pagetype.ModeWalk))
		if matched := got[tc.path].RuleID == rule.ID; matched != tc.want {
			t.Errorf("%s %q on %s = %t, want %t", tc.match.Mode, tc.match.Value, tc.path, matched, tc.want)
		}
	}
}

func TestARuleSaysWhichPagesItMatches(t *testing.T) {
	t.Parallel()

	c, rules := classifier(t, fresh("s1"))
	pages := []pagetype.Facts{aPage("/about-us/", 1), aPage("/contact/", 1), aPage("/peptides/", 1, lists(4))}
	if got := c.Matches(rules[12], pages, pagetype.ModeWalk); !slices.Equal(got, []string{"/about-us/", "/contact/"}) {
		t.Errorf("the slug rule matches %v", got)
	}
	explained := c.Explain(rules[12], "/peptides/", pages, pagetype.ModeWalk)
	if explained.Verdict != pagetype.VerdictUnmet || explained.Outcome.TypeID != "s1-landing" {
		t.Errorf("Explain on another page = %+v, want the slug unmet and the page's own outcome", explained)
	}
}
