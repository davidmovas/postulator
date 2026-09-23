package importmap

import "strings"

type Walk struct {
	binding Binding
	seen    []string
}

func (b Binding) Walk() *Walk {
	return &Walk{binding: b, seen: make([]string, len(b.indent))}
}

func (w *Walk) Path(row []string) string {
	if len(w.binding.indent) == 0 {
		return w.binding.Path(row)
	}

	deepest := -1
	for level, at := range w.binding.indent {
		value := ""
		if at < len(row) {
			value = strings.TrimSpace(row[at])
		}
		if value == "" {
			continue
		}
		w.seen[level] = value
		for below := level + 1; below < len(w.seen); below++ {
			w.seen[below] = ""
		}
		deepest = level
	}
	if deepest < 0 {
		return ""
	}

	segments := make([]string, 0, deepest+1)
	for level := 0; level <= deepest; level++ {
		for _, part := range strings.Split(w.seen[level], "/") {
			trimmed := strings.TrimSpace(part)
			if trimmed != "" {
				segments = append(segments, trimmed)
			}
		}
	}
	if len(segments) == 0 {
		return ""
	}
	return strings.TrimSpace(w.binding.options.Strip("/" + strings.Join(segments, "/") + "/"))
}
