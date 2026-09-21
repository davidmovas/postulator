package app_test

import (
	"encoding/json"
	stderrors "errors"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/davidmovas/postulator/internal/app"
	"github.com/davidmovas/postulator/internal/application/models"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	storedKey  = "sk-proj-AbC123_xyz-0000000000000000000000000000000000000000abcd"
	probeModel = "gpt-5.6-luna"
)

type provider struct {
	mu       sync.Mutex
	headers  []string
	paths    []string
	requests []string
	status   int
	body     string
}

func (p *provider) serve() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		asked, readErr := io.ReadAll(r.Body)
		if readErr != nil {
			http.Error(w, readErr.Error(), http.StatusBadRequest)
			return
		}

		p.mu.Lock()
		p.headers = append(p.headers, r.Header.Get("Authorization"))
		p.paths = append(p.paths, r.URL.Path)
		p.requests = append(p.requests, string(asked))
		status, body := p.status, p.body
		p.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		if status == 0 {
			status = http.StatusOK
		}
		w.WriteHeader(status)
		if body != "" {
			write(w, body)
			return
		}
		write(w, `{"id":"1","object":"chat.completion","model":"`+probeModel+`",`+
			`"choices":[{"index":0,"message":{"role":"assistant","content":"pong"},"finish_reason":"stop"}],`+
			`"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`)
	})
}

func write(w http.ResponseWriter, body string) {
	if _, err := w.Write([]byte(body)); err != nil {
		panic(err)
	}
}

func (p *provider) refuse(status int, message, code string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.status = status
	encoded, err := json.Marshal(map[string]any{
		"error": map[string]any{"message": message, "type": "invalid_request_error", "code": code},
	})
	if err != nil {
		panic(err)
	}
	p.body = string(encoded)
}

func (p *provider) lastRequest() string {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.requests) == 0 {
		return ""
	}
	return p.requests[len(p.requests)-1]
}

func (p *provider) seen() (headers, paths []string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]string(nil), p.headers...), append([]string(nil), p.paths...)
}

func provided(t *testing.T) (*app.Core, *provider) {
	t.Helper()

	fake := &provider{}
	server := httptest.NewServer(fake.serve())
	t.Cleanup(server.Close)

	home := t.TempDir()
	core := openCore(t, app.Config{DatabasePath: filepath.Join(home, "postulator.db"), KeyDir: home})
	t.Cleanup(func() {
		if closeErr := core.Close(); closeErr != nil {
			t.Errorf("Close: %v", closeErr)
		}
	})

	if _, err := core.Models.SetProviderKey(t.Context(), models.SetProviderKeyRequest{
		Provider: "openai", APIKey: storedKey,
	}); err != nil {
		t.Fatalf("SetProviderKey: %v", err)
	}
	pointAt(t, core, server.URL+"/v1")
	return core, fake
}

func pointAt(t *testing.T, core *app.Core, base string) {
	t.Helper()

	encoded, err := json.Marshal(base)
	if err != nil {
		t.Fatalf("encode the base url: %v", err)
	}
	if err = core.SettingsStore.Set(t.Context(), "llm.openai.baseUrl", encoded); err != nil {
		t.Fatalf("store the base url: %v", err)
	}

	stored, err := core.SettingsStore.All(t.Context())
	if err != nil {
		t.Fatalf("read the settings: %v", err)
	}
	if _, err = core.Declarations.Apply(core.Settings, stored); err != nil {
		t.Fatalf("apply the settings: %v", err)
	}
}

func TestTheStoredKeyReachesTheProviderByteForByte(t *testing.T) {
	t.Parallel()

	core, fake := provided(t)

	if _, err := core.Models.TestProvider(t.Context(), models.TestProviderRequest{
		Provider: "openai", Model: probeModel,
	}); err != nil {
		t.Fatalf("TestProvider: %v", err)
	}

	headers, paths := fake.seen()
	if len(headers) == 0 {
		t.Fatal("the provider was never called")
	}
	if want := "Bearer " + storedKey; headers[0] != want {
		t.Fatalf("Authorization = %q, want %q", headers[0], want)
	}
	if paths[0] != "/v1/chat/completions" {
		t.Fatalf("the client called %q, want the configured base url", paths[0])
	}
}

func TestAKeyChangeIsNotServedFromTheCache(t *testing.T) {
	t.Parallel()

	core, fake := provided(t)

	if _, err := core.Models.TestProvider(t.Context(), models.TestProviderRequest{
		Provider: "openai", Model: probeModel,
	}); err != nil {
		t.Fatalf("TestProvider: %v", err)
	}

	const rotated = "sk-proj-RotatedRotatedRotatedRotatedRotatedRotatedRotated9999"
	if _, err := core.Models.SetProviderKey(t.Context(), models.SetProviderKeyRequest{
		Provider: "openai", APIKey: rotated,
	}); err != nil {
		t.Fatalf("SetProviderKey: %v", err)
	}

	if _, err := core.Models.TestProvider(t.Context(), models.TestProviderRequest{
		Provider: "openai", Model: probeModel,
	}); err != nil {
		t.Fatalf("TestProvider after the rotation: %v", err)
	}

	headers, _ := fake.seen()
	if len(headers) < 2 {
		t.Fatalf("the provider was called %d times, want two", len(headers))
	}
	if want := "Bearer " + rotated; headers[len(headers)-1] != want {
		t.Fatalf("after the rotation the client still sent %q, want %q", headers[len(headers)-1], want)
	}
}

