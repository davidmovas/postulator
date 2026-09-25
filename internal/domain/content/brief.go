package content

import (
	"strings"

	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/template"
)

type BriefSection struct {
	Slot             int    `json:"slot"`
	Heading          string `json:"heading"`
	Intent           string `json:"intent"`
	Required         bool   `json:"required"`
	Words            int    `json:"words"`
	PrimaryInHeading bool   `json:"primaryInHeading"`
}

type BriefPhrase struct {
	Text   string `json:"text"`
	Lead   bool   `json:"lead"`
	Within int    `json:"within"`
	Why    string `json:"why"`
}

type Brief struct {
	Title          string         `json:"title"`
	H1             string         `json:"h1"`
	PlannedTitle   bool           `json:"plannedTitle"`
	PlannedH1      bool           `json:"plannedH1"`
	PrimaryKeyword string         `json:"primaryKeyword"`
	TitleRule      bool           `json:"titleRule"`
	H1Rule         bool           `json:"h1Rule"`
	LeadRule       bool           `json:"leadRule"`
	Sections       []BriefSection `json:"sections"`
	Phrases        []BriefPhrase  `json:"phrases"`
	Children       []string       `json:"children"`
}

func NewBrief(spec template.TemplateSpec, rules template.LinkRules, page pagemap.Page, entity graph.Entity, lc LinkContext) Brief {
	brief := Brief{
		Title:          strings.TrimSpace(page.Title),
		H1:             strings.TrimSpace(page.H1),
		PrimaryKeyword: strings.TrimSpace(entity.PrimaryKeyword),
		TitleRule:      spec.KeywordRules.PrimaryInTitle,
		H1Rule:         spec.KeywordRules.PrimaryInH1,
		LeadRule:       spec.KeywordRules.PrimaryInFirstParagraph,
		Sections:       make([]BriefSection, 0, len(spec.Sections)),
		Phrases:        make([]BriefPhrase, 0, len(lc.Targets)+1),
		Children:       make([]string, 0),
	}
	brief.PlannedTitle = brief.Title != ""
	brief.PlannedH1 = brief.H1 != ""

	for i := range spec.Sections {
		section := spec.Sections[i]
		brief.Sections = append(brief.Sections, BriefSection{
			Slot:             i + 1,
			Heading:          strings.TrimSpace(section.Heading),
			Intent:           strings.TrimSpace(section.Intent),
			Required:         section.Required,
			Words:            section.TargetWords,
			PrimaryInHeading: section.KeywordRules.PrimaryInHeading,
		})
	}

	if brief.LeadRule && brief.PrimaryKeyword != "" {
		brief.Phrases = append(brief.Phrases, BriefPhrase{
			Text: brief.PrimaryKeyword, Lead: true, Why: "the primary keyword opens the page",
		})
	}
	for _, target := range owedWithin(lc, rules) {
		if len(target.Anchors) == 0 {
			continue
		}
		if target.Relation == RelationDown && rules.ChildrenSection {
			brief.Children = append(brief.Children, target.Anchors[0])
			continue
		}
		within := 0
		if target.Relation == RelationUp {
			within = rules.ParentLinkWithinParagraphs
		}
		brief.Phrases = append(brief.Phrases, BriefPhrase{Text: target.Anchors[0], Within: within, Why: whyOwed(target)})
	}
	return brief
}

func owedWithin(lc LinkContext, rules template.LinkRules) []LinkTarget {
	if rules.MaxLinks <= 0 || len(lc.Targets) <= rules.MaxLinks {
		return lc.Targets
	}
	return lc.Targets[:rules.MaxLinks]
}

func whyOwed(target LinkTarget) string {
	switch target.Relation {
	case RelationDown:
		return "the page links down to " + target.URL + ", a page below it, with it"
	case RelationSibling:
		return "the page links to " + target.URL + ", a related page, with it"
	default:
		return "the page links to " + target.URL + " with it"
	}
}

func (b Brief) RequiredHeadings() []string {
	out := make([]string, 0, len(b.Sections))
	for i := range b.Sections {
		if b.Sections[i].Required {
			out = append(out, b.Sections[i].Heading)
		}
	}
	return out
}

func (b Brief) PhraseTexts() []string {
	out := make([]string, 0, len(b.Phrases))
	for i := range b.Phrases {
		out = append(out, b.Phrases[i].Text)
	}
	return out
}
