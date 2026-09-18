package pagemap

import (
	"net/url"
	"strings"
	"unicode"
)

func NormalizePath(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", invalid("path must not be empty", "path")
	}

	if strings.Contains(trimmed, "://") {
		parsed, err := url.Parse(trimmed)
		if err != nil {
			return "", invalid("path is not a valid url", "path").WithInternal(err)
		}
		trimmed = parsed.EscapedPath()
	} else {
		trimmed, _, _ = strings.Cut(trimmed, "?")
		trimmed, _, _ = strings.Cut(trimmed, "#")
	}

	for _, r := range trimmed {
		if unicode.IsSpace(r) || unicode.IsControl(r) {
			return "", invalid("path must not contain whitespace or control characters", "path")
		}
	}

	var builder strings.Builder
	builder.WriteByte('/')
	for _, segment := range strings.Split(trimmed, "/") {
		switch segment {
		case "":
			continue
		case ".", "..":
			return "", invalid("path must not contain dot segments", "path")
		}
		builder.WriteString(strings.ToLower(segment))
		builder.WriteByte('/')
	}
	return builder.String(), nil
}

func InternalPath(href, siteHost string) (path string, internal bool) {
	parsed, err := url.Parse(strings.TrimSpace(href))
	if err != nil {
		return "", false
	}
	if parsed.Scheme != "" && parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", false
	}
	if parsed.Host != "" && !strings.EqualFold(parsed.Host, siteHost) {
		return "", false
	}

	escaped := parsed.EscapedPath()
	if escaped == "" {
		if parsed.Host == "" {
			return "", true
		}
		return "/", true
	}
	normalized, err := NormalizePath(escaped)
	if err != nil {
		return "", false
	}
	return normalized, true
}

func ParentPath(path string) string {
	if path == "" || path == "/" {
		return ""
	}
	trimmed := strings.TrimSuffix(path, "/")
	return trimmed[:strings.LastIndex(trimmed, "/")+1]
}

func Slug(path string) string {
	trimmed := strings.TrimSuffix(path, "/")
	return trimmed[strings.LastIndex(trimmed, "/")+1:]
}
