package pagetype

import (
	"strings"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const anySegments = "**"

type glob struct {
	segments []string
}

func compileGlob(pattern string) (glob, error) {
	if !strings.HasPrefix(pattern, "/") {
		return glob{}, errors.New(errors.Invalid, "a pattern starts at the site's root, with a slash")
	}
	if strings.ContainsAny(pattern, "?[]{}\\") {
		return glob{}, errors.New(errors.Invalid, "a pattern takes * for any text in one part of the address and ** for any number of parts, nothing else")
	}
	segments := segmentsOf(pattern)
	if len(segments) == 0 {
		return glob{}, errors.New(errors.Invalid, "a pattern names at least one part of the address")
	}
	for _, segment := range segments {
		if strings.Contains(segment, anySegments) && segment != anySegments {
			return glob{}, errors.New(errors.Invalid, "** stands alone between two slashes")
		}
	}
	return glob{segments: segments}, nil
}

func (g glob) match(path string) bool {
	return matchSegments(g.segments, segmentsOf(path))
}

func matchSegments(pattern, path []string) bool {
	switch {
	case len(pattern) == 0:
		return len(path) == 0
	case pattern[0] == anySegments:
		for skip := 0; skip <= len(path); skip++ {
			if matchSegments(pattern[1:], path[skip:]) {
				return true
			}
		}
		return false
	case len(path) == 0:
		return false
	default:
		return matchSegment(pattern[0], path[0]) && matchSegments(pattern[1:], path[1:])
	}
}

func matchSegment(pattern, segment string) bool {
	parts := strings.Split(pattern, "*")
	if len(parts) == 1 {
		return pattern == segment
	}
	if !strings.HasPrefix(segment, parts[0]) {
		return false
	}
	rest := segment[len(parts[0]):]
	for _, part := range parts[1 : len(parts)-1] {
		at := strings.Index(rest, part)
		if at < 0 {
			return false
		}
		rest = rest[at+len(part):]
	}
	return strings.HasSuffix(rest, parts[len(parts)-1])
}

func segmentsOf(path string) []string {
	out := make([]string, 0, strings.Count(path, "/"))
	for segment := range strings.SplitSeq(path, "/") {
		if segment != "" {
			out = append(out, segment)
		}
	}
	return out
}

func wordsOf(segment string) []string {
	return strings.FieldsFunc(segment, func(r rune) bool { return r == '-' || r == '_' })
}

func holdsWords(segment string, words []string) bool {
	held := wordsOf(segment)
	for start := 0; start+len(words) <= len(held); start++ {
		run := true
		for i := range words {
			if held[start+i] != words[i] {
				run = false
				break
			}
		}
		if run {
			return true
		}
	}
	return false
}
