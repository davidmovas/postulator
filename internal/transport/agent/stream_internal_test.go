package agent

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/gollem-dev/gollem"

	agentapp "github.com/davidmovas/postulator/internal/application/agent"
	domainllm "github.com/davidmovas/postulator/internal/domain/llm"
)

type quietStream struct {
	deltas []string
	rounds []agentapp.RoundUsage
}

func (q *quietStream) Delta(_ context.Context, _ int64, text string) error {
	q.deltas = append(q.deltas, text)
	return nil
}

func (q *quietStream) Spent(_ context.Context, round agentapp.RoundUsage) error {
	q.rounds = append(q.rounds, round)
	return nil
}

func (q *quietStream) ToolStarted(context.Context, string, string, json.RawMessage) error {
	return nil
}

func (q *quietStream) ToolFinished(context.Context, agentapp.ToolOutcome) error {
	return nil
}

func text(pieces ...string) *gollem.ContentResponse {
	return &gollem.ContentResponse{Texts: pieces}
}

func spent(input, cached, output int) *gollem.ContentResponse {
	return &gollem.ContentResponse{InputToken: input, CacheReadInputToken: cached, OutputToken: output}
}

func called(input, cached, output int) *gollem.ContentResponse {
	chunk := spent(input, cached, output)
	chunk.FunctionCalls = []*gollem.FunctionCall{{ID: "c1", Name: "graph_list_entities"}}
	return chunk
}

func watched(t *testing.T, rounds ...[]*gollem.ContentResponse) (
	counted domainllm.Usage, said string, streamed []string) {
	_, counted, said, streamed = billed(t, rounds...)
	return counted, said, streamed
}

func billed(t *testing.T, rounds ...[]*gollem.ContentResponse) (
	recorded []round, counted domainllm.Usage, said string, streamed []string) {
	t.Helper()

	stream := &quietStream{}
	usage := &tally{}
	spoken := &answer{}
	middleware := observe(stream, usage, spoken, func(error) {}, func(done round) {
		recorded = append(recorded, done)
	})

	for _, chunks := range rounds {
		handler := middleware(func(context.Context, *gollem.ContentRequest) (<-chan *gollem.ContentResponse, error) {
			out := make(chan *gollem.ContentResponse, len(chunks))
			for _, chunk := range chunks {
				out <- chunk
			}
			close(out)
			return out, nil
		})

		served, err := handler(t.Context(), &gollem.ContentRequest{})
		if err != nil {
			t.Fatalf("the middleware refused a round: %v", err)
		}
		for range served {
		}
	}
	return recorded, usage.total(), spoken.String(), stream.deltas
}

func TestEveryRoundIsHandedToTheLedgerInOrder(t *testing.T) {
	t.Parallel()

	recorded, counted, _, _ := billed(t,
		[]*gollem.ContentResponse{called(1200, 0, 40), spent(1200, 0, 40)},
		[]*gollem.ContentResponse{called(1800, 1152, 30), spent(1800, 1152, 30)},
		[]*gollem.ContentResponse{text("Twelve entities."), spent(2400, 2048, 12)},
	)

	if len(recorded) != 3 {
		t.Fatalf("the ledger was handed %d rounds, want one per model call", len(recorded))
	}
	for index, done := range recorded {
		if done.index != index+1 || done.failure != nil {
			t.Fatalf("round %d reads %+v", index+1, done)
		}
	}
	if recorded[1].usage != (spend{input: 1800, cached: 1152, output: 30}) {
		t.Fatalf("the second round spent %+v", recorded[1].usage)
	}

	summed := spend{}
	for _, done := range recorded {
		summed.input += done.usage.input
		summed.cached += done.usage.cached
		summed.output += done.usage.output
	}
	if summed.input != counted.Input || summed.cached != counted.CachedInput || summed.output != counted.Output {
		t.Fatalf("the rounds sum to %+v and the turn counted %+v", summed, counted)
	}
}

func TestARoundIsCountedOnceHoweverOftenItReportsItsUsage(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		rounds [][]*gollem.ContentResponse
		want   domainllm.Usage
	}{
		{
			name:   "a plain answer reports its usage in the trailing chunk",
			rounds: [][]*gollem.ContentResponse{{text("Hi"), text(" there"), spent(1200, 0, 40)}},
			want:   domainllm.Usage{Input: 1200, Output: 40, Total: 1240},
		},
		{
			name: "a tool round reports the same totals twice, as the openai client does",
			rounds: [][]*gollem.ContentResponse{
				{called(1200, 1024, 40), spent(1200, 1024, 40)},
			},
			want: domainllm.Usage{Input: 1200, CachedInput: 1024, Output: 40, Total: 1240},
		},
		{
			name: "every round of a turn is counted, each of them once",
			rounds: [][]*gollem.ContentResponse{
				{called(1200, 0, 40), spent(1200, 0, 40)},
				{called(1800, 1152, 30), spent(1800, 1152, 30)},
				{text("Twelve entities."), spent(2400, 2048, 12)},
			},
			want: domainllm.Usage{Input: 5400, CachedInput: 3200, Output: 82, Total: 5482},
		},
		{
			name:   "a round that reports nothing costs nothing",
			rounds: [][]*gollem.ContentResponse{{text("Hi")}},
			want:   domainllm.Usage{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, _, _ := watched(t, tc.rounds...)
			if got != tc.want {
				t.Errorf("usage = %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestTheAnswerIsTheTextTheTurnStreamed(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		rounds [][]*gollem.ContentResponse
		want   string
	}{
		{
			name:   "the pieces of one round are joined with nothing",
			rounds: [][]*gollem.ContentResponse{{text("От", "лич", "ная"), text(" идея", ".")}},
			want:   "Отличная идея.",
		},
		{
			name: "a round that only called a tool says nothing",
			rounds: [][]*gollem.ContentResponse{
				{called(10, 0, 5)},
				{text("Twelve entities."), spent(20, 0, 3)},
			},
			want: "Twelve entities.",
		},
		{
			name: "two speaking rounds are separated by a blank line",
			rounds: [][]*gollem.ContentResponse{
				{text("Let me look."), called(10, 0, 5)},
				{text("Twelve entities."), spent(20, 0, 3)},
			},
			want: "Let me look.\n\nTwelve entities.",
		},
		{
			name:   "a turn that never spoke answers nothing",
			rounds: [][]*gollem.ContentResponse{{called(10, 0, 5)}},
			want:   "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, got, deltas := watched(t, tc.rounds...)
			if got != tc.want {
				t.Errorf("answer = %q, want %q", got, tc.want)
			}
			if streamed := joinAll(deltas); streamed != flatten(tc.rounds) {
				t.Errorf("the stream carried %q, want %q", streamed, flatten(tc.rounds))
			}
		})
	}
}

func joinAll(pieces []string) string {
	out := ""
	for _, piece := range pieces {
		out += piece
	}
	return out
}

func flatten(rounds [][]*gollem.ContentResponse) string {
	out := ""
	for _, chunks := range rounds {
		for _, chunk := range chunks {
			out += joinAll(chunk.Texts)
		}
	}
	return out
}
