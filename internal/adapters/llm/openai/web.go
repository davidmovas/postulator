package openai

import (
	"encoding/json"
	"strconv"
	"strings"

	port "github.com/davidmovas/postulator/internal/application/llm"
)

const (
	itemWebSearchCall  = "web_search_call"
	actionSearch       = "search"
	annotationCitation = "url_citation"

	eventSearchStarted = "response.web_search_call.in_progress"
	eventSearching     = "response.web_search_call.searching"

	webPrefix     = "web:"
	webOpenPrefix = "web-open:"

	markOpen  = ''
	markClose = ''
	markJoin  = ''
)

type wireAction struct {
	Type    string       `json:"type"`
	Query   string       `json:"query"`
	Sources []wireSource `json:"sources"`
}

type wireSource struct {
	URL string `json:"url"`
}

type wireAnnotation struct {
	Type  string `json:"type"`
	URL   string `json:"url"`
	Title string `json:"title"`
}

type wireToolUsage struct {
	WebSearch *struct {
		NumRequests int `json:"num_requests"`
	} `json:"web_search"`
}

func webSearchesOf(decoded wireResponse) int {
	if decoded.ToolUsage != nil && decoded.ToolUsage.WebSearch != nil {
		return decoded.ToolUsage.WebSearch.NumRequests
	}
	count := 0
	for i := range decoded.Output {
		if searched(decoded.Output[i]) {
			count++
		}
	}
	return count
}

func searched(item wireOutput) bool {
	if item.Type != itemWebSearchCall {
		return false
	}
	return item.Action == nil || item.Action.Type == "" || item.Action.Type == actionSearch
}

func sourcesOf(output []wireOutput) []port.Source {
	cited := citedOf(output)
	if len(cited) > 0 {
		return cited
	}
	var read []port.Source
	seen := map[string]bool{}
	for i := range output {
		if output[i].Type != itemWebSearchCall || output[i].Action == nil {
			continue
		}
		for _, source := range output[i].Action.Sources {
			if source.URL != "" && !seen[source.URL] {
				seen[source.URL] = true
				read = append(read, port.Source{URL: source.URL})
			}
		}
	}
	return read
}

func citedOf(output []wireOutput) []port.Source {
	var cited []port.Source
	seen := map[string]bool{}
	for i := range output {
		if output[i].Type != itemMessage {
			continue
		}
		for _, part := range output[i].Content {
			for _, annotation := range part.Annotations {
				if annotation.Type != annotationCitation || annotation.URL == "" || seen[annotation.URL] {
					continue
				}
				seen[annotation.URL] = true
				cited = append(cited, port.Source{URL: annotation.URL, Title: annotation.Title})
			}
		}
	}
	return cited
}

type unmarker struct {
	inside bool
}

func (u *unmarker) clean(text string) string {
	if !u.inside && !strings.ContainsAny(text, string([]rune{markOpen, markClose, markJoin})) {
		return text
	}
	var builder strings.Builder
	for _, r := range text {
		switch {
		case r == markOpen:
			u.inside = true
		case u.inside:
			u.inside = r != markClose
		case r == markClose, r == markJoin:
		default:
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

func unmarked(text string) string {
	return (&unmarker{}).clean(text)
}

func browsedOf(item wireOutput) port.Browsing {
	browsing := port.Browsing{Done: true}
	if item.Action != nil {
		browsing.Query = item.Action.Query
	}
	return browsing
}

func browsingKey(index int, item wireOutput) string {
	if item.ID != "" {
		return webPrefix + item.ID
	}
	return webPrefix + "#" + strconv.Itoa(index)
}

func (s *streamState) browsingStarted(data []byte) outcome {
	var started struct {
		ItemID string `json:"item_id"`
		Index  int    `json:"output_index"`
	}
	if err := json.Unmarshal(data, &started); err != nil {
		return outcome{err: unreadableEvent(err), ended: true}
	}
	key := webOpenPrefix + started.ItemID
	if started.ItemID == "" {
		key = webOpenPrefix + "#" + strconv.Itoa(started.Index)
	}
	if s.delivered[key] {
		return outcome{}
	}
	s.delivered[key] = true
	return outcome{deltas: []port.Delta{{Browsing: &port.Browsing{}}}}
}
