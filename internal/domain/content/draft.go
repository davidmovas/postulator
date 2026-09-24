package content

import (
	"strconv"
	"strings"
	"unicode"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	CodeHeadingLacksKeyword   = "primary_missing_in_heading"
	CodeSectionEmpty          = "section_empty"
	CodeTitleFallback         = "title_fallback"
	CodeH1Fallback            = "h1_fallback"
	CodePlanTitleLacksKeyword = "plan_title_lacks_keyword"
	CodePlanH1LacksKeyword    = "plan_h1_lacks_keyword"

	ReasonIncompleteAnswer = "incomplete_answer"
)

type AnswerSection struct {
	Slot    int    `json:"slot" description:"The number of the section from the brief this one answers, 1 for the first; 0 for a section of your own"`
	Heading string `json:"heading" description:"The H2 heading of this section, exactly as the brief gives it"`
	HTML    string `json:"html" description:"The body of this section as HTML paragraphs and lists, without the heading"`
}

type DraftAnswer struct {
	Title    string          `json:"title" description:"The title of the page"`
	H1       string          `json:"h1" description:"The single on-page H1"`
	Sections []AnswerSection `json:"sections" description:"The sections of the body in reading order"`
	Summary  string          `json:"summary" description:"A one-sentence summary of the page"`
}

type DraftSection struct {
	Heading string `json:"heading"`
	HTML    string `json:"html"`
}

type ContentDraft struct {
	Title    string         `json:"title"`
	H1       string         `json:"h1"`
	Sections []DraftSection `json:"sections"`
	Summary  string         `json:"summary"`
	Findings []Finding      `json:"findings,omitempty"`
}

type RepairRequest struct {
	ParagraphIndex int    `json:"paragraphIndex"`
	Phrase         string `json:"phrase"`
}

type RepairResponse struct {
	Sentence string `json:"sentence" description:"One sentence that contains the requested phrase verbatim"`
}

func incomplete(message string, detail map[string]any) error {
	err := errors.New(errors.External, message).WithDetail("reason", ReasonIncompleteAnswer)
	for key, value := range detail {
		err = err.WithDetail(key, value)
	}
	return err
}

func Assemble(answer DraftAnswer, brief Brief) (ContentDraft, *Document, error) {
	sections, findings, err := placeSections(answer, brief)
	if err != nil {
		return ContentDraft{}, nil, err
	}

	draft := ContentDraft{
		Sections: sections,
		Summary:  strings.TrimSpace(answer.Summary),
		Findings: findings,
	}
	draft.Title, draft.Findings = settleTitle(answer, brief, draft.Findings)
	draft.H1, draft.Findings = settleH1(answer, brief, draft.Findings)
	if draft.Findings == nil {
		draft.Findings = []Finding{}
	}

	doc, err := Render(draft)
	if err != nil {
		return ContentDraft{}, nil, err
	}
	return draft, doc, nil
}

func Render(draft ContentDraft) (*Document, error) {
	if strings.TrimSpace(draft.H1) == "" {
		return nil, errors.New(errors.Invalid, "the draft carries no h1").WithDetail("field", "h1")
	}
	if len(draft.Sections) == 0 {
		return nil, errors.New(errors.Invalid, "the draft carries no section").WithDetail("field", "sections")
	}

	var builder strings.Builder
	builder.WriteString("<h1>" + escapeText(draft.H1) + "</h1>")
	for _, section := range draft.Sections {
		if heading := strings.TrimSpace(section.Heading); heading != "" {
			builder.WriteString("<h2>" + escapeText(heading) + "</h2>")
		}
		builder.WriteString(strings.TrimSpace(section.HTML))
	}
	return Parse(builder.String())
}

type placedSection struct {
	answer AnswerSection
	found  bool
}

