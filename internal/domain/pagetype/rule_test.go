package pagetype_test

import (
	stderrors "errors"
	"slices"
	"strconv"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/pagetype"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func ids() func() string {
	next := 0
	return func() string {
		next++
		return "r" + strconv.Itoa(next)
	}
}

func aRule(typeID string, conditions pagetype.Conditions) pagetype.Rule {
	return pagetype.Rule{
		ID: "r1", SiteID: "s1", Kind: pagetype.KindRule, Position: 1, Name: "A rule", Enabled: true, TypeID: typeID,
		Conditions: conditions, CreatedAt: stamp, UpdatedAt: stamp,
	}
}

func TestARuleIsCheckedWhenItIsSaved(t *testing.T) {
	t.Parallel()

	set := pagetype.NewSet(append(fresh("s1"), legacy("s2")...), nil)
	yes, negative, two, one := true, -1, 2, 1
	guide := "s1-guide"
	cases := map[string]struct {
		rule  pagetype.Rule
		field string
	}{
		"a rule with no condition": {rule: aRule(guide, pagetype.Conditions{}), field: "conditions"},
		"a blank name": {rule: func() pagetype.Rule {
			rule := aRule(guide, pagetype.Conditions{Technical: &yes})
			rule.Name = " "
			return rule
		}(), field: "name"},
		"another site's type":       {rule: aRule("s2-hub", pagetype.Conditions{Technical: &yes}), field: "typeId"},
		"an unknown type":           {rule: aRule("gone", pagetype.Conditions{Technical: &yes}), field: "typeId"},
		"a pattern with a bracket":  {rule: aRule(guide, pagetype.Conditions{URLs: []pagetype.URLMatch{{Mode: pagetype.URLGlob, Value: "/blog/[a-z]/"}}}), field: "conditions.urls[0].value"},
		"a pattern off the root":    {rule: aRule(guide, pagetype.Conditions{URLs: []pagetype.URLMatch{{Mode: pagetype.URLGlob, Value: "blog/*"}}}), field: "conditions.urls[0].value"},
		"a ** inside a part":        {rule: aRule(guide, pagetype.Conditions{URLs: []pagetype.URLMatch{{Mode: pagetype.URLGlob, Value: "/blog**/"}}}), field: "conditions.urls[0].value"},
		"a start that is no path":   {rule: aRule(guide, pagetype.Conditions{URLs: []pagetype.URLMatch{{Mode: pagetype.URLPrefix, Value: "blog/*"}}}), field: "conditions.urls[0].value"},
		"a word with a slash":       {rule: aRule(guide, pagetype.Conditions{URLs: []pagetype.URLMatch{{Mode: pagetype.URLSegment, Value: "a/b"}}}), field: "conditions.urls[0].value"},
		"an unknown way to match":   {rule: aRule(guide, pagetype.Conditions{URLs: []pagetype.URLMatch{{Mode: "regex", Value: "x"}}}), field: "conditions.urls[0].mode"},
		"an empty slug":             {rule: aRule(guide, pagetype.Conditions{Slugs: []string{""}}), field: "conditions.slugs[0]"},
		"an inverted level":         {rule: aRule(guide, pagetype.Conditions{Level: &pagetype.Range{Min: &two, Max: &one}}), field: "conditions.level"},
		"a negative word count":     {rule: aRule(guide, pagetype.Conditions{Words: &pagetype.Range{Min: &negative}}), field: "conditions.words"},
		"an empty range":            {rule: aRule(guide, pagetype.Conditions{Words: &pagetype.Range{}}), field: "conditions.words"},
		"products asked of none":    {rule: aRule(guide, pagetype.Conditions{Listing: &pagetype.Listing{MinProducts: 2}}), field: "conditions.listing.minProducts"},
		"an unknown schema":         {rule: aRule(guide, pagetype.Conditions{Schema: []pagetype.SchemaType{"Recipe"}}), field: "conditions.schema[0]"},
		"a page and a type above":   {rule: aRule(guide, pagetype.Conditions{Ancestor: &pagetype.Ancestor{PageID: "p1", TypeID: guide}}), field: "conditions.ancestor"},
		"nothing above":             {rule: aRule(guide, pagetype.Conditions{Ancestor: &pagetype.Ancestor{}}), field: "conditions.ancestor"},
		"another site's type above": {rule: aRule(guide, pagetype.Conditions{Ancestor: &pagetype.Ancestor{TypeID: "s2-hub"}}), field: "conditions.ancestor.typeId"},
		"a fallback with a condition": {rule: func() pagetype.Rule {
			rule := aRule(guide, pagetype.Conditions{Technical: &yes})
			rule.Kind = pagetype.KindFallback
			return rule
		}(), field: "conditions"},
		"a fallback that types pages as products": {rule: func() pagetype.Rule {
			rule := aRule("s1-product", pagetype.Conditions{})
			rule.Kind = pagetype.KindFallback
			return rule
		}(), field: "typeId"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := tc.rule.Validate(set)
			if !errors.IsCode(err, errors.Invalid) {
				t.Fatalf("Validate = %v, want INVALID", err)
			}
			if got := fieldOf(err); got != tc.field {
				t.Errorf("the refusal names %q, want %q", got, tc.field)
			}
		})
	}

	for name, rule := range map[string]pagetype.Rule{
		"a glob":         aRule(guide, pagetype.Conditions{URLs: []pagetype.URLMatch{{Mode: pagetype.URLGlob, Value: "/blog/**/*-guide/"}}}),
		"a start":        aRule(guide, pagetype.Conditions{URLs: []pagetype.URLMatch{{Mode: pagetype.URLPrefix, Value: "/blog/"}}}),
		"a level":        aRule(guide, pagetype.Conditions{Level: &pagetype.Range{Min: &one}}),
		"no listing":     aRule(guide, pagetype.Conditions{Listing: &pagetype.Listing{}}),
		"a type above":   aRule(guide, pagetype.Conditions{Ancestor: &pagetype.Ancestor{TypeID: guide}}),
		"every default":  pagetype.DefaultRules(set, "s1", stamp, ids())[9],
		"a fallback":     pagetype.DefaultRules(set, "s1", stamp, ids())[14],
		"a page's words": aRule(guide, pagetype.Conditions{Words: &pagetype.Range{Max: &two}, MinH2: 1}),
	} {
		if err := rule.Validate(set); err != nil {
			t.Errorf("%s: Validate = %v", name, err)
		}
	}
}

