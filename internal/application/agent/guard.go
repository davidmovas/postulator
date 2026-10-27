package agent

import (
	"encoding/json"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	ResultKey        = "result"
	droppedKey       = "droppedItems"
	shortenedKey     = "shortenedText"
	cursorDroppedKey = "cursorDropped"
	nextCursorKey    = "nextCursor"
	hasMoreKey       = "hasMore"

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

	cutting := capping{dropped: map[string]int{}}
	for range capRounds {
		if measured(cutting.report(document, total)) <= limit {
			break
		}
		if cutting.thinned(document) {
			continue
		}
		if cut(document) {
			cutting.shortened++
			continue
		}
		break
	}

	capped := cutting.report(document, total)
	if measured(capped) <= limit {
		return capped, true
	}
	return preview(string(encoded), total, limit), true
}

type capping struct {
	dropped   map[string]int
	cursors   []string
	shortened int
}

func (c *capping) report(document any, total int) map[string]any {
	out := map[string]any{TruncatedKey: true, TotalBytesKey: total, ResultKey: document}
	if len(c.dropped) > 0 {
		out[droppedKey] = c.dropped
	}
	if len(c.cursors) > 0 {
		out[cursorDroppedKey] = c.cursors
	}
	if c.shortened > 0 {
		out[shortenedKey] = c.shortened
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
		TruncatedKey: true, TotalBytesKey: total, PreviewKey: cutAtRune(encoded, room),
	}
}

func cutAtRune(text string, limit int) string {
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
	owner map[string]any
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
					path: next, items: items, set: func(kept []any) { owner[name] = kept }, owner: owner,
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

func (c *capping) thinned(document any) bool {
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
	c.dropped[held.path] += len(held.items) - keep
	if _, paged := held.owner[nextCursorKey]; paged {
		delete(held.owner, nextCursorKey)
		held.owner[hasMoreKey] = true
		if !slices.Contains(c.cursors, held.path) {
			c.cursors = append(c.cursors, held.path)
		}
	}
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
	held.set(strings.TrimRight(cutAtRune(held.text, len(held.text)/2), " ") + cutMarker)
	return true
}
