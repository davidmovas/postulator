package openaitest_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/llm/openai/openaitest"
)

const minimalBody = `{"model":"gpt-5.6-terra","input":[{"role":"user","content":"hello"}],"store":false}`

type recorder struct {
	errors []string
	mu     sync.Mutex
}

func (r *recorder) Helper() {}

func (r *recorder) Cleanup(func()) {}

func (r *recorder) Errorf(format string, _ ...any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.errors = append(r.errors, format)
}

func (r *recorder) failures() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.errors)
}

type answer struct {
	Header http.Header
	Body   string
	Status int
}

func post(t *testing.T, server *openaitest.Server, path, key, body string) answer {
	t.Helper()
	return postWith(t, t.Context(), server.URL()+path, key, body)
}

func postWith(t *testing.T, ctx context.Context, url, key, body string) answer {
	t.Helper()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(body))
	if err != nil {
		t.Fatalf("build the request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("send the request: %v", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read the answer: %v", err)
	}
	return answer{Status: resp.StatusCode, Header: resp.Header, Body: string(raw)}
}

func decode(t *testing.T, body string) map[string]any {
	t.Helper()

	var decoded map[string]any
	if err := json.Unmarshal([]byte(body), &decoded); err != nil {
		t.Fatalf("decode %q: %v", body, err)
	}
	return decoded
}

func text(t *testing.T, value any) string {
	t.Helper()

	held, ok := value.(string)
	if !ok {
		t.Fatalf("%v is not a string", value)
	}
	return held
}

func errorOf(t *testing.T, body string) map[string]any {
	t.Helper()

	envelope := decode(t, body)
	fault, ok := envelope["error"].(map[string]any)
	if !ok {
		t.Fatalf("the body %s carries no error envelope", body)
	}
	return fault
}

func TestTheServerServesItsQueueInOrderAndRecordsEachRequest(t *testing.T) {
	t.Parallel()

	server := openaitest.New(t)
	server.Enqueue(openaitest.Text("first").Reply(), openaitest.Text("second").Reply())

	for _, want := range []string{"first", "second"} {
		got := post(t, server, "/responses", openaitest.DefaultKey, minimalBody)
		if got.Status != http.StatusOK {
			t.Fatalf("status = %d, want 200: %s", got.Status, got.Body)
		}
		if !strings.Contains(got.Body, want) {
			t.Errorf("answer = %s, want the %s reply", got.Body, want)
		}
	}

	requests := server.Requests()
	if len(requests) != 2 {
		t.Fatalf("recorded %d requests, want 2", len(requests))
	}
	first := requests[0]
	if first.Method != http.MethodPost || first.Path != "/v1/responses" {
		t.Errorf("recorded %s %s, want POST /v1/responses", first.Method, first.Path)
	}
	if !first.Authorized {
		t.Error("a request with the right key was recorded as unauthorized")
	}
	if first.Header.Get("Authorization") != "" {
		t.Errorf("the recorded headers keep the key: %v", first.Header)
	}
	if first.Header.Get("Content-Type") != "application/json" {
		t.Errorf("the recorded headers lost the content type: %v", first.Header)
	}
	if first.Body["model"] != "gpt-5.6-terra" {
		t.Errorf("decoded body = %v, want the model", first.Body)
	}
	if string(first.Raw) != minimalBody {
		t.Errorf("raw body = %s, want the bytes sent", first.Raw)
	}
	if server.Pending() != 0 {
		t.Errorf("pending = %d, want the queue drained", server.Pending())
	}
}

func TestTheServerRefusesWhatTheRealOneRefusesBeforeTheQueue(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		path   string
		key    string
		body   string
		status int
		code   string
	}{
		{name: "another route", path: "/chat/completions", key: openaitest.DefaultKey, body: minimalBody, status: http.StatusNotFound},
		{name: "no key", path: "/responses", body: minimalBody, status: http.StatusUnauthorized, code: "invalid_api_key"},
		{name: "a wrong key", path: "/responses", key: "other", body: minimalBody, status: http.StatusUnauthorized, code: "invalid_api_key"},
		{name: "a body that is not json", path: "/responses", key: openaitest.DefaultKey, body: "{", status: http.StatusBadRequest},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server := openaitest.New(t)
			server.Enqueue(openaitest.Text("never").Reply())

			got := post(t, server, tc.path, tc.key, tc.body)
			if got.Status != tc.status {
				t.Fatalf("status = %d, want %d: %s", got.Status, tc.status, got.Body)
			}
			fault := errorOf(t, got.Body)
			if tc.code != "" && fault["code"] != tc.code {
				t.Errorf("code = %v, want %s", fault["code"], tc.code)
			}
			if server.Pending() != 1 {
				t.Errorf("pending = %d, want the scripted reply kept", server.Pending())
			}
		})
	}
}

