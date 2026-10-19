package steps

import (
	"io/fs"
	"regexp"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/keyword"
	"github.com/davidmovas/postulator/internal/domain/note"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/template"
)

const fullPicture = "THE PAGE\nPath: /peptides/\nTitle: Peptides\n\nTHE TRAIL, NEAREST PARENT FIRST\n- Home (/)\n"

var definedUser = regexp.MustCompile(`\{\{define "([a-z_]+)\.user"\}\}`)

func definedPrompts(t *testing.T) []string {
	t.Helper()

	names := make([]string, 0)
	err := fs.WalkDir(promptFS, "prompts", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		body, readErr := fs.ReadFile(promptFS, path)
		if readErr != nil {
			return readErr
		}
		for _, match := range definedUser.FindAllStringSubmatch(string(body), -1) {
			names = append(names, match[1])
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk the prompts: %v", err)
	}
	return names
}

func fullBrief() content.Brief {
	volume := 1200
	return content.Brief{
		Title: "Peptides", H1: "Research peptides", PrimaryKeyword: "peptides", PlannedTitle: true,
		Keywords:  []content.BriefKeyword{{Rank: 1, Text: "peptides", Volume: &volume, Required: true}, {Rank: 2, Text: "research peptides"}},
		TitleRule: true, H1Rule: true, LeadRule: true,
		Sections: []content.BriefSection{{Slot: 1, Heading: "What they are", Intent: "Define them", Required: true, Words: 200, PrimaryInHeading: true}},
		Phrases:  []content.BriefPhrase{{Text: "peptides", Lead: true, Why: "the primary keyword opens the page"}, {Text: "home", Within: 2, Why: "the parent"}},
		Children: []string{"BPC-157"},
		Product: &content.BriefProduct{
			ShortDescription: true, ShortWords: 40, ShortIntent: "Sell it", ShortKeyword: true,
			Specifications: []content.BriefSpecification{{Name: "Purity", Intent: "As tested"}},
		},
	}
}

func TestEveryPromptExecutes(t *testing.T) {
	t.Parallel()

	page := pagemap.Page{ID: "page", Path: "/peptides/", Title: "Peptides", Notes: []note.Note{{Label: "Tone", Text: "Plain"}}}
	entity := graph.Entity{Name: "Peptides", Intent: "commercial"}
	spec := template.TemplateSpec{Tone: "factual", Sections: []template.Section{{Heading: "What they are", Required: true}}}
	spec.Length.Min, spec.Length.Max = 600, 1200
	spec.MetaRules = template.MetaRules{TitlePattern: "{entityName} | {siteName}", DescriptionMax: 155}
	spec.KeywordRules.PrimaryInTitle, spec.KeywordRules.MaxDensity = true, 0.02

	data := map[string]any{
		NameGenerateBody: bodyPrompt{Page: page, Type: "Hub", Entity: entity, Spec: spec, Brief: fullBrief(), Product: true, Picture: fullPicture},
		NameGenerateMeta: metaPrompt{
			Page: page, Entity: entity, Keywords: keyword.Of("peptides", "research peptides"), Spec: spec,
			Draft:    content.ContentDraft{Title: "Peptides", H1: "Research peptides", Summary: "All about them."},
			SiteName: "Lab", Canonical: "https://lab.example.com/peptides/", Pattern: "Peptides | Lab",
			Picture: fullPicture, Description: []string{"Field: SEO description", "Length: at most 155 characters"},
		},
		NameGenerateFields: fieldsPrompt{
			Page: page, Picture: fullPicture,
			Asked: []askedField{{Key: "excerpt", Name: "Excerpt", Rules: []string{"Field: Excerpt", "Length: at most 160 characters"}}},
		},
		NameRepairLinks: repairPrompt{
			Page: page, Entity: entity, Phrase: "home", Why: "the parent", Paragraph: "Peptides are short chains.",
			Identity: "Peptides (/peptides/, Hub)", About: "Home (/): Entity for Home",
		},
	}

	for _, name := range definedPrompts(t) {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			prompt, covered := data[name]
			if !covered {
				t.Fatalf("the prompt %s has no full case here", name)
			}
			system, user, err := prompts.Render(name, prompt)
			if err != nil {
				t.Fatalf("Render(%s): %v", name, err)
			}
			if system == "" || user == "" {
				t.Fatalf("%s rendered an empty half", name)
			}
			if strings.Contains(system+user, "<no value>") {
				t.Errorf("%s reads a field its data does not have:\n%s\n%s", name, system, user)
			}
			if name != NameRepairLinks && !strings.Contains(user, fullPicture) {
				t.Errorf("%s does not carry the full picture:\n%s", name, user)
			}
		})
	}
}

func TestACacheKeyIsNamedOnlyForAStablePrefixLongEnoughToCache(t *testing.T) {
	t.Parallel()

	if got := cacheKey(NameGenerateBody, "site-1", "You write pages."); got != "" {
		t.Errorf("a short prefix names the cache key %q, want none", got)
	}
	long := strings.Repeat("Every page on the site is written from the full picture of it. ", 120)
	if got := cacheKey(NameGenerateBody, "site-1", long); got != NameGenerateBody+":site-1" {
		t.Errorf("a long prefix names the cache key %q, want the step and the site", got)
	}
}
