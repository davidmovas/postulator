package pagemap

import "strings"

type Note struct {
	Label string `json:"label"`
	Text  string `json:"text"`
}

func NewNotes(notes []Note) []Note {
	out := make([]Note, 0, len(notes))
	held := make(map[string]struct{}, len(notes))
	for _, note := range notes {
		label := strings.TrimSpace(note.Label)
		text := strings.TrimSpace(note.Text)
		if label == "" || text == "" {
			continue
		}
		key := strings.ToLower(label)
		if _, repeated := held[key]; repeated {
			continue
		}
		held[key] = struct{}{}
		out = append(out, Note{Label: label, Text: text})
	}
	return out
}