func placeSections(answer DraftAnswer, brief Brief) ([]DraftSection, []Finding, error) {
	if len(answer.Sections) == 0 {
		return nil, nil, incomplete("the model answered with no section at all", nil)
	}

	taken := make([]bool, len(answer.Sections))
	placed := make([]placedSection, len(brief.Sections))
	for i := range brief.Sections {
		if at, ok := matchBySlot(answer.Sections, taken, brief.Sections[i].Slot); ok {
			placed[i] = placedSection{answer: answer.Sections[at], found: true}
			taken[at] = true
		}
	}
	for i := range brief.Sections {
		if placed[i].found {
			continue
		}
		if at, ok := matchByHeading(answer.Sections, taken, brief.Sections[i].Heading); ok {
			placed[i] = placedSection{answer: answer.Sections[at], found: true}
			taken[at] = true
		}
	}
	for i := range brief.Sections {
		if placed[i].found {
			continue
		}
		if at, ok := nextFree(answer.Sections, taken); ok && brief.Sections[i].Required {
			placed[i] = placedSection{answer: answer.Sections[at], found: true}
			taken[at] = true
		}
	}

	findings := make([]Finding, 0)
	out := make([]DraftSection, 0, len(answer.Sections))
	for i := range brief.Sections {
		section := brief.Sections[i]
		if !placed[i].found {
			if section.Required {
				return nil, nil, incomplete("the model left out the required section "+section.Heading,
					map[string]any{"heading": section.Heading, "slot": section.Slot})
			}
			continue
		}
		html, stripped := Sanitize(placed[i].answer.HTML)
		findings = append(findings, strippedFindings(section.Heading, stripped)...)
		if strings.TrimSpace(html) == "" {
			if section.Required {
				return nil, nil, incomplete("the model left the required section "+section.Heading+" empty",
					map[string]any{"heading": section.Heading, "slot": section.Slot})
			}
			findings = append(findings, Finding{
				Severity: SeverityWarn, Code: CodeSectionEmpty,
				Message: "the section " + section.Heading + " came back empty and was left out",
				Details: map[string]any{"heading": section.Heading},
			})
			continue
		}
		heading, headingFindings := settleHeading(section, placed[i].answer.Heading, brief.PrimaryKeyword)
		findings = append(findings, headingFindings...)
		out = append(out, DraftSection{Heading: heading, HTML: html})
	}

	for at := range answer.Sections {
		if taken[at] {
			continue
		}
		html, stripped := Sanitize(answer.Sections[at].HTML)
		heading := strings.TrimSpace(answer.Sections[at].Heading)
		findings = append(findings, strippedFindings(heading, stripped)...)
		if strings.TrimSpace(html) == "" {
			continue
		}
		out = append(out, DraftSection{Heading: heading, HTML: html})
	}
	if len(out) == 0 {
		return nil, nil, incomplete("every section the model answered with was empty", nil)
	}
	return out, findings, nil
}

func matchBySlot(sections []AnswerSection, taken []bool, slot int) (int, bool) {
	for at := range sections {
		if !taken[at] && sections[at].Slot == slot {
			return at, true
		}
	}
	return 0, false
}

func matchByHeading(sections []AnswerSection, taken []bool, heading string) (int, bool) {
	wanted := strings.TrimSpace(heading)
	if wanted == "" {
		return 0, false
	}
	for at := range sections {
		if taken[at] {
			continue
		}
		if _, _, found := findFold(sections[at].Heading, wanted); found {
			return at, true
		}
	}
	return 0, false
}

func nextFree(sections []AnswerSection, taken []bool) (int, bool) {
	for at := range sections {
		if !taken[at] && sections[at].Slot == 0 {
			return at, true
		}
	}
	for at := range sections {
		if !taken[at] {
			return at, true
		}
	}
	return 0, false
}

func settleHeading(section BriefSection, answered, primary string) (string, []Finding) {
	answered = strings.TrimSpace(answered)
	if !section.PrimaryInHeading || primary == "" {
		return section.Heading, nil
	}
	if carries(section.Heading, primary) {
		return section.Heading, nil
	}
	if carries(answered, section.Heading) && carries(answered, primary) {
		return answered, nil
	}
	return section.Heading, []Finding{{
		Severity: SeverityWarn, Code: CodeHeadingLacksKeyword,
		Message: "the heading " + section.Heading + " does not carry the primary keyword " + primary +
			", although the template asks for it; write it into the heading of the template or leave the rule off",
		Details: map[string]any{"heading": section.Heading, "primaryKeyword": primary},
	}}
}

