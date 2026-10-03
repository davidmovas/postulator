package openai_test

import (
	"bytes"
	"context"
	"encoding/json"
	stderrors "errors"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/iotest"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/llm/openai"
	"github.com/davidmovas/postulator/internal/adapters/llm/openai/openaitest"
	port "github.com/davidmovas/postulator/internal/application/llm"
	"github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type vault map[string]string

func (v vault) Get(_ context.Context, ref string) (string, error) {
	value, ok := v[ref]
	if !ok {
		return "", errors.New(errors.NotFound, "no secret is stored under this reference")
	}
	return value, nil
}

type brokenVault struct{}

func (brokenVault) Get(context.Context, string) (string, error) {
	return "", errors.New(errors.Locked, "the vault is locked")
}

type catalog map[string]llm.ModelInfo

func (c catalog) Lookup(_ context.Context, ref llm.ModelRef) (llm.ModelInfo, error) {
	info, ok := c[ref.String()]
	if !ok {
		return llm.ModelInfo{}, errors.New(errors.NotFound, "the catalog has no such model")
	}
	return info, nil
}

func ref(model string) llm.ModelRef {
	return llm.ModelRef{Provider: openai.Provider, Model: model}
}

func terra() llm.ModelInfo {
	return llm.ModelInfo{
		Ref: ref("gpt-5.6-terra"), ReasoningEffort: llm.EffortMedium,
		ContextTokens: 1_050_000, MaxOutputTokens: 128_000,
		InputUSDPerM: 2, CachedInputUSDPerM: 0.2, CacheWriteUSDPerM: 2.5, OutputUSDPerM: 12,
		FlexInputUSDPerM: 1, FlexCachedInputUSDPerM: 0.1, FlexCacheWriteUSDPerM: 1.25, FlexOutputUSDPerM: 6,
		Reasoning: true,
	}
}

func luna() llm.ModelInfo {
	return llm.ModelInfo{
		Ref: ref("gpt-5.6-luna"), ReasoningEffort: llm.EffortLow,
		ContextTokens: 1_050_000, MaxOutputTokens: 128_000,
		InputUSDPerM: 0.2, CachedInputUSDPerM: 0.02, OutputUSDPerM: 1.2,
		Reasoning: true,
	}
}

func plain() llm.ModelInfo {
	return llm.ModelInfo{Ref: ref("gpt-4.1"), ContextTokens: 1_000_000, MaxOutputTokens: 32_768, InputUSDPerM: 2, OutputUSDPerM: 8}
}

func models() catalog {
	return catalog{terra().Ref.String(): terra(), luna().Ref.String(): luna(), plain().Ref.String(): plain()}
}

func keys() vault {
	return vault{llm.SecretRef(openai.Provider): openaitest.DefaultKey}
}

func newClient(server *openaitest.Server, opts ...openai.Option) *openai.Client {
	return openai.New(keys(), models(), append([]openai.Option{openai.WithBaseURL(server.URL())}, opts...)...)
}

func write(text string) port.Request {
	return port.Request{Ref: ref("gpt-5.6-terra"), Messages: []port.Message{{Role: port.RoleUser, Text: text}}}
}

func assertGolden(t *testing.T, name string, raw []byte) {
	t.Helper()

	var indented bytes.Buffer
	if err := json.Indent(&indented, raw, "", "  "); err != nil {
		t.Fatalf("indent the request %s: %v", raw, err)
	}
	stored, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read the golden %s: %v", name, err)
	}
	want := strings.TrimSpace(strings.ReplaceAll(string(stored), "\r\n", "\n"))
	if got := strings.TrimSpace(indented.String()); got != want {
		t.Errorf("the request sent =\n%s\nwant (%s)\n%s", got, name, want)
	}
}

func only(t *testing.T, server *openaitest.Server) openaitest.Request {
	t.Helper()

	requests := server.Requests()
	if len(requests) != 1 {
		t.Fatalf("the server saw %d requests, want 1", len(requests))
	}
	return requests[0]
}

func kernelOf(t *testing.T, err error) *errors.Error {
	t.Helper()

	var kernel *errors.Error
	if !stderrors.As(err, &kernel) || kernel == nil {
		t.Fatalf("error %v (%T) is not a kernel error", err, err)
	}
	return kernel
}

type section struct {
	Heading string `json:"heading" description:"The H2 heading"`
	Kind    string `json:"kind" enum:"text,list"`
}

type draft struct {
	Words    *int      `json:"words" minimum:"1"`
	Title    string    `json:"title" description:"The title of the page"`
	Note     string    `json:"note,omitempty"`
	Sections []section `json:"sections"`
}

