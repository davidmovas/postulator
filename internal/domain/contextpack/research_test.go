package contextpack_test

import (
	"slices"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/contextpack"
	"github.com/davidmovas/postulator/internal/domain/field"
)

func TestTheResearchIsAFactPerItemRightAfterTheCard(t *testing.T) {
	t.Parallel()

	researched := &content.Research{
		Text:    "## Dosage\n- Caffeine peaks within an hour [who.int]\n\n- Its half life is about five hours [nih.gov]\n",
		Sources: []content.Source{{URL: "https://www.who.int/caffeine", Title: "WHO"}},
	}
	cases := []struct {
		name  string
		facts contextpack.Facts
		items []string
	}{
		{name: "a page researched", facts: contextpack.Facts{Research: researched},
			items: []string{"Dosage", "Caffeine peaks within an hour [who.int]", "Its half life is about five hours [nih.gov]"}},
		{name: "a page nobody researched", facts: contextpack.Facts{}},
		{name: "a search that found nothing", facts: contextpack.Facts{Research: &content.Research{Text: "\n \n"}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			pack := contextpack.Build(tc.facts, field.Scope{Key: field.Body, Enabled: true})
			at := slices.IndexFunc(pack.Sections, func(section contextpack.Section) bool {
				return section.Name == contextpack.SectionResearch
			})
			if tc.items == nil {
				if at >= 0 {
					t.Fatalf("the pack carries a research section: %+v", pack.Sections[at])
				}
				return
			}
			if at < 0 {
				t.Fatalf("the pack carries no research section: %+v", pack.Sections)
			}
			section := pack.Sections[at]
			if !slices.Equal(section.Items, tc.items) || section.Priority != 2 || section.Tokens <= 0 {
				t.Errorf("the research section = %+v, want one item per fact at priority 2", section)
			}
		})
	}
}