func TestTheServerRecordsAWrongKeyAsUnauthorized(t *testing.T) {
	t.Parallel()

	server := openaitest.New(t, openaitest.WithKey("another-test-key"))
	server.Enqueue(openaitest.Text("served").Reply())
	post(t, server, "/responses", openaitest.DefaultKey, minimalBody)

	requests := server.Requests()
	if len(requests) != 1 || requests[0].Authorized {
		t.Fatalf("requests = %+v, want one unauthorized request", requests)
	}
	if got := post(t, server, "/responses", "another-test-key", minimalBody); got.Status != http.StatusOK {
		t.Errorf("status = %d, want the scripted reply once the key is right", got.Status)
	}
}

func TestAnUnexpectedRequestFailsTheTest(t *testing.T) {
	t.Parallel()

	tb := &recorder{}
	server := openaitest.New(tb)
	defer server.Close()

	got := post(t, server, "/responses", openaitest.DefaultKey, minimalBody)
	if got.Status != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", got.Status)
	}
	if tb.failures() != 1 {
		t.Errorf("failures = %d, want the unexpected request reported", tb.failures())
	}
}

func TestAReplyCarriesItsStatusAndHeaders(t *testing.T) {
	t.Parallel()

	server := openaitest.New(t)
	server.Enqueue(openaitest.Failure(http.StatusTooManyRequests, openaitest.RateLimit()).
		WithHeader("Retry-After", "7").
		WithHeader("x-ratelimit-reset-tokens", "6m0s"))

	got := post(t, server, "/responses", openaitest.DefaultKey, minimalBody)
	if got.Status != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429", got.Status)
	}
	if got.Header.Get("Retry-After") != "7" || got.Header.Get("X-Ratelimit-Reset-Tokens") != "6m0s" {
		t.Errorf("headers = %v, want the scripted ones", got.Header)
	}
	if fault := errorOf(t, got.Body); fault["type"] != "rate_limit_error" || fault["code"] != "rate_limit_exceeded" {
		t.Errorf("fault = %v, want a rate limit", fault)
	}
}

func TestASlowReplyGivesUpWhenTheCallerLeaves(t *testing.T) {
	t.Parallel()

	server := openaitest.New(t)
	server.Enqueue(openaitest.Text("late").Reply().After(10 * time.Second))

	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, server.URL()+"/responses", strings.NewReader(minimalBody))
	if err != nil {
		t.Fatalf("build the request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+openaitest.DefaultKey)

	started := time.Now()
	resp, err := http.DefaultClient.Do(req)
	if err == nil {
		resp.Body.Close()
		t.Fatal("the slow reply answered before the caller left")
	}
	if waited := time.Since(started); waited > 5*time.Second {
		t.Errorf("the caller waited %s for a reply it had abandoned", waited)
	}
}

func TestASlowReplyAnswersWhenItsTimeComes(t *testing.T) {
	t.Parallel()

	server := openaitest.New(t)
	server.Enqueue(openaitest.Text("late").Reply().After(30 * time.Millisecond))

	started := time.Now()
	got := post(t, server, "/responses", openaitest.DefaultKey, minimalBody)
	if got.Status != http.StatusOK || !strings.Contains(got.Body, "late") {
		t.Fatalf("answer = %d %s, want the late reply", got.Status, got.Body)
	}
	if waited := time.Since(started); waited < 30*time.Millisecond {
		t.Errorf("the reply came after %s, want at least the scripted delay", waited)
	}
}

type frame struct {
	name string
	data string
}

func frames(t *testing.T, body string) []frame {
	t.Helper()

	var (
		read    []frame
		current frame
	)
	scanner := bufio.NewScanner(strings.NewReader(body))
	scanner.Buffer(make([]byte, 0, 64<<10), 16<<20)
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case line == "":
			if current.name != "" || current.data != "" {
				read = append(read, current)
			}
			current = frame{}
		case strings.HasPrefix(line, "event: "):
			current.name = strings.TrimPrefix(line, "event: ")
		case strings.HasPrefix(line, "data: "):
			current.data = strings.TrimPrefix(line, "data: ")
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("read the frames: %v", err)
	}
	return read
}