func TestTheWritersCallGoesOutStrictOnFlexWithoutACacheWrite(t *testing.T) {
	t.Parallel()

	schema, err := port.SchemaFor[draft]()
	if err != nil {
		t.Fatalf("SchemaFor: %v", err)
	}

	server := openaitest.New(t)
	server.Enqueue(openaitest.Answer{Text: `{"title":"Koffein","note":null,"words":null,"sections":[]}`, Tier: "flex"}.Reply())

	resp, err := newClient(server).Complete(t.Context(), port.Request{
		Ref:       ref("gpt-5.6-terra"),
		System:    "You write pages.",
		Messages:  []port.Message{{Role: port.RoleUser, Text: "Write the page about caffeine."}},
		Schema:    schema,
		MaxTokens: 6000,
		Meta:      port.CallMeta{Step: "generate_body", RunID: "run-1"},
		Effort:    llm.EffortMedium,
		Tier:      llm.TierFlex,
	})
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if resp.Tier != llm.TierFlex {
		t.Errorf("tier = %s, want the flex the provider served", resp.Tier)
	}
	assertGolden(t, "writer_request.json", only(t, server).Raw)
}

func TestAModelThatDoesNotReasonIsSentNoEffortAndNoSampling(t *testing.T) {
	t.Parallel()

	warm := 0.3
	server := openaitest.New(t)
	server.Enqueue(openaitest.Text("Koffein").Reply())

	if _, err := newClient(server).Complete(t.Context(), port.Request{
		Ref:         ref("gpt-4.1"),
		Messages:    []port.Message{{Role: port.RoleUser, Text: "Name the page."}},
		MaxTokens:   256,
		Temperature: &warm,
		Effort:      llm.EffortLow,
		Tier:        llm.TierFlex,
	}); err != nil {
		t.Fatalf("Complete: %v", err)
	}
	assertGolden(t, "plain_request.json", only(t, server).Raw)
}

func TestAModelWithoutFlexPricesIsSentTheDefaultTier(t *testing.T) {
	t.Parallel()

	server := openaitest.New(t)
	server.Enqueue(openaitest.Text("Tighter.").Reply())

	resp, err := newClient(server).Complete(t.Context(), port.Request{
		Ref:       ref("gpt-5.6-luna"),
		System:    "You edit pages.",
		Messages:  []port.Message{{Role: port.RoleUser, Text: "Tighten the page."}},
		MaxTokens: 600,
		Tier:      llm.TierFlex,
		CacheKey:  "editor:site-1",
	})
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if resp.Tier != llm.TierDefault {
		t.Errorf("tier = %s, want default", resp.Tier)
	}
	assertGolden(t, "noflex_request.json", only(t, server).Raw)
}

func TestTheKeyIsReadOnEveryCallAndSentAsABearer(t *testing.T) {
	t.Parallel()

	server := openaitest.New(t)
	server.Enqueue(openaitest.Text("one").Reply(), openaitest.Text("two").Reply())

	stored := keys()
	client := openai.New(stored, models(), openai.WithBaseURL(server.URL()+"/"))
	if _, err := client.Complete(t.Context(), write("one")); err != nil {
		t.Fatalf("Complete: %v", err)
	}
	stored[llm.SecretRef(openai.Provider)] = "rotated-key"
	if _, err := client.Complete(t.Context(), write("two")); !errors.IsCode(err, errors.Unauthorized) {
		t.Fatalf("Complete with a rotated key = %v, want the server's refusal of it", err)
	}

	requests := server.Requests()
	if len(requests) != 2 || !requests[0].Authorized || requests[1].Authorized {
		t.Fatalf("requests = %+v, want the first key accepted and the rotated one sent", requests)
	}
	first := requests[0]
	if first.Path != "/v1/responses" || first.Header.Get("Content-Type") != "application/json" {
		t.Errorf("request = %s %v, want a JSON post to /v1/responses", first.Path, first.Header)
	}
}

func TestMarkupTravelsAsItIsWritten(t *testing.T) {
	t.Parallel()

	server := openaitest.New(t)
	server.Enqueue(openaitest.Text("ok").Reply())

	req := write("<p>Koffein & Powder</p>")
	req.System = "Answer in <h2> sections."
	if _, err := newClient(server).Complete(t.Context(), req); err != nil {
		t.Fatalf("Complete: %v", err)
	}
	raw := string(only(t, server).Raw)
	for _, want := range []string{`"<p>Koffein & Powder</p>"`, `"Answer in <h2> sections."`} {
		if !strings.Contains(raw, want) {
			t.Errorf("the body %s does not carry %s unescaped", raw, want)
		}
	}
}

