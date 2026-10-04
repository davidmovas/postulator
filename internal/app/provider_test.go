package app_test

import (
	"encoding/json"
	stderrors "errors"
	"net/http"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/llm/openai/openaitest"
	"github.com/davidmovas/postulator/internal/app"
	"github.com/davidmovas/postulator/internal/application/agent"
	"github.com/davidmovas/postulator/internal/application/models"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	storedKey  = "sk-proj-AbC123_xyz-0000000000000000000000000000000000000000abcd"
	probeModel = "gpt-5.6-luna"
)

func provided(t *testing.T) (*app.Core, *openaitest.Server) {
	t.Helper()

	return providedWith(t, storedKey)
}

func providedWith(t *testing.T, serverKey string) (*app.Core, *openaitest.Server) {
	t.Helper()

	server := openaitest.New(t, openaitest.WithKey(serverKey))

	home := t.TempDir()
	cfg := app.Config{DatabasePath: filepath.Join(home, "postulator.db"), KeyDir: home}
	first := openCore(t, cfg)
	if _, err := first.Models.SetProviderKey(t.Context(), models.SetProviderKeyRequest{
		Provider: "openai", APIKey: storedKey,
	}); err != nil {
		t.Fatalf("SetProviderKey: %v", err)
	}
	pointAt(t, first, "llm.openai.baseUrl", server.URL())
	if err := first.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	core := openCore(t, cfg)
	t.Cleanup(func() {
		if closeErr := core.Close(); closeErr != nil {
			t.Errorf("Close: %v", closeErr)
		}
	})
	return core, server
}

func pointAt(t *testing.T, core *app.Core, key, base string) {
	t.Helper()

	encoded, err := json.Marshal(base)
	if err != nil {
		t.Fatalf("encode the base url: %v", err)
	}
	if err = core.SettingsStore.Set(t.Context(), key, encoded); err != nil {
		t.Fatalf("store the base url: %v", err)
	}
}

func lastBody(t *testing.T, server *openaitest.Server) map[string]any {
	t.Helper()

	asked := server.Requests()
	if len(asked) == 0 {
		t.Fatal("the provider was never called")
	}
	if asked[len(asked)-1].Body == nil {
		t.Fatalf("the provider was sent %s, which is not a JSON object", asked[len(asked)-1].Raw)
	}
	return asked[len(asked)-1].Body
}

func TestTheStoredKeyReachesTheProviderByteForByte(t *testing.T) {
	t.Parallel()

	core, server := provided(t)
	server.Enqueue(openaitest.Text("pong").Reply())

	if _, err := core.Models.TestProvider(t.Context(), models.TestProviderRequest{
		Provider: "openai", Model: probeModel,
	}); err != nil {
		t.Fatalf("TestProvider: %v", err)
	}

	asked := server.Requests()
	if len(asked) != 1 || !asked[0].Authorized {
		t.Fatalf("the provider saw %d requests, authorized %v", len(asked), len(asked) == 1 && asked[0].Authorized)
	}
	if asked[0].Path != "/v1/responses" {
		t.Fatalf("the client called %q, want the Responses API under the configured base url", asked[0].Path)
	}
	if store, ok := asked[0].Body["store"].(bool); !ok || store {
		t.Fatalf("store = %v, want false so nothing is kept on the provider's side", asked[0].Body["store"])
	}
}

func TestAKeyChangeIsNotServedFromTheCache(t *testing.T) {
	t.Parallel()

	const rotated = "sk-proj-RotatedRotatedRotatedRotatedRotatedRotatedRotated9999"
	core, server := providedWith(t, rotated)

	if _, err := core.Models.TestProvider(t.Context(), models.TestProviderRequest{
		Provider: "openai", Model: probeModel,
	}); !errors.IsCode(err, errors.Unauthorized) {
		t.Fatalf("TestProvider with the old key = %v, want the provider to refuse it", err)
	}

	if _, err := core.Models.SetProviderKey(t.Context(), models.SetProviderKeyRequest{
		Provider: "openai", APIKey: rotated,
	}); err != nil {
		t.Fatalf("SetProviderKey: %v", err)
	}
	server.Enqueue(openaitest.Text("pong").Reply())

	if _, err := core.Models.TestProvider(t.Context(), models.TestProviderRequest{
		Provider: "openai", Model: probeModel,
	}); err != nil {
		t.Fatalf("TestProvider after the rotation: %v", err)
	}
	if asked := server.Requests(); !asked[len(asked)-1].Authorized {
		t.Fatal("after the rotation the client still sent the old key")
	}
}