func TestTheProviderRefusalTellsTheTruth(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		status   int
		message  string
		code     string
		want     errors.Code
		sentence string
		provider string
	}{
		{
			name:   "a rejected key",
			status: http.StatusUnauthorized,
			message: "Incorrect API key provided: sk-proj-AbC123_xyz-0000000000000000000000000000000000000000abcd. " +
				"You can find your API key at https://platform.openai.com/account/api-keys.",
			code: "invalid_api_key", want: errors.Unauthorized,
			sentence: "the model provider rejected the api key",
			provider: "Incorrect API key provided: sk-…abcd. You can find your API key at " +
				"https://platform.openai.com/account/api-keys.",
		},
		{
			name:    "a key without access to the model",
			status:  http.StatusForbidden,
			message: "Project `proj_x` does not have access to model `gpt-6-astra`.",
			code:    "model_not_found", want: errors.Unauthorized,
			sentence: "the key has no access to this model",
			provider: "Project `proj_x` does not have access to model `gpt-6-astra`.",
		},
		{
			name:    "an unknown model",
			status:  http.StatusNotFound,
			message: "The model `gpt-9` does not exist.",
			code:    "model_not_found", want: errors.NotFound,
			sentence: "the model provider has no such model",
			provider: "The model `gpt-9` does not exist.",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			core, fake := provided(t)
			fake.refuse(tc.status, tc.message, tc.code)

			_, err := core.Models.TestProvider(t.Context(), models.TestProviderRequest{
				Provider: "openai", Model: probeModel,
			})
			if !errors.IsCode(err, tc.want) {
				t.Fatalf("code = %q, want %q (%v)", errors.CodeOf(err), tc.want, err)
			}

			var kernel *errors.Error
			if !asKernelError(err, &kernel) {
				t.Fatalf("the failure is %v, want a kernel error", err)
			}
			if kernel.Message != tc.sentence {
				t.Errorf("message = %q, want %q", kernel.Message, tc.sentence)
			}

			told, ok := kernel.Details["providerMessage"].(string)
			if !ok {
				t.Fatalf("details = %v, want the provider's own message", kernel.Details)
			}
			if told != tc.provider {
				t.Errorf("providerMessage = %q, want %q", told, tc.provider)
			}
			if containsText(told, storedKey) {
				t.Errorf("providerMessage carries the whole key: %q", told)
			}
		})
	}
}

func asKernelError(err error, target **errors.Error) bool {
	return stderrors.As(err, target)
}

func containsText(text, needle string) bool {
	return strings.Contains(text, needle)
}

func TestTheProviderTestProbesTheCheapestModel(t *testing.T) {
	t.Parallel()

	core, fake := provided(t)

	if _, err := core.Models.TestProvider(t.Context(), models.TestProviderRequest{Provider: "openai"}); err != nil {
		t.Fatalf("TestProvider without a model: %v", err)
	}

	listed, err := core.Models.ListModels(t.Context(), models.ListModelsRequest{})
	if err != nil {
		t.Fatalf("ListModels: %v", err)
	}

	cheapest, price := "", 0.0
	for _, model := range listed.Models {
		if model.Provider != "openai" {
			continue
		}
		if cost := model.InputUSDPerM + model.OutputUSDPerM; cheapest == "" || cost < price {
			cheapest, price = model.Model, cost
		}
	}
	if cheapest == "" {
		t.Fatal("the catalog carries no openai model")
	}

	_, paths := fake.seen()
	if len(paths) != 1 {
		t.Fatalf("the provider was called %d times, want once", len(paths))
	}
	if !containsText(fake.lastRequest(), cheapest) {
		t.Fatalf("the probe did not name %q; it sent %s", cheapest, fake.lastRequest())
	}
}

func TestEveryRoleHasADefaultOnAFreshInstall(t *testing.T) {
	t.Parallel()

	home := t.TempDir()
	core := openCore(t, app.Config{DatabasePath: filepath.Join(home, "postulator.db"), KeyDir: home})
	t.Cleanup(func() {
		if closeErr := core.Close(); closeErr != nil {
			t.Errorf("Close: %v", closeErr)
		}
	})

	answered, err := core.Models.GetProfiles(t.Context(), models.GetProfilesRequest{})
	if err != nil {
		t.Fatalf("GetProfiles: %v", err)
	}

	want := map[string]string{
		"writer": "gpt-5.6-sol",
		"editor": "gpt-5.6-terra",
		"linker": "gpt-5.6-luna",
		"judge":  "gpt-5.6-terra",
		"chat":   "gpt-5.6-terra",
		"image":  "gpt-image-2",
		"titler": "gpt-5.6-luna",
	}
	if len(answered.Profiles) != len(want) {
		t.Fatalf("profiles = %d, want %d", len(answered.Profiles), len(want))
	}
	for _, profile := range answered.Profiles {
		expected, known := want[profile.Role]
		if !known {
			t.Errorf("unexpected role %q", profile.Role)
			continue
		}
		if profile.Effective == nil {
			t.Errorf("role %q has no effective model on a fresh install", profile.Role)
			continue
		}
		if profile.Effective.Provider != "openai" || profile.Effective.Model != expected {
			t.Errorf("role %q resolves to %s/%s, want openai/%s",
				profile.Role, profile.Effective.Provider, profile.Effective.Model, expected)
		}
	}
}
