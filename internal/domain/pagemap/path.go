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

type LinkKind string

const (
	LinkPath         LinkKind = "path"
	LinkSameDocument LinkKind = "same_document"
	LinkExternal     LinkKind = "external"
	LinkUnresolved   LinkKind = "unresolved"
)

type Site struct {
	Scheme string `json:"scheme"`
	Host   string `json:"host"`
	Base   string `json:"base"`
}

func NewSite(baseURL string) Site {
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil {
		return Site{Base: "/"}
	}
	base, err := NormalizePath(parsed.EscapedPath())
	if err != nil {
		base = "/"
	}
	return Site{Scheme: strings.ToLower(parsed.Scheme), Host: strings.ToLower(parsed.Host), Base: base}
}

func (s Site) basePath() string {
	if s.Base == "" {
		return "/"
	}
	return s.Base
}

func (s Site) origin() string {
	if s.Host == "" {
		return ""
	}
	scheme := s.Scheme
	if scheme == "" {
		scheme = "https"
	}
	return scheme + "://" + s.Host
}

func (s Site) URL(path string) string {
	base := s.basePath()
	switch {
	case path == "":
		return s.origin() + base
	case !strings.HasPrefix(path, "/"):
		return s.origin() + base + path
	case base != "/" && !strings.HasPrefix(path, base):
		return s.origin() + base + strings.TrimPrefix(path, "/")
	default:
		return s.origin() + path
	}
}

func (s Site) Resolve(href string) (path string, kind LinkKind) {
	trimmed := strings.TrimSpace(href)
	if trimmed == "" || trimmed[0] == '#' || trimmed[0] == '?' {
		return "", LinkSameDocument
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", LinkUnresolved
	}
	if parsed.Scheme != "" && parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", LinkExternal
	}
	if parsed.Host != "" && !strings.EqualFold(parsed.Host, s.Host) {
		return "", LinkExternal
	}

	escaped := parsed.EscapedPath()
	switch {
	case escaped == "" && parsed.Host == "":
		return "", LinkSameDocument
	case escaped == "":
		return "/", LinkPath
	case escaped[0] != '/' && parsed.Host != "":
		return "", LinkExternal
	case escaped[0] != '/':
		escaped = s.basePath() + escaped
	}

	normalized, err := NormalizePath(escaped)
	if err != nil {
		return "", LinkUnresolved
	}
	return normalized, LinkPath
}

func InternalPath(href, siteHost string) (path string, internal bool) {
	resolved, kind := Site{Host: siteHost}.Resolve(href)
	switch kind {
	case LinkPath:
		return resolved, true
	case LinkSameDocument:
		return "", true
	default:
		return "", false
	}
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
