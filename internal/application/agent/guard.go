package agent

import (
	"encoding/json"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	ResultKey    = "result"
	DroppedKey   = "droppedItems"
	ShortenedKey = "shortenedText"

	minTextBytes = 160
	cutMarker    = "…"
	capRounds    = 64
)

func Fence(result any) map[string]any {
	return map[string]any{UntrustedMarker: true, UntrustedData: result}
}

func Permit(allowed []string, tool string) error {
	if len(allowed) == 0 || slices.Contains(allowed, tool) {
		return nil
	}
	return errors.New(errors.Unauthorized, "the tool "+tool+" is not open to this conversation").
		WithDetail("tool", tool)
}

func Cap(result map[string]any, limit int) (map[string]any, bool) {
	encoded, err := json.Marshal(result)
	if err != nil || limit <= 0 || len(encoded) <= limit {
		return result, false
	}

	total := len(encoded)
	var document any
	if json.Unmarshal(encoded, &document) != nil {
		return preview(string(encoded), total, limit), true
	}

	dropped := map[string]int{}
	shortened := 0
	for range capRounds {
		if measured(report(document, total, dropped, shortened)) <= limit {
			break
		}
		if thinned(document, dropped) {
			continue
		}
		if cut(document) {
			shortened++
			continue
		}
		break
	}

	capped := report(document, total, dropped, shortened)
	if measured(capped) <= limit {
		return capped, true
	}
	return preview(string(encoded), total, limit), true
}

func report(document any, total int, dropped map[string]int, shortened int) map[string]any {
	out := map[string]any{TruncatedKey: true, TotalBytesKey: total, ResultKey: document}
	if len(dropped) > 0 {
		out[DroppedKey] = dropped
	}
	if shortened > 0 {
		out[ShortenedKey] = shortened
	}
	return out
}

func measured(value any) int {
	encoded, err := json.Marshal(value)
	if err != nil {
		return 0
	}
	return len(encoded)
}

func preview(encoded string, total, limit int) map[string]any {
	room := min(max(limit/2, MinPreviewBytes), len(encoded))
	return map[string]any{
		TruncatedKey: true, TotalBytesKey: total, PreviewKey: CutAtRune(encoded, room),
	}
}

func CutAtRune(text string, limit int) string {
	if len(text) <= limit {
		return text
	}

	at := limit
	for at > 0 && !utf8.RuneStart(text[at]) {
		at--
	}
	return text[:at]
}

type arrayAt struct {
	path  string
	items []any
	set   func([]any)
}

type textAt struct {
	text string
	set  func(string)
}

func arrays(value any, path string, into *[]arrayAt) {
	switch held := value.(type) {
	case map[string]any:
		for key, child := range held {
			next := join(path, key)
			if items, ok := child.([]any); ok {
				owner, name := held, key
				*into = append(*into, arrayAt{
					path: next, items: items, set: func(kept []any) { owner[name] = kept },
				})
			}
			arrays(child, next, into)
		}
	case []any:
		for i, child := range held {
			owner, at := held, i
			if items, ok := child.([]any); ok {
				*into = append(*into, arrayAt{
					path: path + "[]", items: items, set: func(kept []any) { owner[at] = kept },
				})
			}
			arrays(child, path+"[]", into)
		}
	default:
	}
}

func texts(value any, into *[]textAt) {
	switch held := value.(type) {
	case map[string]any:
		for key, child := range held {
			if text, ok := child.(string); ok {
				owner, name := held, key
				*into = append(*into, textAt{text: text, set: func(kept string) { owner[name] = kept }})
			}
			texts(child, into)
		}
	case []any:
		for i, child := range held {
			if text, ok := child.(string); ok {
				owner, at := held, i
				*into = append(*into, textAt{text: text, set: func(kept string) { owner[at] = kept }})
			}
			texts(child, into)
		}
	default:
	}
}

func join(path, key string) string {
	if path == "" {
		return key
	}
	return path + "." + key
}

func thinned(document any, dropped map[string]int) bool {
	var found []arrayAt
	arrays(document, "", &found)

	widest := -1
	for i := range found {
		if len(found[i].items) > 1 && (widest == -1 || len(found[i].items) > len(found[widest].items)) {
			widest = i
		}
	}
	if widest == -1 {
		return false
	}

	held := found[widest]
	keep := len(held.items) / 2
	held.set(held.items[:keep])
	dropped[held.path] += len(held.items) - keep
	return true
}

func cut(document any) bool {
	var found []textAt
	texts(document, &found)

	longest := -1
	for i := range found {
		if len(found[i].text) > minTextBytes && (longest == -1 || len(found[i].text) > len(found[longest].text)) {
			longest = i
		}
	}
	if longest == -1 {
		return false
	}

	held := found[longest]
	held.set(strings.TrimRight(CutAtRune(held.text, len(held.text)/2), " ") + cutMarker)
	return true
}
