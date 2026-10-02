package keyword

import (
	"cmp"
	"slices"
	"strings"
)

type Keyword struct {
	Text   string `json:"text"`
	Volume *int   `json:"volume,omitempty"`
}

type List []Keyword

func New(items []Keyword) List {
	list := make(List, 0, len(items))
	held := make(map[string]int, len(items))
	for _, item := range items {
		text := strings.TrimSpace(item.Text)
		if text == "" {
			continue
		}
		volume := measured(item.Volume)
		at, repeated := held[fold(text)]
		if repeated {
			if list[at].Volume == nil {
				list[at].Volume = volume
			}
			continue
		}
		held[fold(text)] = len(list)
		list = append(list, Keyword{Text: text, Volume: volume})
	}
	slices.SortStableFunc(list, byVolume)
	return list
}

func Of(texts ...string) List {
	items := make([]Keyword, 0, len(texts))
	for _, text := range texts {
		items = append(items, Keyword{Text: text})
	}
	return New(items)
}

func fold(text string) string {
	return strings.ToLower(text)
}

func measured(volume *int) *int {
	if volume == nil || *volume < 0 {
		return nil
	}
	return new(*volume)
}

func byVolume(a, b Keyword) int {
	switch {
	case a.Volume == nil && b.Volume == nil:
		return 0
	case a.Volume == nil:
		return 1
	case b.Volume == nil:
		return -1
	default:
		return cmp.Compare(*b.Volume, *a.Volume)
	}
}

func (l List) Main() string {
	if len(l) == 0 {
		return ""
	}
	return l[0].Text
}

func (l List) Texts() []string {
	texts := make([]string, 0, len(l))
	for _, item := range l {
		texts = append(texts, item.Text)
	}
	return texts
}

func (l List) Merge(file List) List {
	merged := New(l)
	held := make(map[string]int, len(merged))
	for at, stored := range merged {
		held[fold(stored.Text)] = at
	}
	for _, incoming := range New(file) {
		at, stored := held[fold(incoming.Text)]
		switch {
		case !stored:
			held[fold(incoming.Text)] = len(merged)
			merged = append(merged, incoming)
		case incoming.Volume != nil:
			merged[at].Volume = incoming.Volume
		}
	}
	return New(merged)
}
