package field

import (
	"html"
	"strings"
)

func (k Key) Plain() bool {
	switch k {
	case Title, Excerpt, SEOTitle, SEODescription:
		return true
	default:
		return false
	}
}

func SameText(key Key, a, b string) bool {
	if a == b {
		return true
	}
	if !key.Plain() {
		return false
	}
	return decoded(a) == decoded(b)
}

func decoded(text string) string {
	return strings.TrimSpace(html.UnescapeString(text))
}
