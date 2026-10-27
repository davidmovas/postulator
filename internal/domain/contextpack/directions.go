package contextpack

import (
	"strconv"
	"strings"

	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/run"
)

const (
	SectionBrief     SectionName = "brief"
	SectionReference SectionName = "reference"
	SectionStyle     SectionName = "style"
)

type MustLinkRef struct {
	Path   string `json:"path"`
	Anchor string `json:"anchor,omitempty"`
}

type Directions struct {
	Goal        string        `json:"goal,omitempty"`
	Focus       string        `json:"focus,omitempty"`
	Tone        string        `json:"tone,omitempty"`
	Avoid       string        `json:"avoid,omitempty"`
	Notes       string        `json:"notes,omitempty"`
	AddSections []string      `json:"addSections,omitempty"`
	Keywords    []string      `json:"keywords,omitempty"`
	Anchors     []string      `json:"anchors,omitempty"`
	MustLink    []MustLinkRef `json:"mustLink,omitempty"`
	MinWords    int           `json:"minWords,omitempty"`
	MaxWords    int           `json:"maxWords,omitempty"`
	Reference   string        `json:"reference,omitempty"`
	StyleSample string        `json:"styleSample,omitempty"`
}

func DirectionsOf(brief run.Brief, directive run.Directive, index pagemap.Index) *Directions {
	if brief.Empty() && directive.Empty() {
		return nil
	}
	out := &Directions{
		Goal: brief.Goal, Focus: brief.Focus, Tone: brief.Tone, Avoid: brief.Avoid, Reference: brief.Reference,
		StyleSample: brief.StyleSample, Notes: directive.Notes, AddSections: directive.AddSections, Keywords: directive.Keywords,
		Anchors: directive.Anchors, MinWords: directive.MinWords, MaxWords: directive.MaxWords,
	}
	for _, link := range directive.MustLink {
		if target, found := index.ByID(link.PageID); found {
			out.MustLink = append(out.MustLink, MustLinkRef{Path: target.Path, Anchor: link.Anchor})
		}
	}
	return out
}

func briefSection(f Facts) Section {
	section := Section{Name: SectionBrief, Heading: "The run's brief", Priority: 1}
	d := f.Directions
	if d == nil {
		return section
	}
	add := func(label, text string) {
		if text = strings.TrimSpace(text); text != "" {
			section.Lines = append(section.Lines, label+": "+text)
		}
	}
	add("Goal", d.Goal)
	add("Focus", d.Focus)
	add("Tone", d.Tone)
	add("Avoid", d.Avoid)
	add("For this page", d.Notes)
	add("Add these sections", strings.Join(d.AddSections, "; "))
	add("Use these keywords", strings.Join(d.Keywords, ", "))
	add("Prefer these anchors", strings.Join(d.Anchors, ", "))
	for _, link := range d.MustLink {
		line := "Link to " + link.Path
		if anchor := strings.TrimSpace(link.Anchor); anchor != "" {
			line += " with the anchor “" + anchor + "”"
		}
		section.Lines = append(section.Lines, line)
	}
	add("Length for this run", wordsRange(d.MinWords, d.MaxWords))
	return section
}

func (f Facts) BriefText() string {
	section := briefSection(f)
	if len(section.Lines) == 0 {
		return ""
	}
	return Fitted{Sections: []Section{section}}.Text()
}

func wordsRange(least, most int) string {
	switch {
	case least > 0 && most > 0:
		return strconv.Itoa(least) + " to " + strconv.Itoa(most) + " words"
	case least > 0:
		return "at least " + strconv.Itoa(least) + " words"
	case most > 0:
		return "at most " + strconv.Itoa(most) + " words"
	default:
		return ""
	}
}

func referenceSection(f Facts) Section {
	section := Section{Name: SectionReference, Heading: "Reference text the person gave for this run", Priority: 2}
	if f.Directions == nil {
		return section
	}
	for _, paragraph := range strings.Split(f.Directions.Reference, "\n\n") {
		if paragraph = strings.TrimSpace(paragraph); paragraph != "" {
			section.Items = append(section.Items, paragraph)
		}
	}
	return section
}

func styleSection(f Facts) Section {
	section := Section{Name: SectionStyle, Heading: "A sample of the style to write in", Priority: 2}
	if f.Directions != nil {
		if sample := strings.TrimSpace(f.Directions.StyleSample); sample != "" {
			section.Lines = append(section.Lines, sample)
		}
	}
	return section
}
