package contextpack_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/contextpack"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/run"
)

func directedPack(t *testing.T) contextpack.Pack {
	t.Helper()

	s := replicaLike().snapshot(t)
	facts, err := s.Facts("bpc", planFor(t, s, "bpc"))
	if err != nil {
		t.Fatalf("Facts: %v", err)
	}
	facts.Directions = &contextpack.Directions{
		Goal: "Win the lab buyer who compares suppliers", Focus: "storage and stability", Tone: "plain, exact",
		Avoid: "any medical claim", Notes: "Lead with how it is shipped.", AddSections: []string{"Shipping", "Stability"},
		Keywords: []string{"bpc-157 storage"}, Anchors: []string{"research peptides"},
		MustLink: []contextpack.MustLinkRef{{Path: "/peptides/", Anchor: "research peptides"}}, MinWords: 600, MaxWords: 900,
		Reference:   "Shipped on dry ice.\n\nStable for 24 months at −20 °C.",
		StyleSample: "Short sentences. One fact each.",
	}
	body, _ := facts.Field(field.Body)
	return contextpack.Build(facts, body.Scope)
}

func TestTheRunsBriefIsAPrioritySectionOfItsOwn(t *testing.T) {
	t.Parallel()

	pack := directedPack(t)
	got := names(pack.Sections)
	if at := slices.Index(got, contextpack.SectionBrief); at != 2 || got[0] != contextpack.SectionPage || got[1] != contextpack.SectionRules {
		t.Fatalf("sections = %v, want the brief right after the page and the field's rules", got)
	}
	brief := sectionOf(t, pack, contextpack.SectionBrief)
	if brief.Priority != 1 || brief.Heading != "The run's brief" {
		t.Fatalf("the brief = %+v", brief)
	}
	text := strings.Join(brief.Lines, "\n")
	for _, want := range []string{
		"Goal: Win the lab buyer who compares suppliers", "Focus: storage and stability", "Tone: plain, exact",
		"Avoid: any medical claim", "For this page: Lead with how it is shipped.", "Add these sections: Shipping; Stability",
		"Use these keywords: bpc-157 storage", "Prefer these anchors: research peptides",
		"Link to /peptides/ with the anchor “research peptides”", "Length for this run: 600 to 900 words",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("the brief lacks %q:\n%s", want, text)
		}
	}
	reference, style := sectionOf(t, pack, contextpack.SectionReference), sectionOf(t, pack, contextpack.SectionStyle)
	if reference.Priority != 2 || len(reference.Items) != 2 || style.Priority != 2 || len(style.Lines) != 1 {
		t.Fatalf("the reference = %+v, the style = %+v", reference, style)
	}
	if at := slices.Index(got, contextpack.SectionReference); at < slices.Index(got, contextpack.SectionCurrent) {
		t.Errorf("the reference text comes before the current value: %v", got)
	}
}

func TestFitTrimsTheReferenceBeforeTheCardAndNeverTheBrief(t *testing.T) {
	t.Parallel()

	pack := directedPack(t)
	kept := 0
	for _, section := range pack.Sections {
		if section.Priority <= 2 && section.Name != contextpack.SectionReference && section.Name != contextpack.SectionStyle {
			kept += section.Tokens
		}
	}
	fitted, err := contextpack.Fit(pack, kept+kept/10)
	if err != nil {
		t.Fatalf("Fit: %v", err)
	}
	left := names(fitted.Sections)
	for _, held := range []contextpack.SectionName{contextpack.SectionBrief, contextpack.SectionCard} {
		if !slices.Contains(left, held) {
			t.Errorf("Fit dropped %s before the reference and the style: %v", held, left)
		}
	}
	if slices.Contains(left, contextpack.SectionStyle) {
		t.Errorf("Fit kept the style sample over the card: %v", left)
	}
	if _, err = contextpack.Fit(pack, 10); err == nil {
		t.Error("a budget below the brief was taken")
	}
}

func TestARunsBriefAndAPagesDirectiveBecomeTheWritersDirections(t *testing.T) {
	t.Parallel()

	index := replicaLike().snapshot(t).Index
	if got := contextpack.DirectionsOf(run.Brief{}, run.Directive{PageID: "bpc"}, index); got != nil {
		t.Fatalf("no brief and an empty directive gave %+v", got)
	}
	got := contextpack.DirectionsOf(
		run.Brief{Goal: "sell", Reference: "dry ice"},
		run.Directive{PageID: "bpc", Notes: "storage", MustLink: []run.MustLink{{PageID: "hub", Anchor: "peptides"}, {PageID: "gone"}}, MaxWords: 900},
		index,
	)
	if got == nil || got.Goal != "sell" || got.Reference != "dry ice" || got.Notes != "storage" || got.MaxWords != 900 {
		t.Fatalf("the directions = %+v", got)
	}
	if len(got.MustLink) != 1 || got.MustLink[0].Path != "/peptides/" || got.MustLink[0].Anchor != "peptides" {
		t.Errorf("the must-link targets = %+v, want the hub by its path and none for a page the site lacks", got.MustLink)
	}
}

func TestTheBriefReadsTheSameWhereNoPackCarriesIt(t *testing.T) {
	t.Parallel()

	facts := contextpack.Facts{Directions: &contextpack.Directions{
		Goal: "Win the lab buyer", Notes: "Lead with how it is shipped.",
		MustLink: []contextpack.MustLinkRef{{Path: "/peptides/", Anchor: "research peptides"}},
	}}
	text := facts.BriefText()
	for _, want := range []string{"THE RUN'S BRIEF\n", "Goal: Win the lab buyer\n", "For this page: Lead with how it is shipped.\n",
		"Link to /peptides/ with the anchor “research peptides”\n"} {
		if !strings.Contains(text, want) {
			t.Errorf("the brief's text lacks %q:\n%s", want, text)
		}
	}
	if got := (contextpack.Facts{}).BriefText(); got != "" {
		t.Errorf("a run without a brief reads %q, want nothing", got)
	}
}

func TestNoBriefAddsNoSection(t *testing.T) {
	t.Parallel()

	for _, section := range hubPack(t).Sections {
		if section.Name == contextpack.SectionBrief || section.Name == contextpack.SectionReference || section.Name == contextpack.SectionStyle {
			t.Fatalf("a run without a brief carries %s", section.Name)
		}
	}
}
