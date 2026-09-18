package wp

import (
	"net/url"
	"strings"
)

func NormalizePath(rawPath string) string {
	var builder strings.Builder
	builder.Grow(len(rawPath) + 2)
	builder.WriteByte('/')

	slashed := true
	for index := range len(rawPath) {
		symbol := rawPath[index]
		if symbol == '/' {
			if slashed {
				continue
			}
			slashed = true
			builder.WriteByte('/')
			continue
		}
		slashed = false
		if symbol >= 'A' && symbol <= 'Z' {
			symbol += 'a' - 'A'
		}
		builder.WriteByte(symbol)
	}

	normalised := builder.String()
	if strings.HasSuffix(normalised, "/") {
		return normalised
	}
	return normalised + "/"
}

func InternalPath(siteHost, href string) (string, bool) {
	trimmed := strings.TrimSpace(href)
	if trimmed == "" {
		return "", false
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", false
	}
	if parsed.Scheme != "" && parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", false
	}
	if parsed.Host != "" && !strings.EqualFold(parsed.Host, siteHost) {
		return "", false
	}

	target := parsed.EscapedPath()
	if target == "" && parsed.Host == "" {
		return "", false
	}
	return NormalizePath(target), true
}
