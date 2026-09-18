package content

import (
	"strings"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type DraftSection struct {
	Heading string `json:"heading" description:"The H2 heading of this section"`
	HTML    string `json:"html" description:"The body of this section as HTML paragraphs and lists, without the heading"`
}

type ContentDraft struct {
	Title    string         `json:"title" description:"The meta title of the page"`
	H1       string         `json:"h1" description:"The single on-page H1"`
	Sections []DraftSection `json:"sections" description:"The sections of the body in reading order"`
	Summary  string         `json:"summary" description:"A one-sentence summary of the page"`
}

type RepairRequest struct {
	ParagraphIndex int    `json:"paragraphIndex"`
	Phrase         string `json:"phrase"`
}

type RepairResponse struct {
	Sentence string `json:"sentence" description:"One sentence that contains the requested phrase verbatim"`
}

func (d ContentDraft) Validate() error {
	switch {
	case strings.TrimSpace(d.Title) == "":
		return errors.New(errors.Invalid, "the draft carries no title").WithDetail("field", "title")
	case strings.TrimSpace(d.H1) == "":
		return errors.New(errors.Invalid, "the draft carries no h1").WithDetail("field", "h1")
	case len(d.Sections) == 0:
		return errors.New(errors.Invalid, "the draft carries no section").WithDetail("field", "sections")
	}

	for i := range d.Sections {
		if strings.TrimSpace(d.Sections[i].HTML) == "" {
			return errors.New(errors.Invalid, "a draft section carries no body").
				WithDetail("field", "sections").WithDetail("index", i)
		}
	}
	return nil
}

func Assemble(draft ContentDraft) (*Document, error) {
	if err := draft.Validate(); err != nil {
		return nil, err
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

func escapeText(text string) string {
	replacer := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")
	return replacer.Replace(strings.TrimSpace(text))
}
