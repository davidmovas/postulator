package content

import (
	"strings"

	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/keyword"
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

type BriefKeyword struct {
	Rank     int    `json:"rank"`
	Text     string `json:"text"`
	Volume   *int   `json:"volume,omitempty"`
	Required bool   `json:"required"`
}

type Brief struct {
	Title          string         `json:"title"`
	H1             string         `json:"h1"`
	PlannedTitle   bool           `json:"plannedTitle"`
	PlannedH1      bool           `json:"plannedH1"`
	PrimaryKeyword string         `json:"primaryKeyword"`
	Keywords       []BriefKeyword `json:"keywords"`
	TitleRule      bool           `json:"titleRule"`
	H1Rule         bool           `json:"h1Rule"`
	LeadRule       bool           `json:"leadRule"`
	Sections       []BriefSection `json:"sections"`
	Phrases        []BriefPhrase  `json:"phrases"`
	Children       []string       `json:"children"`
	Product        *BriefProduct  `json:"product,omitempty"`
}

func RequiredKeywords(keywords keyword.List, rules template.KeywordRules) keyword.List {
	if rules.RequiredKeywords == nil || *rules.RequiredKeywords >= len(keywords) {
		return keywords
	}
	return keywords[:max(*rules.RequiredKeywords, 0)]
}

func briefKeywords(keywords keyword.List, rules template.KeywordRules) []BriefKeyword {
	required := len(RequiredKeywords(keywords, rules))
	out := make([]BriefKeyword, 0, len(keywords))
	for at, item := range keywords {
		out = append(out, BriefKeyword{Rank: at + 1, Text: item.Text, Volume: item.Volume, Required: at < required})
	}
	return out
}

func NewBrief(spec template.TemplateSpec, rules template.LinkRules, page pagemap.Page, entity graph.Entity, lc LinkContext) Brief {
	keywords := pagemap.Keywords(page, entity)
	brief := Brief{
		Title:          strings.TrimSpace(page.Title),
		H1:             strings.TrimSpace(page.H1),
		PrimaryKeyword: keywords.Main(),
		Keywords:       briefKeywords(keywords, spec.KeywordRules),
		TitleRule:      spec.KeywordRules.PrimaryInTitle,
		H1Rule:         spec.KeywordRules.PrimaryInH1,
		LeadRule:       spec.KeywordRules.PrimaryInFirstParagraph,
		Sections:       make([]BriefSection, 0, len(spec.Sections)),
		Phrases:        make([]BriefPhrase, 0, len(lc.Targets)+1),
		Children:       make([]string, 0),
	}
	if page.WPType == pagemap.WPProduct {
		if name := StoreName(page); name != "" {
			brief.H1 = name
		}
		if spec.Product != nil {
			brief.Product = productBrief(*spec.Product)
		}
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
