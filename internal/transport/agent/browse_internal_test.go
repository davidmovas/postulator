package agent

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	agentapp "github.com/davidmovas/postulator/internal/application/agent"
	"github.com/davidmovas/postulator/internal/application/llm"
	domainllm "github.com/davidmovas/postulator/internal/domain/llm"
)

type toolEvent struct {
	started bool
	callID  string
	tool    string
	args    string
}

type watchingStream struct {
	quietStream
	events []toolEvent
}

func (w *watchingStream) ToolStarted(_ context.Context, callID, tool string, args json.RawMessage) error {
	w.events = append(w.events, toolEvent{started: true, callID: callID, tool: tool, args: string(args)})
	return nil
}

func (w *watchingStream) ToolFinished(_ context.Context, outcome agentapp.ToolOutcome) error {
	w.events = append(w.events, toolEvent{callID: outcome.CallID, tool: outcome.Tool, args: string(outcome.Args)})
	return nil
}

func browsed(query string, finished bool) llm.Delta {
	return llm.Delta{Browsing: &llm.Browsing{Query: query, Done: finished}}
}

func sourced(input, output, searches int, sources ...llm.Source) llm.Delta {
	used := domainllm.Usage{Input: input, Output: output, Total: input + output, WebSearches: searches}
	return llm.Delta{Done: true, Usage: &used, Finish: llm.FinishStop, Tier: domainllm.TierDefault, Sources: sources}
}

func TestTheAssistantSearchesOnlyWhenAPersonTurnedItOn(t *testing.T) {
	t.Parallel()

	allowed := llm.WebSearch{Depth: domainllm.DepthMedium, MaxCalls: 2}
	cases := []struct {
		name   string
		search func() *llm.WebSearch
		want   *llm.WebSearch
	}{
		{name: "the setting is off", search: func() *llm.WebSearch { return nil }},
		{name: "the setting is on", search: func() *llm.WebSearch { return &allowed }, want: &allowed},
		{name: "a runner built without the switch"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			client := &scriptedStream{rounds: [][]llm.Delta{round(text("Done."), one(done(10, 0, 3, "")))}}
			if _, err := playedWith(t, client, &quietStream{}, 2, func(deps *Deps) { deps.WebSearch = tc.search }); err != nil {
				t.Fatalf("the turn failed: %v", err)
			}
			got := client.asked[0].WebSearch
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("the round asked with %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestASearchingAnswerShowsItsSearchesAndEndsWithItsSources(t *testing.T) {
	t.Parallel()

	allowed := llm.WebSearch{Depth: domainllm.DepthLow, MaxCalls: 3}
	client := &scriptedStream{rounds: [][]llm.Delta{round(
		one(browsed("", false)), one(browsed("caffeine peak time", true)),
		one(browsed("caffeine half life", true)),
		text("Caffeine peaks within an hour, see [the WHO](https://www.who.int/caffeine)."),
		one(sourced(900, 40, 2,
			llm.Source{URL: "https://www.who.int/caffeine", Title: "WHO"},
			llm.Source{URL: "https://nih.gov/caffeine", Title: "NIH on caffeine"},
			llm.Source{URL: "https://example.org/no-title"},
		)),
	)}}
	stream := &watchingStream{}
	current, err := playedWith(t, client, stream, 2, func(deps *Deps) {
		deps.WebSearch = func() *llm.WebSearch { return &allowed }
	})
	if err != nil {
		t.Fatalf("the turn failed: %v", err)
	}

	if len(stream.events) != 4 {
		t.Fatalf("the window was told %+v, want two searches started and finished", stream.events)
	}
	first, second := stream.events[0], stream.events[1]
	if !first.started || first.tool != webSearchTool || second.started || second.callID != first.callID ||
		!strings.Contains(second.args, "caffeine peak time") {
		t.Errorf("the first search reads %+v then %+v", first, second)
	}
	if third := stream.events[2]; !third.started || third.callID == first.callID || !strings.Contains(third.args, "caffeine half life") {
		t.Errorf("a search announced only when it ended still starts on its own: %+v", third)
	}

	answer := current.spoken.String()
	if strings.Count(answer, "https://www.who.int/caffeine") != 1 {
		t.Errorf("the answer lists a source it already cites: %q", answer)
	}
	for _, want := range []string{"[NIH on caffeine](https://nih.gov/caffeine)", "[example.org](https://example.org/no-title)"} {
		if !strings.Contains(answer, want) {
			t.Errorf("the answer lacks the source %q: %q", want, answer)
		}
	}
	if joined := strings.Join(stream.deltas, ""); joined != answer {
		t.Errorf("the window saw %q, want the answer with its sources %q", joined, answer)
	}

	priced := domainllm.Cost(domainllm.Usage{Input: 900, Output: 40, Total: 940, WebSearches: 2}, domainllm.ModelInfo{
		InputUSDPerM: 2, CachedInputUSDPerM: 0.2, OutputUSDPerM: 12,
	}, domainllm.TierDefault)
	if spent := stream.rounds[0].USD; spent != priced {
		t.Errorf("the round cost %v, want %v with the two searches' fee", spent, priced)
	}
}
