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

func MergeNotes(stored, file []Note) []Note {
	merged := NewNotes(stored)
	at := make(map[string]int, len(merged))
	for i := range merged {
		at[strings.ToLower(merged[i].Label)] = i
	}
	for _, incoming := range NewNotes(file) {
		if i, held := at[strings.ToLower(incoming.Label)]; held {
			merged[i].Text = incoming.Text
			continue
		}
		at[strings.ToLower(incoming.Label)] = len(merged)
		merged = append(merged, incoming)
	}
	return merged
}
