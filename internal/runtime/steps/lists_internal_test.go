package steps

import (
	"slices"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/contextpack"
	"github.com/davidmovas/postulator/internal/domain/field"
)

var offered = []contextpack.Candidate{
	{WPID: 21, Name: "TB-500", Path: "/product/tb/"},
	{WPID: 30, Name: "BPC-157 Capsules", Path: "/product/caps/"},
}

func TestAListAnswerKeepsTheCandidatesItNamesInItsOrder(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		answer string
		want   string
	}{
		{name: "two candidates", answer: "30, 21", want: "30,21"},
		{name: "an id that is no candidate", answer: "30, 999, 21", want: "30,21"},
		{name: "a repeat and loose words", answer: "ids: 21 and 21; 30.", want: "21,30"},
		{name: "nothing it may choose", answer: "999", want: ""},
		{name: "no id at all", answer: "none fits", want: ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := listAnswer(tc.answer, offered); got != tc.want {
				t.Errorf("listAnswer(%q) = %q, want %q", tc.answer, got, tc.want)
			}
		})
	}
}

func TestAListIsAskedOnlyWithCandidatesAndTheyAreNamed(t *testing.T) {
	t.Parallel()

	asked := []askedScope{
		{Key: field.Upsells, Scope: field.Scope{Key: field.Upsells}},
		{Key: field.Excerpt, Scope: field.Scope{Key: field.Excerpt}},
	}
	kept, skipped := withCandidates(nil, asked, []FieldSkip{})
	if len(kept) != 1 || kept[0].Key != field.Excerpt || len(skipped) != 1 || skipped[0].Key != string(field.Upsells) {
		t.Errorf("without candidates: asked %+v, skipped %+v, want the list left out with its reason", kept, skipped)
	}
	kept, skipped = withCandidates(offered, asked, []FieldSkip{})
	if len(kept) != 2 || len(skipped) != 0 {
		t.Fatalf("with candidates: asked %+v, skipped %+v", kept, skipped)
	}
	rules := candidateRules(offered)
	joined := strings.Join(rules, "\n")
	for _, want := range []string{"21: TB-500 (/product/tb/)", "30: BPC-157 Capsules (/product/caps/)", "only"} {
		if !strings.Contains(joined, want) {
			t.Errorf("the rules lack %q:\n%s", want, joined)
		}
	}
	if !slices.Contains(listFields, field.CrossSells) {
		t.Error("cross-sells are not a list field")
	}
}
