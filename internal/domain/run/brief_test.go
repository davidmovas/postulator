package run_test

import (
	stderrors "errors"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func TestARunsBriefAndDirectivesAreHeldToTheirLimits(t *testing.T) {
	t.Parallel()

	long := func(runes int) string { return strings.Repeat("ж", runes) }
	cases := []struct {
		name       string
		brief      run.Brief
		directives []run.Directive
		field      string
	}{
		{name: "an empty brief and no directives"},
		{
			name:  "a whole brief within its limits",
			brief: run.Brief{Goal: long(1000), Focus: "dosing", Tone: "plain", Avoid: "medical claims", Reference: long(20000), StyleSample: long(4000)},
			directives: []run.Directive{{
				PageID: "p1", Notes: long(2000), AddSections: []string{"Storage"}, Keywords: []string{"bpc-157 storage"},
				Anchors: []string{"how to store BPC-157"}, MustLink: []run.MustLink{{PageID: "p2", Anchor: "TB-500"}}, MinWords: 600, MaxWords: 900,
			}},
		},
		{name: "a goal too long", brief: run.Brief{Goal: long(1001)}, field: "brief.goal"},
		{name: "a reference too long", brief: run.Brief{Reference: long(20001)}, field: "brief.reference"},
		{name: "a style sample too long", brief: run.Brief{StyleSample: long(4001)}, field: "brief.styleSample"},
		{name: "a directive with no page", directives: []run.Directive{{Notes: "x"}}, field: "directives.pageId"},
		{name: "two directives for one page", directives: []run.Directive{{PageID: "p1"}, {PageID: "p1"}}, field: "directives.pageId"},
		{name: "a length upside down", directives: []run.Directive{{PageID: "p1", MinWords: 900, MaxWords: 600}}, field: "directives.maxWords"},
		{name: "a must-link with no page", directives: []run.Directive{{PageID: "p1", MustLink: []run.MustLink{{Anchor: "x"}}}}, field: "directives.mustLink"},
		{name: "a page that links to itself", directives: []run.Directive{{PageID: "p1", MustLink: []run.MustLink{{PageID: "p1"}}}}, field: "directives.mustLink"},
		{name: "too many sections", directives: []run.Directive{{PageID: "p1", AddSections: make([]string, 11)}}, field: "directives.addSections"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := run.ValidateBrief(tc.brief, tc.directives)
			if tc.field == "" {
				if err != nil {
					t.Fatalf("ValidateBrief: %v", err)
				}
				return
			}
			var refused *errors.Error
			if !stderrors.As(err, &refused) || refused.Code != errors.Invalid || refused.Details["field"] != tc.field {
				t.Fatalf("ValidateBrief = %v, want INVALID on %s", err, tc.field)
			}
		})
	}
}

func TestAStepsInputHashMovesWithTheRunsBriefOnly(t *testing.T) {
	t.Parallel()

	params := map[string]any{"tone": "plain"}
	if got := run.HashedParams(params, run.Brief{}, run.Directive{PageID: "p1"}); len(got) != 1 {
		t.Fatalf("a run without a brief hashed %v, want its params alone so its old hashes still match", got)
	}
	briefed := run.HashedParams(params, run.Brief{Goal: "sell"}, run.Directive{PageID: "p1", Notes: "storage"})
	if briefed["runBrief"] == "" || briefed["pageDirective"] == "" || len(params) != 1 {
		t.Fatalf("a briefed run hashed %v and left the params %v", briefed, params)
	}
	plain, err := run.InputHash("generate_body", params, nil, 1, run.NewCheckpoint())
	if err != nil {
		t.Fatalf("InputHash: %v", err)
	}
	with, err := run.InputHash("generate_body", briefed, nil, 1, run.NewCheckpoint())
	if err != nil || with == plain {
		t.Fatalf("the brief left the input hash at %s (%v)", with, err)
	}
}

func TestABriefAndADirectiveHashOnlyWhenTheyHoldSomething(t *testing.T) {
	t.Parallel()

	if (run.Brief{}).Hash() != "" || (run.Directive{PageID: "p1"}).Hash() != "" {
		t.Fatal("an empty brief or directive changed a step's input hash")
	}
	one, other := run.Brief{Goal: "sell"}.Hash(), run.Brief{Goal: "teach"}.Hash()
	if one == "" || one == other {
		t.Fatalf("two briefs hash %q and %q", one, other)
	}
	if (run.Directive{PageID: "p1", Notes: "n"}).Hash() == "" || !(run.Directive{PageID: "p1"}).Empty() {
		t.Fatal("a directive with notes does not hash, or one without anything is not empty")
	}
}