func fieldOf(err error) string {
	var kerr *errors.Error
	if !stderrors.As(err, &kerr) {
		return ""
	}
	field, named := kerr.Details["field"].(string)
	if !named {
		return ""
	}
	return field
}

func TestEveryDefaultRuleIsValidAndPointsAtTheSitesOwnType(t *testing.T) {
	t.Parallel()

	for name, types := range map[string][]pagetype.Type{"a 2.6 site": fresh("s1"), "an upgraded site": append(legacy("s1"), fresh("s1")[:4]...)} {
		set := pagetype.NewSet(types, nil)
		rules := pagetype.DefaultRules(set, "s1", stamp, ids())
		if len(rules) != 16 {
			t.Fatalf("%s: %d default rules, want fifteen and the fallback", name, len(rules))
		}
		for i := range rules {
			if err := rules[i].Validate(set); err != nil {
				t.Errorf("%s: rule %d %q: %v", name, rules[i].Position, rules[i].Name, err)
			}
			if rules[i].Position != i+1 {
				t.Errorf("%s: rule %q sits at %d, want %d", name, rules[i].Name, rules[i].Position, i+1)
			}
		}
		if last := rules[len(rules)-1]; last.Kind != pagetype.KindFallback || last.TypeID != "s1-info" {
			t.Errorf("%s: the last rule = %+v, want the fallback to Info", name, last)
		}
		if under := rules[11].Conditions.Ancestor; under == nil || under.TypeID != "s1-guide" {
			t.Errorf("%s: under a guide names %+v, want the site's Guide type", name, under)
		}
	}

	renamed := legacy("s1")
	renamed[0].Name = "Category"
	upgraded := slices.Concat(renamed, []pagetype.Type{fresh("s1")[0], fresh("s1")[2], fresh("s1")[3]})
	rules := pagetype.DefaultRules(pagetype.NewSet(upgraded, nil), "s1", stamp, ids())
	if rules[7].TypeID != "s1-hub" {
		t.Errorf("the category rule on a site whose Hub the client renamed Category = %s, want that hub", rules[7].TypeID)
	}
}

func TestAConditionsListIsTrimmedLoweredAndDeduplicated(t *testing.T) {
	t.Parallel()

	got := pagetype.Conditions{
		PostTypes: []string{" Post ", "post", ""},
		Slugs:     []string{"About", "about"},
		URLs:      []pagetype.URLMatch{{Mode: pagetype.URLPrefix, Value: "Blog/"}, {Mode: pagetype.URLPrefix, Value: "/blog/"}},
		Schema:    []pagetype.SchemaType{pagetype.SchemaHowTo, pagetype.SchemaHowTo},
	}.Normalized()
	if len(got.PostTypes) != 1 || got.PostTypes[0] != "post" || len(got.Slugs) != 1 || got.Slugs[0] != "about" {
		t.Errorf("the lists = %v %v", got.PostTypes, got.Slugs)
	}
	if len(got.URLs) != 1 || got.URLs[0].Value != "/blog/" || len(got.Schema) != 1 {
		t.Errorf("the addresses and schema = %+v %v", got.URLs, got.Schema)
	}
	if empty := (pagetype.Conditions{PostTypes: []string{" "}}).Normalized(); !empty.Empty() {
		t.Errorf("a list of blanks = %+v, want no condition", empty)
	}
}
