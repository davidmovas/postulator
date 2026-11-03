package content

import (
	"strings"
	"unicode"
)

type Source struct {
	URL   string `json:"url"`
	Title string `json:"title,omitempty"`
}

type Research struct {
	Text     string   `json:"text"`
	Sources  []Source `json:"sources"`
	Searches int      `json:"searches"`
}

func (r Research) Facts() []string {
	out := []string{}
	for _, line := range strings.Split(r.Text, "\n") {
		if fact := unmarked(line); fact != "" {
			out = append(out, fact)
		}
	}
	return out
}

func (r Research) Empty() bool {
	return len(r.Facts()) == 0
}

func unmarked(line string) string {
	text := strings.TrimSpace(line)
	text = strings.TrimSpace(strings.TrimLeft(text, "#"))
	for _, bullet := range []string{"- ", "* ", "• "} {
		text = strings.TrimPrefix(text, bullet)
	}
	digits := strings.IndexFunc(text, func(r rune) bool { return !unicode.IsDigit(r) })
	if digits > 0 && digits+1 < len(text) && (text[digits] == '.' || text[digits] == ')') && text[digits+1] == ' ' {
		text = text[digits+2:]
	}
	return strings.TrimSpace(text)
}
