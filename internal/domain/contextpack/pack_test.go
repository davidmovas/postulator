package contextpack_test

import (
	stderrors "errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/contextpack"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func hubPack(t *testing.T) contextpack.Pack {
	t.Helper()

	s := replicaLike().snapshot(t)
	facts, err := s.Facts("hub", planFor(t, s, "hub"))
	if err != nil {
		t.Fatalf("Facts: %v", err)
	}
	body, _ := facts.Field(field.Body)
	return contextpack.Build(facts, body.Scope)
}

func sectionOf(t *testing.T, pack contextpack.Pack, name contextpack.SectionName) contextpack.Section {
	t.Helper()

	for _, section := range pack.Sections {
		if section.Name == name {
			return section
		}
	}
	t.Fatalf("the pack has no %s section", name)
	return contextpack.Section{}
}

func names(sections []contextpack.Section) []contextpack.SectionName {
	out := make([]contextpack.SectionName, 0, len(sections))
	for _, section := range sections {
		out = append(out, section.Name)
	}
	return out
}

func golden(t *testing.T, name, got string) {
	t.Helper()

	path := filepath.Join("testdata", name)
	if os.Getenv("POSTULATOR_UPDATE_GOLDENS") == "1" {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(path, []byte(got), 0o600); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v (POSTULATOR_UPDATE_GOLDENS=1 writes it)", path, err)
	}
	if got != string(want) {
		t.Errorf("%s differs from what was built:\n%s", path, got)
	}
}

func TestBuildOrdersTheSectionsByPriority(t *testing.T) {
	t.Parallel()

	pack := hubPack(t)
	want := []contextpack.SectionName{
		contextpack.SectionPage, contextpack.SectionRules, contextpack.SectionCard, contextpack.SectionCurrent,
		contextpack.SectionKeywords, contextpack.SectionTrail, contextpack.SectionNotes, contextpack.SectionTargets,
		contextpack.SectionProducts, contextpack.SectionSiblings, contextpack.SectionChildren, contextpack.SectionFields,
		contextpack.SectionGenerated,
	}
	if got := names(pack.Sections); !slices.Equal(got, want) {
		t.Fatalf("sections = %v, want %v", got, want)
	}
	priorities := map[contextpack.SectionName]int{
		contextpack.SectionPage: 1, contextpack.SectionRules: 1, contextpack.SectionCard: 2, contextpack.SectionCurrent: 2,
		contextpack.SectionKeywords: 3, contextpack.SectionTrail: 3, contextpack.SectionNotes: 3, contextpack.SectionTargets: 4,
		contextpack.SectionProducts: 4, contextpack.SectionSiblings: 5, contextpack.SectionChildren: 5,
		contextpack.SectionFields: 6, contextpack.SectionGenerated: 6,
	}
	total := 0
	for _, section := range pack.Sections {
		if section.Priority != priorities[section.Name] {
			t.Errorf("%s has priority %d, want %d", section.Name, section.Priority, priorities[section.Name])
		}
		if section.Tokens <= 0 {
			t.Errorf("%s counts no tokens", section.Name)
		}
		total += section.Tokens
	}
	if total != pack.Tokens {
		t.Errorf("pack counts %d tokens, its sections %d", pack.Tokens, total)
	}
	golden(t, "hub.body.txt", contextpack.Fitted{Sections: pack.Sections}.Text())
}

func TestTheCurrentBodyNamesWhatItHoldsBesideText(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		kind block.Kind
		raw  string
		want []string
	}{
		{
			name: "a classic body's shortcode", kind: block.Classic,
			raw:  "<p>Peptides that help repair.</p>\n\n[products limit=\"4\"]",
			want: []string{"Peptides that help repair.", "(a [products] shortcode, kept where it stands)"},
		},
		{
			name: "a shortcode block", kind: block.Blocks,
			raw:  "<!-- wp:shortcode -->[woocommerce_my_account]<!-- /wp:shortcode -->",
			want: []string{"(a [woocommerce_my_account] shortcode, kept where it stands)"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			f := replicaLike()
			f.bodies["healing"] = block.Snapshot{PageID: "healing", Side: block.Current, Kind: tc.kind, ObservedAt: stamp, Raw: tc.raw}
			s := f.snapshot(t)
			facts, err := s.Facts("healing", planFor(t, s, "healing"))
			if err != nil {
				t.Fatalf("Facts: %v", err)
			}
			body, _ := facts.Field(field.Body)
			current := sectionOf(t, contextpack.Build(facts, body.Scope), contextpack.SectionCurrent)
			if !slices.Equal(current.Lines, tc.want) {
				t.Errorf("the current value = %q, want %q", current.Lines, tc.want)
			}
		})
	}
}

