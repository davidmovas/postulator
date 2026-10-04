package openai

import (
	"bytes"
	"context"
	stderrors "errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"testing/iotest"
	"time"

	port "github.com/davidmovas/postulator/internal/application/llm"
	"github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func readAll(t *testing.T, reader *sseReader) ([]sseEvent, error) {
	t.Helper()

	var events []sseEvent
	for {
		event, err := reader.next()
		if err != nil {
			return events, err
		}
		events = append(events, event)
	}
}

func TestTheEventReaderFollowsTheServerSentEventsFormat(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		input string
		want  []sseEvent
	}{
		{
			name:  "named events",
			input: "event: a\ndata: {\"x\":1}\n\nevent: b\ndata: 2\n\n",
			want:  []sseEvent{{name: "a", data: []byte(`{"x":1}`)}, {name: "b", data: []byte("2")}},
		},
		{
			name:  "carriage returns",
			input: "event: a\r\ndata: 1\r\n\r\n",
			want:  []sseEvent{{name: "a", data: []byte("1")}},
		},
		{
			name:  "data over several lines",
			input: "data: first\ndata: second\n\n",
			want:  []sseEvent{{data: []byte("first\nsecond")}},
		},
		{
			name:  "comments, ids and blank lines are passed over",
			input: ": keep-alive\n\n\nid: 7\nretry: 100\nevent: a\ndata:1\n\n",
			want:  []sseEvent{{name: "a", data: []byte("1")}},
		},
		{
			name:  "a last event without its blank line",
			input: "event: a\ndata: 1",
			want:  []sseEvent{{name: "a", data: []byte("1")}},
		},
		{
			name:  "an empty data line",
			input: "event: ping\ndata:\n\n",
			want:  []sseEvent{{name: "ping", data: []byte{}}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := readAll(t, newSSEReader(strings.NewReader(tc.input), 1<<10))
			if !stderrors.Is(err, io.EOF) {
				t.Fatalf("the reader stopped with %v, want the end of the stream", err)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("events = %+v, want %+v", got, tc.want)
			}
			for i := range got {
				if got[i].name != tc.want[i].name || !bytes.Equal(got[i].data, tc.want[i].data) {
					t.Errorf("event %d = %s %q, want %s %q", i, got[i].name, got[i].data, tc.want[i].name, tc.want[i].data)
				}
			}
		})
	}
}

func TestTheEventReaderRefusesAnEventPastItsCap(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		input string
	}{
		{name: "one long line", input: "data: " + strings.Repeat("x", 4<<10) + "\n\n"},
		{name: "many short lines", input: strings.Repeat("data: "+strings.Repeat("x", 100)+"\n", 50) + "\n"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := newSSEReader(strings.NewReader(tc.input), 1<<10).next()
			if !stderrors.Is(err, errEventTooLarge) {
				t.Fatalf("next = %v, want the cap refused", err)
			}
		})
	}
}

func TestTheEventReaderPassesOnAFailedRead(t *testing.T) {
	t.Parallel()

	broken := io.MultiReader(strings.NewReader("event: a\ndata: 1"), iotest.ErrReader(io.ErrUnexpectedEOF))
	if _, err := newSSEReader(broken, 1<<10).next(); !stderrors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("next = %v, want the read failure", err)
	}
}

func TestTheStreamStateReadsEveryEventItCaresAbout(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		event  sseEvent
		deltas int
		fails  bool
		ended  bool
		code   string
	}{
		{name: "a delta named only by its event", event: sseEvent{name: eventTextDelta, data: []byte(`{"delta":"x"}`)}, deltas: 1},
		{name: "the end sentinel", event: sseEvent{data: []byte(" [DONE] ")}},
		{name: "an event the client does not need", event: sseEvent{name: "response.content_part.added", data: []byte(`{"type":"response.content_part.added"}`)}},
		{name: "a delta that is not text", event: sseEvent{data: []byte(`{"type":"response.output_text.delta","delta":5}`)}, fails: true, ended: true},
		{name: "an item that is not an item", event: sseEvent{data: []byte(`{"type":"response.output_item.done","item":"x"}`)}, fails: true, ended: true},
		{name: "an item that is not a call", event: sseEvent{data: []byte(`{"type":"response.output_item.done","item":{"type":"message"}}`)}},
		{name: "a final event that is not a response", event: sseEvent{data: []byte(`{"type":"response.completed","response":"x"}`)}, fails: true, ended: true},
		{name: "a failure that is not a response", event: sseEvent{data: []byte(`{"type":"response.failed","response":"x"}`)}, fails: true, ended: true},
		{name: "an error that is not an error", event: sseEvent{data: []byte(`{"type":"error","error":"x"}`)}, fails: true, ended: true},
		{
			name:  "a lone failed response",
			event: sseEvent{data: []byte(`{"type":"response.failed","response":{"status":"failed","error":{"code":"server_error","message":"m"}}}`)},
			fails: true, ended: true, code: "server_error",
		},
		{
			name:  "an error in the documented flat shape",
			event: sseEvent{data: []byte(`{"type":"error","code":"rate_limit_exceeded","message":"Please try again in 3s.","param":null}`)},
			fails: true, ended: true, code: "rate_limit_exceeded",
		},
		{
			name:  "an error in the probed nested shape",
			event: sseEvent{data: []byte(`{"type":"error","error":{"type":"insufficient_quota","code":"credit_balance_exhausted","message":"m","param":null}}`)},
			fails: true, ended: true, code: "credit_balance_exhausted",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := newStreamState(tierDefault).read(tc.event)
			if len(got.deltas) != tc.deltas || (got.err != nil) != tc.fails || got.ended != tc.ended {
				t.Fatalf("outcome = %+v, want %d deltas, failure %t, ended %t", got, tc.deltas, tc.fails, tc.ended)
			}
			if tc.code == "" {
				return
			}
			refused := refusalIn(got.err)
			if refused == nil || refused.fault.Code != tc.code || refused.status != 0 {
				t.Errorf("failure = %#v, want an in-stream refusal coded %s", got.err, tc.code)
			}
		})
	}
}