func TestAStreamedReplyIsServedAsServerSentEvents(t *testing.T) {
	t.Parallel()

	server := openaitest.New(t)
	server.Enqueue(openaitest.Stream(openaitest.Event{Name: "response.created", Data: `{"type":"response.created"}`},
		openaitest.Event{Name: "response.completed", Data: `{"type":"response.completed"}`, Pause: 10 * time.Millisecond}))

	got := post(t, server, "/responses", openaitest.DefaultKey, minimalBody)
	if got.Status != http.StatusOK {
		t.Fatalf("status = %d, want 200", got.Status)
	}
	if !strings.HasPrefix(got.Header.Get("Content-Type"), "text/event-stream") {
		t.Errorf("content type = %q, want an event stream", got.Header.Get("Content-Type"))
	}
	read := frames(t, got.Body)
	if len(read) != 2 || read[0].name != "response.created" || read[1].data != `{"type":"response.completed"}` {
		t.Errorf("frames = %+v, want the two scripted events", read)
	}
}

type hooks struct {
	seen []string
	mu   sync.Mutex
}

func (h *hooks) mark(name string) func() {
	return func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		h.seen = append(h.seen, name)
	}
}

func (h *hooks) marked() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]string(nil), h.seen...)
}

func TestAReplySaysWhenItIsReached(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		reply func(h *hooks) openaitest.Reply
		want  []string
		event string
	}{
		{
			name:  "a json reply runs its arrival hook before it answers",
			reply: func(h *hooks) openaitest.Reply { return openaitest.Text("hi").Reply().OnArrival(h.mark("arrived")) },
			want:  []string{"arrived"},
		},
		{
			name: "a stream runs its arrival hook, then each event's hook before that event",
			reply: func(h *hooks) openaitest.Reply {
				return openaitest.Stream(
					openaitest.Event{Name: "response.created", Data: `{"type":"response.created"}`, Before: h.mark("created")},
					openaitest.Event{Name: "response.completed", Data: `{"type":"response.completed"}`, Before: h.mark("completed")},
				).OnArrival(h.mark("arrived"))
			},
			want:  []string{"arrived", "created", "completed"},
			event: "response.completed",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			seen := &hooks{}
			server := openaitest.New(t)
			server.Enqueue(tc.reply(seen))

			got := post(t, server, "/responses", openaitest.DefaultKey, minimalBody)
			if got.Status != http.StatusOK {
				t.Fatalf("status = %d, want 200", got.Status)
			}
			if marked := seen.marked(); strings.Join(marked, ",") != strings.Join(tc.want, ",") {
				t.Errorf("hooks ran %v, want %v", marked, tc.want)
			}
			if tc.event != "" && !strings.Contains(got.Body, tc.event) {
				t.Errorf("the stream lost %s after its hook: %s", tc.event, got.Body)
			}
		})
	}
}

func TestAPausedStreamStopsWhenTheCallerLeaves(t *testing.T) {
	t.Parallel()

	server := openaitest.New(t)
	server.Enqueue(openaitest.Stream(openaitest.Event{Name: "response.created", Data: `{"type":"response.created"}`},
		openaitest.Event{Name: "response.completed", Data: `{"type":"response.completed"}`, Pause: 10 * time.Second}))

	ctx, cancel := context.WithCancel(t.Context())
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, server.URL()+"/responses", bytes.NewReader([]byte(minimalBody)))
	if err != nil {
		t.Fatalf("build the request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+openaitest.DefaultKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("send the request: %v", err)
	}
	defer resp.Body.Close()

	reader := bufio.NewReader(resp.Body)
	if _, err = reader.ReadString('\n'); err != nil {
		t.Fatalf("read the first frame: %v", err)
	}
	cancel()

	started := time.Now()
	if _, err = io.ReadAll(reader); err == nil {
		t.Error("the stream ran on after the caller left")
	}
	if waited := time.Since(started); waited > 5*time.Second {
		t.Errorf("the stream held the caller for %s", waited)
	}
}