func TestACallWithoutAUsableKeyNeverLeaves(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		secrets interface {
			Get(context.Context, string) (string, error)
		}
		want errors.Code
	}{
		{name: "no key stored", secrets: vault{}, want: errors.Unauthorized},
		{name: "an empty key", secrets: vault{llm.SecretRef(openai.Provider): ""}, want: errors.Unauthorized},
		{name: "a vault that cannot answer", secrets: brokenVault{}, want: errors.Locked},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server := openaitest.New(t)
			client := openai.New(tc.secrets, models(), openai.WithBaseURL(server.URL()))

			_, err := client.Complete(t.Context(), write("hello"))
			if !errors.IsCode(err, tc.want) {
				t.Fatalf("Complete = %v (%s), want %s", err, errors.CodeOf(err), tc.want)
			}
			if tc.want == errors.Unauthorized && kernelOf(t, err).Details["provider"] != openai.Provider {
				t.Errorf("details = %v, want the provider named", kernelOf(t, err).Details)
			}
			if len(server.Requests()) != 0 {
				t.Error("a request left without a usable key")
			}
		})
	}
}

func TestARequestTheClientCannotSendIsRefusedAtHome(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		req  port.Request
	}{
		{name: "no messages", req: port.Request{Ref: ref("gpt-5.6-terra")}},
		{name: "another provider", req: port.Request{Ref: llm.ModelRef{Provider: "anthropic", Model: "claude-opus-5"}, Messages: write("x").Messages}},
		{name: "an array answer", req: func() port.Request {
			req := write("x")
			req.Schema = &port.Schema{Type: port.SchemaArray, Items: &port.Schema{Type: port.SchemaString}}
			return req
		}()},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server := openaitest.New(t)
			client := newClient(server)
			if _, err := client.Complete(t.Context(), tc.req); !errors.IsCode(err, errors.Invalid) {
				t.Errorf("Complete = %v, want %s", err, errors.Invalid)
			}
			if len(server.Requests()) != 0 {
				t.Error("a request the client refused still left")
			}
		})
	}
}

func TestAnUnlistedModelIsStillCalled(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		models func(base string) *openai.Client
	}{
		{name: "no catalog at all", models: func(base string) *openai.Client {
			return openai.New(keys(), nil, openai.WithBaseURL(base))
		}},
		{name: "a catalog without the model", models: func(base string) *openai.Client {
			return openai.New(keys(), models(), openai.WithBaseURL(base))
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server := openaitest.New(t)
			server.Enqueue(openaitest.Text("hi").Reply())

			req := write("hello")
			req.Ref = ref("gpt-6-preview")
			req.MaxTokens = 100
			req.Tier = llm.TierFlex
			if _, err := tc.models(server.URL()).Complete(t.Context(), req); err != nil {
				t.Fatalf("Complete: %v", err)
			}
			body := only(t, server).Body
			if body["max_output_tokens"] != float64(100) || body["reasoning"] != nil || body["service_tier"] != "default" {
				t.Errorf("body = %v, want the asked ceiling, no effort and the default tier", body)
			}
		})
	}
}

type transport func(*http.Request) (*http.Response, error)

func (f transport) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func breakingBody(status int) *http.Client {
	return &http.Client{Transport: transport(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: status,
			Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
			Body:       io.NopCloser(iotest.ErrReader(io.ErrUnexpectedEOF)),
			Request:    req,
		}, nil
	})}
}

func TestABodyThatBreaksOffIsARetryableFailure(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		status int
		stream bool
	}{
		{name: "an answer", status: http.StatusOK},
		{name: "a refusal", status: http.StatusServiceUnavailable},
		{name: "a stream", status: http.StatusOK, stream: true},
		{name: "a refused stream", status: http.StatusServiceUnavailable, stream: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			client := openai.New(keys(), models(), openai.WithHTTPClient(breakingBody(tc.status)))
			var err error
			if tc.stream {
				_, err = client.Stream(t.Context(), write("hello"))
			} else {
				_, err = client.Complete(t.Context(), write("hello"))
			}
			if !errors.IsCode(err, errors.External) || kernelOf(t, err).Retry == nil {
				t.Fatalf("error = %v, want a retryable external failure", err)
			}
		})
	}
}