func TestAProductListIsLessUrgentOnAnArticle(t *testing.T) {
	t.Parallel()

	s := replicaLike().snapshot(t)
	facts, err := s.Facts("hub", planFor(t, s, "hub"))
	if err != nil {
		t.Fatalf("Facts: %v", err)
	}
	facts.Page.Preset = "entity"
	body, _ := facts.Field(field.Body)
	if got := sectionOf(t, contextpack.Build(facts, body.Scope), contextpack.SectionProducts).Priority; got != 6 {
		t.Errorf("an article's products have priority %d, want 6", got)
	}
}

func TestAPresetIsWrittenForByItsRole(t *testing.T) {
	t.Parallel()

	s := replicaLike().snapshot(t)
	cases := map[pagetype.Preset]struct {
		role     string
		priority int
	}{
		pagetype.Landing:  {"a hub: it introduces its topic", 4},
		pagetype.Category: {"a hub: it introduces its topic", 4},
		pagetype.Guide:    {"an article about one entity", 6},
		pagetype.Info:     {"an article about one entity", 6},
	}
	for preset, tc := range cases {
		facts, err := s.Facts("hub", planFor(t, s, "hub"))
		if err != nil {
			t.Fatalf("Facts: %v", err)
		}
		facts.Page.Type, facts.Page.Preset = preset.DefaultName(), preset
		body, _ := facts.Field(field.Body)
		pack := contextpack.Build(facts, body.Scope)
		page := strings.Join(sectionOf(t, pack, contextpack.SectionPage).Lines, "\n")
		if !strings.Contains(page, "Page type: "+preset.DefaultName()+", "+tc.role) {
			t.Errorf("%s reads %q, want its role %q", preset, page, tc.role)
		}
		if got := sectionOf(t, pack, contextpack.SectionProducts).Priority; got != tc.priority {
			t.Errorf("%s's products have priority %d, want %d", preset, got, tc.priority)
		}
	}
}

