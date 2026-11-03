package agent

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"

	agentapp "github.com/davidmovas/postulator/internal/application/agent"
	"github.com/davidmovas/postulator/internal/application/llm"
	domainagent "github.com/davidmovas/postulator/internal/domain/agent"
)

const (
	webSearchTool  = "web_search"
	webCallPrefix  = "web-"
	sourcesHeading = "\n\nSources:\n"
)

type browsing struct {
	open []string
	next int
}

func (b *browsing) start() string {
	b.next++
	id := webCallPrefix + strconv.Itoa(b.next)
	b.open = append(b.open, id)
	return id
}

func (b *browsing) finish() (string, bool) {
	if len(b.open) == 0 {
		b.next++
		return webCallPrefix + strconv.Itoa(b.next), false
	}
	id := b.open[0]
	b.open = b.open[1:]
	return id, true
}

func (t *turn) browse(ctx context.Context, step llm.Browsing) {
	if t.spec.Stream == nil {
		return
	}
	if !step.Done {
		t.guard.note(t.spec.Stream.ToolStarted(ctx, t.browsed.start(), webSearchTool, json.RawMessage(emptyArguments)))
		return
	}
	args, err := json.Marshal(map[string]string{"query": step.Query})
	if err != nil {
		t.guard.note(err)
		return
	}
	id, opened := t.browsed.finish()
	if !opened {
		t.guard.note(t.spec.Stream.ToolStarted(ctx, id, webSearchTool, args))
	}
	t.guard.note(t.spec.Stream.ToolFinished(ctx, agentapp.ToolOutcome{
		CallID: id, Tool: webSearchTool, Args: args, Result: args, Status: string(domainagent.CallOK),
	}))
}

func sourcesNote(answer string, sources []llm.Source) string {
	var listed strings.Builder
	for _, source := range sources {
		if source.URL == "" || strings.Contains(answer, source.URL) {
			continue
		}
		title := strings.TrimSpace(source.Title)
		if title == "" {
			title = hostOf(source.URL)
		}
		listed.WriteString("- [" + strings.ReplaceAll(title, "]", ")") + "](" + source.URL + ")\n")
	}
	if listed.Len() == 0 {
		return ""
	}
	return sourcesHeading + strings.TrimSuffix(listed.String(), "\n")
}

func hostOf(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" {
		return raw
	}
	return parsed.Host
}
