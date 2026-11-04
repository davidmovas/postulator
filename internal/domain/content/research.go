package content

import (
	"net/url"
	"regexp"
	"strings"
	"unicode"
)

type Source struct {
	URL   string `json:"url"`
	Title string `json:"title,omitempty"`
}

type Research struct {
	Text     string   `json:"text"`
	Sources  []Source `json:"sources"`
	Searches int      `json:"searches"`
}

type Citation struct {
	Fact string `json:"fact"`
	Tag  string `json:"tag,omitempty"`
	URL  string `json:"url,omitempty"`
}

func (c Citation) Tagged() string {
	if c.Tag == "" {
		return c.Fact
	}
	return c.Fact + " [" + c.Tag + "]"
}

var (
	citationLink = regexp.MustCompile(`\(?\[[^\]]*\]\((https?://[^\s()]+)\)\)?`)
	bracketTag   = regexp.MustCompile(`\s*\[([^\[\]]+)\]\s*$`)
	parenAddress = regexp.MustCompile(`\s*\((https?://[^\s()]+)\)\s*$`)
)

func (r Research) Citations() []Citation {
	found := r.found()
	out := []Citation{}
	for _, line := range strings.Split(r.Text, "\n") {
		text := unmarked(line)
		if text == "" {
			continue
		}
		addresses := make([]string, 0, 2)
		for _, match := range citationLink.FindAllStringSubmatch(text, -1) {
			addresses = append(addresses, match[1])
		}
		text, tag := trailingOf(strings.Join(strings.Fields(citationLink.ReplaceAllString(text, " ")), " "))
		if address, named := addressIn(tag); named {
			addresses = append(addresses, address)
			tag = sourceSite(address)
		}
		if text == "" {
			continue
		}
		out = append(out, Citation{Fact: text, Tag: tag, URL: firstFound(addresses, found)})
	}
	return out
}

func trailingOf(text string) (fact, tag string) {
	for _, pattern := range []*regexp.Regexp{bracketTag, parenAddress} {
		if at := pattern.FindStringSubmatchIndex(text); at != nil {
			return strings.TrimSpace(text[:at[0]]), strings.TrimSpace(text[at[2]:at[3]])
		}
	}
	return text, ""
}

func addressIn(tag string) (string, bool) {
	tokens := strings.FieldsFunc(tag, func(r rune) bool { return r == ',' || r == ';' || unicode.IsSpace(r) })
	for _, token := range tokens {
		if _, valid := SourceHref(token); valid {
			return token, true
		}
	}
	return "", false
}

func firstFound(addresses []string, found map[string]string) string {
	for _, address := range addresses {
		if href, held := found[sourceKeyOf(address)]; held {
			return href
		}
	}
	return ""
}

func (r Research) Facts() []string {
	citations := r.Citations()
	out := make([]string, 0, len(citations))
	for _, citation := range citations {
		out = append(out, citation.Tagged())
	}
	return out
}

func (r Research) Empty() bool {
	return len(r.Citations()) == 0
}

func (r Research) found() map[string]string {
	out := make(map[string]string, len(r.Sources))
	for _, source := range r.Sources {
		if href, valid := SourceHref(source.URL); valid {
			out[sourceKeyOf(href)] = href
		}
	}
	return out
}

func SourceHref(raw string) (string, bool) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return "", false
	}
	parsed.Host = strings.ToLower(parsed.Host)
	parsed.Fragment, parsed.RawFragment = "", ""
	kept := make([]string, 0, 4)
	for _, part := range strings.Split(parsed.RawQuery, "&") {
		if part != "" && !tracking(part) {
			kept = append(kept, part)
		}
	}
	parsed.RawQuery = strings.Join(kept, "&")
	parsed.ForceQuery = false
	return parsed.String(), true
}

func tracking(part string) bool {
	key, _, _ := strings.Cut(part, "=")
	if unescaped, err := url.QueryUnescape(key); err == nil {
		key = unescaped
	}
	return strings.HasPrefix(strings.ToLower(strings.TrimLeft(key, "; ")), "utm_")
}

func sourceKeyOf(raw string) string {
	href, valid := SourceHref(raw)
	if !valid {
		return ""
	}
	parsed, err := url.Parse(href)
	if err != nil {
		return ""
	}
	parsed.Path = strings.TrimSuffix(parsed.Path, "/")
	parsed.RawPath = ""
	return parsed.String()
}

func sourceSite(href string) string {
	parsed, err := url.Parse(href)
	if err != nil {
		return ""
	}
	return strings.TrimPrefix(strings.ToLower(parsed.Hostname()), "www.")
}

func unmarked(line string) string {
	text := strings.TrimSpace(line)
	text = strings.TrimSpace(strings.TrimLeft(text, "#"))
	for _, bullet := range []string{"- ", "* ", "• "} {
		text = strings.TrimPrefix(text, bullet)
	}
	digits := strings.IndexFunc(text, func(r rune) bool { return !unicode.IsDigit(r) })
	if digits > 0 && digits+1 < len(text) && (text[digits] == '.' || text[digits] == ')') && text[digits+1] == ' ' {
		text = text[digits+2:]
	}
	return strings.TrimSpace(text)
}