func TestTheProviderRefusalTellsTheTruth(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		status   int
		fault    openaitest.Fault
		want     errors.Code
		sentence string
		provider string
	}{
		{
			name:   "a rejected key",
			status: http.StatusUnauthorized,
			fault: openaitest.Fault{
				Type: "invalid_request_error", Code: "invalid_api_key",
				Message: "Incorrect API key provided: " + storedKey + ". " +
					"You can find your API key at https://platform.openai.com/account/api-keys.",
			},
			want:     errors.Unauthorized,
			sentence: "the model provider rejected the api key",
			provider: "Incorrect API key provided: sk-…abcd. You can find your API key at " +
				"https://platform.openai.com/account/api-keys.",
		},
		{
			name:   "a key without access to the model",
			status: http.StatusForbidden,
			fault: openaitest.Fault{
				Type: "invalid_request_error", Code: "model_not_found",
				Message: "Project `proj_x` does not have access to model `gpt-6-astra`.",
			},
			want:     errors.Unauthorized,
			sentence: "the key has no access to this model",
			provider: "Project `proj_x` does not have access to model `gpt-6-astra`.",
		},
		{
			name:   "an unknown model",
			status: http.StatusNotFound,
			fault: openaitest.Fault{
				Type: "invalid_request_error", Code: "model_not_found", Message: "The model `gpt-9` does not exist.",
			},
			want:     errors.NotFound,
			sentence: "the model provider has no such model",
			provider: "The model `gpt-9` does not exist.",
		},
		{
			name:   "an account out of credit",
			status: http.StatusTooManyRequests,
			fault:  openaitest.Quota(),
			want:   errors.NeedsHuman,
			sentence: "the OpenAI account is out of credit or over its spending limit; " +
				"add credits or raise the limit in the OpenAI billing settings, then try again",
			provider: openaitest.Quota().Message,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			core, server := provided(t)
			server.Enqueue(openaitest.Failure(tc.status, tc.fault))

			_, err := core.Models.TestProvider(t.Context(), models.TestProviderRequest{
				Provider: "openai", Model: probeModel,
			})
			if !errors.IsCode(err, tc.want) {
				t.Fatalf("code = %q, want %q (%v)", errors.CodeOf(err), tc.want, err)
			}

			var kernel *errors.Error
			if !stderrors.As(err, &kernel) {
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
			if strings.Contains(told, storedKey) {
				t.Errorf("providerMessage carries the whole key: %q", told)
			}
			if len(server.Requests()) != 1 {
				t.Errorf("the provider was called %d times; a refusal like this is not retried", len(server.Requests()))
			}
		})
	}
}

