package content_test

import (
	"slices"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/content"
)

func TestAResearchReadsItsFactsWithoutTheMarkup(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		text string
		want []string
	}{
		{name: "nothing found", text: "", want: []string{}},
		{name: "blank lines only", text: "\n  \n", want: []string{}},
		{
			name: "bullets, numbers and headings",
			text: "- one\n* two\n1. three\n  4) four  \n### Heading\nplain",
			want: []string{"one", "two", "three", "four", "Heading", "plain"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			research := content.Research{Text: tc.text}
			if got := research.Facts(); !slices.Equal(got, tc.want) {
				t.Errorf("Facts(%q) = %q, want %q", tc.text, got, tc.want)
			}
			if research.Empty() != (len(tc.want) == 0) {
				t.Errorf("Empty() = %t for %q", research.Empty(), tc.text)
			}
		})
	}
}

const (
	manualURL   = "https://assets.thermofisher.com/TFS-Assets/LSG/manuals/custompeptide_man.pdf?utm_source=openai"
	dailymedURL = "https://dailymed.nlm.nih.gov/dailymed/downloadpdffile.cfm?setId=99b0ef9b-600a-40dc-e053-2a95a90aa07e&utm_source=openai"
	pmcURL      = "https://pmc.ncbi.nlm.nih.gov/articles/PMC4830481/"
)

func researchedSources() []content.Source {
	return []content.Source{
		{URL: manualURL, Title: "Custom Peptide Storage and Dissolution"},
		{URL: dailymedURL, Title: "BACTERIOSTATIC WATER FOR INJECTION, USP 30mL VIAL"},
		{URL: pmcURL},
		{URL: "https://www.cdc.gov/injection-safety/hcp/clinical-safety/?%3Butm_source=openai&CDC_AAref_Val=x"},
		{URL: "https://www.cdc.gov/injection-safety/hcp/clinical-safety/"},
	}
}

func TestAFactKeepsTheAddressOfTheSourceItCites(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		line string
		want content.Citation
	}{
		{
			name: "a markdown citation and the domain the prompt asks for",
			line: "- Reconstitution is product-specific. ([assets.thermofisher.com](" + manualURL + ")) [thermofisher.com]",
			want: content.Citation{
				Fact: "Reconstitution is product-specific.", Tag: "thermofisher.com",
				URL: "https://assets.thermofisher.com/TFS-Assets/LSG/manuals/custompeptide_man.pdf",
			},
		},
		{
			name: "the tracking parameter goes and the others stay in their order",
			line: "- Inspect the vial before use. ([dailymed.nlm.nih.gov](" + dailymedURL + ")) [dailymed.nlm.nih.gov]",
			want: content.Citation{
				Fact: "Inspect the vial before use.", Tag: "dailymed.nlm.nih.gov",
				URL: "https://dailymed.nlm.nih.gov/dailymed/downloadpdffile.cfm?setId=99b0ef9b-600a-40dc-e053-2a95a90aa07e",
			},
		},
		{
			name: "a source the search returned without a tracking parameter",
			line: "- Store aliquots frozen. ([pmc.ncbi.nlm.nih.gov](https://pmc.ncbi.nlm.nih.gov/articles/PMC4830481)) [pmc.ncbi.nlm.nih.gov]",
			want: content.Citation{Fact: "Store aliquots frozen.", Tag: "pmc.ncbi.nlm.nih.gov", URL: pmcURL},
		},
		{
			name: "no citation keeps the tag and links nothing",
			line: "- Volume is the amount divided by the concentration. [calculator]",
			want: content.Citation{Fact: "Volume is the amount divided by the concentration.", Tag: "calculator"},
		},
		{
			name: "an address the search never returned links nothing",
			line: "- Peptides keep for years. ([example.org](https://example.org/peptides?utm_source=openai)) [example.org]",
			want: content.Citation{Fact: "Peptides keep for years.", Tag: "example.org"},
		},
		{
			name: "the address the prompt asks for, at the end in brackets",
			line: "- Discard a punctured vial within 28 days. [https://www.cdc.gov/injection-safety/hcp/clinical-safety/]",
			want: content.Citation{
				Fact: "Discard a punctured vial within 28 days.", Tag: "cdc.gov",
				URL: "https://www.cdc.gov/injection-safety/hcp/clinical-safety/",
			},
		},
		{
			name: "the address in parentheses",
			line: "- Discard a punctured vial within 28 days. (https://www.cdc.gov/injection-safety/hcp/clinical-safety)",
			want: content.Citation{
				Fact: "Discard a punctured vial within 28 days.", Tag: "cdc.gov",
				URL: "https://www.cdc.gov/injection-safety/hcp/clinical-safety/",
			},
		},
		{
			name: "an address the search never read keeps only its site",
			line: "- Peptides keep for years. [https://example.org/peptides]",
			want: content.Citation{Fact: "Peptides keep for years.", Tag: "example.org"},
		},
		{
			name: "a heading is a line of its own",
			line: "## Step-by-Step Instructions",
			want: content.Citation{Fact: "Step-by-Step Instructions"},
		},
		{
			name: "a research written before the citations were read",
			line: "- Bring the vial to room temperature first. [merckmillipore.com]",
			want: content.Citation{Fact: "Bring the vial to room temperature first.", Tag: "merckmillipore.com"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			research := content.Research{Text: tc.line, Sources: researchedSources()}
			got := research.Citations()
			if len(got) != 1 || got[0] != tc.want {
				t.Fatalf("Citations(%q) = %+v, want %+v", tc.line, got, tc.want)
			}
		})
	}
}

func TestTheFactsReadWithoutTheCitationsTheyCarried(t *testing.T) {
	t.Parallel()

	research := content.Research{
		Text: "## Introduction\n- Reconstitution is product-specific. ([assets.thermofisher.com](" + manualURL +
			")) [thermofisher.com]\n- Volume is the amount divided by the concentration. [calculator]",
		Sources: researchedSources(),
	}
	want := []string{
		"Introduction",
		"Reconstitution is product-specific. [thermofisher.com]",
		"Volume is the amount divided by the concentration. [calculator]",
	}
	if got := research.Facts(); !slices.Equal(got, want) {
		t.Errorf("Facts() = %q, want %q", got, want)
	}
}

func TestASourceAddressIsComparedWithoutWhatTracksOrPointsInsideIt(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		raw   string
		href  string
		valid bool
	}{
		{name: "tracking only", raw: manualURL, href: "https://assets.thermofisher.com/TFS-Assets/LSG/manuals/custompeptide_man.pdf", valid: true},
		{name: "a fragment", raw: "https://www.osha.gov/laws#scope", href: "https://www.osha.gov/laws", valid: true},
		{name: "every tracking parameter", raw: "https://who.int/a?utm_medium=x&id=4&utm_campaign=y", href: "https://who.int/a?id=4", valid: true},
		{name: "a tracking parameter written escaped", raw: "https://cdc.gov/a/?%3Butm_source=openai&ref=x", href: "https://cdc.gov/a/?ref=x", valid: true},
		{name: "a host in capitals", raw: "https://WHO.int/a", href: "https://who.int/a", valid: true},
		{name: "not the web", raw: "mailto:lab@who.int", valid: false},
		{name: "no host", raw: "https:///a", valid: false},
		{name: "not an address", raw: "who.int/a", valid: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			href, valid := content.SourceHref(tc.raw)
			if valid != tc.valid || href != tc.href {
				t.Errorf("SourceHref(%q) = %q, %t, want %q, %t", tc.raw, href, valid, tc.href, tc.valid)
			}
		})
	}
}