func settleTitle(answer DraftAnswer, brief Brief, findings []Finding) (string, []Finding) {
	answered := strings.TrimSpace(answer.Title)
	if brief.PlannedTitle {
		if brief.TitleRule && brief.PrimaryKeyword != "" && !carries(brief.Title, brief.PrimaryKeyword) {
			findings = append(findings, Finding{
				Severity: SeverityWarn, Code: CodePlanTitleLacksKeyword,
				Message: "the planned title " + brief.Title + " does not carry the primary keyword " + brief.PrimaryKeyword,
				Details: map[string]any{"title": brief.Title, "primaryKeyword": brief.PrimaryKeyword},
			})
		}
		return brief.Title, findings
	}
	if answered != "" && (!brief.TitleRule || brief.PrimaryKeyword == "" || carries(answered, brief.PrimaryKeyword)) {
		return answered, findings
	}
	fallback := fallbackHeading(brief.PrimaryKeyword, brief.H1, answer.H1)
	return fallback, append(findings, Finding{
		Severity: SeverityWarn, Code: CodeTitleFallback,
		Message: "the model's title " + strconv.Quote(answered) + " did not carry the primary keyword, so the page is titled " + fallback,
		Details: map[string]any{"answered": answered, "title": fallback, "primaryKeyword": brief.PrimaryKeyword},
	})
}

func settleH1(answer DraftAnswer, brief Brief, findings []Finding) (string, []Finding) {
	answered := strings.TrimSpace(answer.H1)
	if brief.PlannedH1 {
		if brief.H1Rule && brief.PrimaryKeyword != "" && !carries(brief.H1, brief.PrimaryKeyword) {
			findings = append(findings, Finding{
				Severity: SeverityWarn, Code: CodePlanH1LacksKeyword,
				Message: "the planned h1 " + brief.H1 + " does not carry the primary keyword " + brief.PrimaryKeyword,
				Details: map[string]any{"h1": brief.H1, "primaryKeyword": brief.PrimaryKeyword},
			})
		}
		return brief.H1, findings
	}
	if answered != "" && (!brief.H1Rule || brief.PrimaryKeyword == "" || carries(answered, brief.PrimaryKeyword)) {
		return answered, findings
	}
	fallback := fallbackHeading(brief.PrimaryKeyword, brief.Title, answer.Title)
	return fallback, append(findings, Finding{
		Severity: SeverityWarn, Code: CodeH1Fallback,
		Message: "the model's h1 " + strconv.Quote(answered) + " did not carry the primary keyword, so the page opens with " + fallback,
		Details: map[string]any{"answered": answered, "h1": fallback, "primaryKeyword": brief.PrimaryKeyword},
	})
}

func fallbackHeading(primary string, candidates ...string) string {
	for _, candidate := range candidates {
		if trimmed := strings.TrimSpace(candidate); trimmed != "" && carries(trimmed, primary) {
			return trimmed
		}
	}
	if primary != "" {
		return titleCase(primary)
	}
	for _, candidate := range candidates {
		if trimmed := strings.TrimSpace(candidate); trimmed != "" {
			return trimmed
		}
	}
	return "Untitled"
}

func carries(text, phrase string) bool {
	_, _, found := findFold(text, strings.TrimSpace(phrase))
	return found
}

func titleCase(text string) string {
	words := strings.Fields(text)
	for i, word := range words {
		runes := []rune(word)
		runes[0] = unicode.ToUpper(runes[0])
		words[i] = string(runes)
	}
	return strings.Join(words, " ")
}

func escapeText(text string) string {
	replacer := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")
	return replacer.Replace(strings.TrimSpace(text))
}