func TestTheProviderTestProbesTheCheapestModel(t *testing.T) {
	t.Parallel()

	core, server := provided(t)
	server.Enqueue(openaitest.Text("pong").Reply())

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

	if len(server.Requests()) != 1 {
		t.Fatalf("the provider was called %d times, want once", len(server.Requests()))
	}
	if asked := lastBody(t, server); asked["model"] != cheapest {
		t.Fatalf("the probe named %v, want %q", asked["model"], cheapest)
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
		"writer": "gpt-5.6-terra",
		"editor": "gpt-5.6-luna",
		"linker": "gpt-5.6-luna",
		"judge":  "gpt-5.6-luna",
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

func kindOf(item map[string]any) string {
	if kind, typed := item["type"].(string); typed && kind != "" {
		return kind
	}
	if role, spoken := item["role"].(string); spoken {
		return role
	}
	return ""
}

func effortOf(asked map[string]any) (string, bool) {
	reasoning, ok := asked["reasoning"].(map[string]any)
	if !ok {
		return "", false
	}
	effort, ok := reasoning["effort"].(string)
	return effort, ok
}

func TestTheProviderTestPaysForNoReasoning(t *testing.T) {
	t.Parallel()

	for _, model := range []string{"gpt-5.6-luna", "gpt-5.6-terra", "gpt-5.6-sol"} {
		t.Run(model, func(t *testing.T) {
			t.Parallel()

			core, server := provided(t)
			server.Enqueue(openaitest.Text("pong").Reply())
			if _, err := core.Models.TestProvider(t.Context(), models.TestProviderRequest{
				Provider: "openai", Model: model,
			}); err != nil {
				t.Fatalf("TestProvider: %v", err)
			}

			asked := lastBody(t, server)
			if effort, carried := effortOf(asked); !carried || effort != "none" {
				t.Fatalf("reasoning.effort = %q (carried %v), want none sent explicitly", effort, carried)
			}
			if ceiling, ok := asked["max_output_tokens"].(float64); !ok || ceiling > 64 {
				t.Fatalf("max_output_tokens = %v, want a few tokens and no reasoning allowance", asked["max_output_tokens"])
			}
			if text, shaped := asked["text"].(map[string]any); shaped && text["verbosity"] != nil {
				t.Fatalf("the request carries a verbosity nobody asked for: %v", text)
			}
		})
	}
}

func TestAModelWithNoReasoningSendsNoEffort(t *testing.T) {
	t.Parallel()

	core, server := provided(t)
	if _, err := core.Models.UpsertModel(t.Context(), models.UpsertModelRequest{
		Provider: "openai", Model: "gpt-plain", ContextTokens: 200000, MaxOutputTokens: 64000,
		InputUSDPerM: 1, OutputUSDPerM: 2, RPM: 60, TPM: 120000,
	}); err != nil {
		t.Fatalf("UpsertModel: %v", err)
	}
	server.Enqueue(openaitest.Text("pong").Reply())

	if _, err := core.Models.TestProvider(t.Context(), models.TestProviderRequest{
		Provider: "openai", Model: "gpt-plain",
	}); err != nil {
		t.Fatalf("TestProvider: %v", err)
	}

	if effort, carried := effortOf(lastBody(t, server)); carried {
		t.Fatalf("the request carries reasoning.effort %q, want none at all", effort)
	}
}

func TestAModelOfARemovedProviderIsRefused(t *testing.T) {
	t.Parallel()

	home := t.TempDir()
	core := openCore(t, app.Config{DatabasePath: filepath.Join(home, "postulator.db"), KeyDir: home})
	t.Cleanup(func() {
		if closeErr := core.Close(); closeErr != nil {
			t.Errorf("Close: %v", closeErr)
		}
	})

	_, err := core.Models.UpsertModel(t.Context(), models.UpsertModelRequest{
		Provider: "retired", Model: "old-model", ContextTokens: 200000, MaxOutputTokens: 64000,
		InputUSDPerM: 1, OutputUSDPerM: 2, RPM: 60, TPM: 120000,
	})
	if !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("UpsertModel of a removed provider = %v, want INVALID", err)
	}
}

func TestAChatTurnTalksToTheProviderWithItsToolsAndNothingStored(t *testing.T) {
	t.Parallel()

	core, server := provided(t)
	server.Enqueue(
		openaitest.Calls(openaitest.Call{ID: "call_sites", Name: "sites_list", Arguments: `{}`}).Stream(),
		openaitest.Answer{Text: "You have no sites yet.", Chunks: []string{"You have ", "no sites yet."}}.Stream(),
	)

	opened, err := core.Agent.CreateConversation(t.Context(), agent.CreateConversationRequest{
		Title: "Sites on file", Mode: "autonomous",
	})
	if err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}
	if _, err = core.Agent.Send(t.Context(), agent.SendRequest{
		ConversationID: opened.Conversation.ID, Text: "Which sites do I have?",
	}); err != nil {
		t.Fatalf("Send: %v", err)
	}

	answer := ""
	deadline := time.Now().Add(15 * time.Second)
	for answer == "" && time.Now().Before(deadline) {
		listed, listErr := core.Agent.ListMessages(t.Context(), agent.ListMessagesRequest{ConversationID: opened.Conversation.ID})
		if listErr != nil {
			t.Fatalf("ListMessages: %v", listErr)
		}
		for _, message := range listed.Items {
			if message.Role == "assistant" {
				answer = message.Text
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	if answer != "You have no sites yet." {
		t.Fatalf("the turn answered %q", answer)
	}

	asked := server.Requests()
	if len(asked) != 2 {
		t.Fatalf("the provider saw %d rounds, want a call and an answer", len(asked))
	}
	for index, request := range asked {
		body := request.Body
		if store, ok := body["store"].(bool); !ok || store || body["stream"] != true {
			t.Fatalf("round %d asks with store %v and stream %v", index+1, body["store"], body["stream"])
		}
		if effort, carried := effortOf(body); !carried || effort != "none" {
			t.Fatalf("round %d asks with effort %q, want none", index+1, effort)
		}
		if body["prompt_cache_key"] != "chat:autonomous" || body["service_tier"] != "default" {
			t.Fatalf("round %d asks with %v on %v", index+1, body["prompt_cache_key"], body["service_tier"])
		}
		if tools, ok := body["tools"].([]any); !ok || len(tools) != len(core.Tools.Names()) {
			t.Fatalf("round %d offers %d tools of %d", index+1, len(tools), len(core.Tools.Names()))
		}
	}

	input, ok := asked[1].Body["input"].([]any)
	if !ok {
		t.Fatalf("the second round sent %s", asked[1].Raw)
	}
	kinds := make([]string, 0, len(input))
	for _, raw := range input {
		item, isItem := raw.(map[string]any)
		if !isItem {
			t.Fatalf("an input item is %v", raw)
		}
		kinds = append(kinds, kindOf(item))
	}
	if want := []string{"developer", "user", "function_call", "function_call_output"}; !slices.Equal(kinds, want) {
		t.Fatalf("the second round sent %v, want %v", kinds, want)
	}
}