func TestAnAddressThatIsNotAURLIsRefused(t *testing.T) {
	t.Parallel()

	client := openai.New(keys(), models(), openai.WithBaseURL("http://bad host"))
	if _, err := client.Complete(t.Context(), write("hello")); !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("Complete = %v, want %s", err, errors.Invalid)
	}
}

func TestARequestThatCannotBeEncodedIsAnInternalFault(t *testing.T) {
	t.Parallel()

	broken := math.NaN()
	req := write("hello")
	req.Tools = []port.Tool{{Name: "pages_list", Schema: &port.Schema{
		Type: port.SchemaObject, Properties: map[string]*port.Schema{"limit": {Type: port.SchemaInteger, Minimum: &broken}},
	}}}

	server := openaitest.New(t)
	if _, err := newClient(server).Complete(t.Context(), req); !errors.IsCode(err, errors.Internal) {
		t.Fatalf("Complete = %v, want %s", err, errors.Internal)
	}
	if len(server.Requests()) != 0 {
		t.Error("a request that could not be encoded still left")
	}
}

func TestACancelledCallerIsCancelled(t *testing.T) {
	t.Parallel()

	server := openaitest.New(t)
	server.Enqueue(openaitest.Text("late").Reply().After(5 * time.Second))

	ctx, cancel := context.WithCancel(t.Context())
	go func() {
		time.Sleep(30 * time.Millisecond)
		cancel()
	}()

	_, err := newClient(server).Complete(ctx, write("hello"))
	if !errors.IsCode(err, errors.Cancelled) {
		t.Fatalf("Complete = %v (%s), want %s", err, errors.CodeOf(err), errors.Cancelled)
	}
	if kernelOf(t, err).Retry != nil {
		t.Error("a cancelled call carries a retry hint")
	}
}

func TestTheClientsOwnTimeoutIsARetryableFailure(t *testing.T) {
	t.Parallel()

	server := openaitest.New(t)
	server.Enqueue(openaitest.Text("late").Reply().After(5 * time.Second))

	_, err := newClient(server, openai.WithTimeout(30*time.Millisecond)).Complete(t.Context(), write("hello"))
	if !errors.IsCode(err, errors.External) {
		t.Fatalf("Complete = %v (%s), want %s", err, errors.CodeOf(err), errors.External)
	}
	if kernelOf(t, err).Retry == nil {
		t.Error("the client's own timeout is not retryable")
	}
}

func TestACallUnderADeadlineRunsToThatDeadline(t *testing.T) {
	t.Parallel()

	server := openaitest.New(t)
	server.Enqueue(openaitest.Text("in time").Reply().After(120 * time.Millisecond))

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	resp, err := newClient(server, openai.WithTimeout(20*time.Millisecond)).Complete(ctx, write("hello"))
	if err != nil {
		t.Fatalf("a caller with a deadline of its own was cut by the client's timeout: %v", err)
	}
	if resp.Text != "in time" {
		t.Errorf("text = %q, want the answer", resp.Text)
	}
}

func TestAClientWithoutATimeoutWaits(t *testing.T) {
	t.Parallel()

	server := openaitest.New(t)
	server.Enqueue(openaitest.Text("patient").Reply().After(50 * time.Millisecond))

	resp, err := newClient(server, openai.WithTimeout(0), openai.WithHTTPClient(&http.Client{})).Complete(t.Context(), write("hello"))
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if resp.Text != "patient" {
		t.Errorf("text = %q, want the answer", resp.Text)
	}
}

func TestAnUnreachableProviderIsARetryableFailure(t *testing.T) {
	t.Parallel()

	server := openaitest.New(t)
	base := server.URL()
	server.Close()

	_, err := openai.New(keys(), models(), openai.WithBaseURL(base)).Complete(t.Context(), write("hello"))
	if !errors.IsCode(err, errors.External) {
		t.Fatalf("Complete = %v (%s), want %s", err, errors.CodeOf(err), errors.External)
	}
	if kernelOf(t, err).Retry == nil {
		t.Error("an unreachable provider is not retryable")
	}
	if strings.Contains(err.Error(), openaitest.DefaultKey) {
		t.Errorf("the failure %q carries the key", err)
	}
}

func TestTheDefaultsAreTheProvidersOwn(t *testing.T) {
	t.Parallel()

	if openai.DefaultBaseURL != "https://api.openai.com/v1" {
		t.Errorf("the default base URL is %q", openai.DefaultBaseURL)
	}
	if openai.DefaultTimeout <= 0 {
		t.Errorf("the default timeout is %s", openai.DefaultTimeout)
	}
	if openai.New(keys(), models()) == nil {
		t.Error("New returned nothing")
	}
}