func TestFitDropsTheLowestPriorityFirst(t *testing.T) {
	t.Parallel()

	pack := hubPack(t)
	tokens := func(sections ...contextpack.SectionName) int {
		total := 0
		for _, name := range sections {
			total += sectionOf(t, pack, name).Tokens
		}
		return total
	}
	lastChild := sectionOf(t, pack, contextpack.SectionChildren)

	cases := []struct {
		name    string
		budget  int
		gone    []contextpack.SectionName
		kept    []contextpack.SectionName
		trimmed contextpack.SectionName
	}{
		{
			name: "everything fits", budget: budgetFor(pack.Tokens),
			kept: names(pack.Sections),
		},
		{
			name:    "a list is trimmed from its tail before it goes",
			budget:  budgetFor(pack.Tokens - tokens(contextpack.SectionGenerated, contextpack.SectionFields) - 1),
			gone:    []contextpack.SectionName{contextpack.SectionGenerated, contextpack.SectionFields},
			kept:    []contextpack.SectionName{contextpack.SectionChildren, contextpack.SectionSiblings, contextpack.SectionTrail},
			trimmed: contextpack.SectionChildren,
		},
		{
			name: "siblings drop before the trail",
			budget: budgetFor(pack.Tokens - tokens(contextpack.SectionGenerated, contextpack.SectionFields,
				contextpack.SectionChildren, contextpack.SectionSiblings)),
			gone: []contextpack.SectionName{contextpack.SectionSiblings, contextpack.SectionChildren},
			kept: []contextpack.SectionName{contextpack.SectionTrail, contextpack.SectionProducts, contextpack.SectionTargets},
		},
		{
			name: "products drop before the keywords",
			budget: budgetFor(pack.Tokens - tokens(contextpack.SectionGenerated, contextpack.SectionFields,
				contextpack.SectionChildren, contextpack.SectionSiblings, contextpack.SectionProducts)),
			gone: []contextpack.SectionName{contextpack.SectionProducts},
			kept: []contextpack.SectionName{contextpack.SectionKeywords, contextpack.SectionTargets, contextpack.SectionTrail},
		},
		{
			name:   "only what every call carries is left",
			budget: budgetFor(tokens(contextpack.SectionPage, contextpack.SectionRules)),
			gone:   []contextpack.SectionName{contextpack.SectionCard, contextpack.SectionCurrent, contextpack.SectionKeywords},
			kept:   []contextpack.SectionName{contextpack.SectionPage, contextpack.SectionRules},
		},
		{
			name: "no budget keeps everything", budget: 0,
			kept: names(pack.Sections),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			fitted, err := contextpack.Fit(pack, tc.budget)
			if err != nil {
				t.Fatalf("Fit(%d): %v", tc.budget, err)
			}
			if tc.budget > 0 && fitted.Tokens > roomOf(tc.budget) {
				t.Errorf("fitted %d tokens into a budget of %d, whose room is %d", fitted.Tokens, tc.budget, roomOf(tc.budget))
			}
			held := names(fitted.Sections)
			for _, name := range tc.gone {
				if slices.Contains(held, name) {
					t.Errorf("%s survived a budget of %d", name, tc.budget)
				}
			}
			for _, name := range tc.kept {
				if !slices.Contains(held, name) {
					t.Errorf("%s was dropped from a budget of %d", name, tc.budget)
				}
			}
			dropped := 0
			for _, drop := range fitted.Dropped {
				dropped += drop.Tokens
				if drop.Section == tc.trimmed && drop.Whole {
					t.Errorf("%s went whole, want its tail trimmed", drop.Section)
				}
			}
			if pack.Tokens-dropped != fitted.Tokens {
				t.Errorf("the drops count %d of %d tokens, the fit keeps %d", dropped, pack.Tokens, fitted.Tokens)
			}
			if tc.trimmed != "" {
				section, _ := fitted.Section(tc.trimmed)
				if len(section.Items) != len(lastChild.Items)-1 || section.Items[0] != lastChild.Items[0] {
					t.Errorf("%s kept %v, want all but its last item", tc.trimmed, section.Items)
				}
			}
		})
	}
}

func roomOf(budget int) int {
	return budget - budget/20
}

func budgetFor(room int) int {
	budget := room
	for roomOf(budget) < room {
		budget++
	}
	return budget
}

func TestFitKeepsATwentiethOfTheBudgetAsHeadroom(t *testing.T) {
	t.Parallel()

	pack := hubPack(t)
	fitted, err := contextpack.Fit(pack, pack.Tokens)
	if err != nil {
		t.Fatalf("Fit(%d): %v", pack.Tokens, err)
	}
	if fitted.Tokens > roomOf(pack.Tokens) || len(fitted.Dropped) == 0 {
		t.Errorf("a pack of exactly its budget kept %d tokens and dropped %v, want it under %d, a twentieth kept for what the counter misses",
			fitted.Tokens, fitted.Dropped, roomOf(pack.Tokens))
	}
	whole, err := contextpack.Fit(pack, budgetFor(pack.Tokens))
	if err != nil || len(whole.Dropped) != 0 || whole.Budget != budgetFor(pack.Tokens) {
		t.Errorf("Fit(%d) = %d dropped, budget %d, %v; want the whole pack within its room", budgetFor(pack.Tokens), len(whole.Dropped), whole.Budget, err)
	}
}

func TestFitRefusesABudgetBelowWhatEveryCallCarries(t *testing.T) {
	t.Parallel()

	pack := hubPack(t)
	kept := sectionOf(t, pack, contextpack.SectionPage).Tokens + sectionOf(t, pack, contextpack.SectionRules).Tokens
	_, err := contextpack.Fit(pack, kept-1)
	if !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("Fit below the kept sections = %v, want Invalid", err)
	}
	refused, typed := stderrors.AsType[*errors.Error](err)
	if !typed || refused.Details["reason"] != contextpack.ReasonBudgetTooSmall || refused.Details["needed"] != kept {
		t.Errorf("refusal = %+v, want the reason and the tokens needed", refused)
	}
}