func TestAFlatErrorIsNotMistakenForItsEventType(t *testing.T) {
	t.Parallel()

	got := newStreamState(tierDefault).read(sseEvent{data: []byte(`{"type":"error","code":"rate_limit_exceeded","message":"Please try again in 3s."}`)})
	refused := refusalIn(got.err)
	if refused == nil || refused.fault.Type != "" || refused.effectiveStatus() != http.StatusTooManyRequests {
		t.Fatalf("refusal = %#v, want a rate limit without a borrowed type", refused)
	}
	if delay := refused.kernel(llm.ModelRef{Provider: llm.ProviderOpenAI, Model: "m"}, time.Now()); !errors.IsCode(delay, errors.RateLimited) {
		t.Errorf("kernel = %v, want %s", delay, errors.RateLimited)
	}
}

func TestACallIsDeliveredOnce(t *testing.T) {
	t.Parallel()

	state := newStreamState(tierDefault)
	done := sseEvent{data: []byte(`{"type":"response.output_item.done","item":{"type":"function_call","call_id":"call_1","name":"f","arguments":"{}"}}`)}
	if first := state.read(done); len(first.deltas) != 1 {
		t.Fatalf("first = %+v, want the call", first)
	}
	if again := state.read(done); len(again.deltas) != 0 {
		t.Errorf("again = %+v, want nothing for a call already delivered", again)
	}
	final := state.read(sseEvent{data: []byte(`{"type":"response.completed","response":{"status":"completed",` +
		`"output":[{"type":"function_call","call_id":"call_1","name":"f","arguments":"{}"}]}}`)})
	if len(final.deltas) != 1 || !final.deltas[0].Done {
		t.Errorf("final = %+v, want only the final delta", final)
	}
}

func TestASearchItemIsDeliveredOnceWithOrWithoutItsID(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		item string
	}{
		{name: "an item the server named", item: `{"type":"tool_search_call","id":"tsc_1","execution":"server","arguments":{"paths":["pages"]}}`},
		{name: "an item without an id", item: `{"type":"tool_search_output","execution":"server","tools":[]}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			state := newStreamState(tierDefault)
			done := sseEvent{data: []byte(`{"type":"response.output_item.done","output_index":0,"item":` + tc.item + `}`)}
			if first := state.read(done); len(first.deltas) != 1 || first.deltas[0].Search == nil {
				t.Fatalf("first = %+v, want the search", first)
			}
			final := state.read(sseEvent{data: []byte(`{"type":"response.completed","response":{"status":"completed","output":[` + tc.item + `]}}`)})
			if len(final.deltas) != 1 || !final.deltas[0].Done {
				t.Errorf("final = %+v, want only the final delta", final)
			}
		})
	}
}

func TestACallerWhoLeftIsSentNothing(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	out := make(chan port.Delta, 4)
	client := New(nil, nil)
	live := &liveStream{
		body:    io.NopCloser(strings.NewReader("")),
		events:  newSSEReader(strings.NewReader(""), 1<<10),
		state:   newStreamState(tierDefault),
		release: func() {},
		pending: []port.Delta{{Text: "never"}},
	}
	client.pump(ctx, exchange{}, live, out)
	client.fail(ctx, exchange{}, out, io.ErrUnexpectedEOF)
	live.close()

	if send(ctx, out, port.Delta{Text: "late"}) {
		t.Error("send reported a delivery to a caller who left")
	}
	if len(out) != 0 {
		t.Errorf("%d deltas reached a caller who left", len(out))
	}
}

func TestABrokenStreamSaysHowItBroke(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		err  error
		code errors.Code
	}{
		{name: "the end came early", err: io.EOF, code: errors.External},
		{name: "an event past the cap", err: errEventTooLarge, code: errors.External},
	}
	for _, tc := range cases {
		if got := streamBroken(tc.err); !errors.IsCode(got, tc.code) {
			t.Errorf("%s: streamBroken = %v, want %s", tc.name, got, tc.code)
		}
	}
	if got := streamBroken(io.ErrUnexpectedEOF); !stderrors.Is(got, io.ErrUnexpectedEOF) {
		t.Errorf("streamBroken = %v, want a transport failure passed on for classification", got)
	}
}

func TestTheEventReaderReadsLinesLongerThanItsBuffer(t *testing.T) {
	t.Parallel()

	long := strings.Repeat("y", 200<<10)
	event, err := newSSEReader(strings.NewReader("data: "+long+"\n\n"), 1<<20).next()
	if err != nil {
		t.Fatalf("next: %v", err)
	}
	if string(event.data) != long {
		t.Errorf("data has %d bytes, want %d", len(event.data), len(long))
	}
}
