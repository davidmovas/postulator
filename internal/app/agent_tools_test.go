package app_test

import (
	"encoding/json"
	"slices"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/llm/openai/openaitest"
	"github.com/davidmovas/postulator/internal/app"
	"github.com/davidmovas/postulator/internal/application/agent"
)

func answered(t *testing.T, core *app.Core, conversationID string) string {
	t.Helper()

	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		listed, err := core.Agent.ListMessages(t.Context(), agent.ListMessagesRequest{ConversationID: conversationID})
		if err != nil {
			t.Fatalf("ListMessages: %v", err)
		}
		for i := range listed.Items {
			if listed.Items[i].Role == "assistant" && listed.Items[i].Text != "" {
				return listed.Items[i].Text
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	return ""
}

func offeredTools(t *testing.T, body map[string]any) (whole, namespaces []string, searchable bool) {
	t.Helper()

	listed, ok := body["tools"].([]any)
	if !ok {
		t.Fatalf("the round offers no tools: %v", body["tools"])
	}
	for _, raw := range listed {
		tool, isTool := raw.(map[string]any)
		if !isTool {
			t.Fatalf("a tool is %v", raw)
		}
		name, named := tool["name"].(string)
		if !named && tool["type"] != "tool_search" {
			t.Fatalf("a tool carries no name: %v", tool)
		}
		switch tool["type"] {
		case "function":
			whole = append(whole, name)
		case "namespace":
			namespaces = append(namespaces, name)
		case "tool_search":
			searchable = true
		}
	}
	return whole, namespaces, searchable
}

func TestAChatTurnLoadsItsToolsOnDemandWhenTheSettingSaysSo(t *testing.T) {
	t.Parallel()

	core, server := provided(t)
	if _, err := core.Declarations.Apply(core.Settings, map[string]json.RawMessage{
		"agent.toolLoading": json.RawMessage(`"deferred"`),
	}); err != nil {
		t.Fatalf("Apply the setting: %v", err)
	}
	server.Enqueue(
		openaitest.Answer{
			Searches: []openaitest.Search{{Arguments: `{"paths":["models"]}`}},
			Calls:    []openaitest.Call{{ID: "call_models", Namespace: "models", Name: "models_list", Arguments: `{}`}},
		}.Stream(),
		openaitest.Text("Two models are on file.").Stream(),
	)

	opened, err := core.Agent.CreateConversation(t.Context(), agent.CreateConversationRequest{Title: "Models", Mode: "autonomous"})
	if err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}
	if _, err = core.Agent.Send(t.Context(), agent.SendRequest{
		ConversationID: opened.Conversation.ID, Text: "Which models can you call?",
	}); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if got := answered(t, core, opened.Conversation.ID); got != "Two models are on file." {
		t.Fatalf("the turn answered %q", got)
	}

	asked := server.Requests()
	if len(asked) != 2 {
		t.Fatalf("the provider saw %d rounds, want a search with a call and an answer", len(asked))
	}
	whole, namespaces, searchable := offeredTools(t, asked[0].Body)
	if !searchable || !slices.Contains(whole, "sites_list") || slices.Contains(whole, "models_list") {
		t.Fatalf("the first round offers %v whole and tool search %t, want the orienting reads whole beside tool search",
			whole, searchable)
	}
	if !slices.Contains(namespaces, "models") || len(whole)+len(namespaces) >= len(core.Tools.Names()) {
		t.Fatalf("the first round offers the groups %v and %d tools whole, want the rest inside their groups",
			namespaces, len(whole))
	}

	input, ok := asked[1].Body["input"].([]any)
	if !ok {
		t.Fatalf("the second round sent %s", asked[1].Raw)
	}
	kinds := make([]string, 0, len(input))
	namespace := ""
	for _, raw := range input {
		item, isItem := raw.(map[string]any)
		if !isItem {
			t.Fatalf("an input item is %v", raw)
		}
		kinds = append(kinds, kindOf(item))
		if held, carried := item["namespace"].(string); carried && item["type"] == "function_call" {
			namespace = held
		}
	}
	want := []string{"developer", "user", "tool_search_call", "tool_search_output", "function_call", "function_call_output"}
	if !slices.Equal(kinds, want) || namespace != "models" {
		t.Fatalf("the second round sent %v with the call in %q, want %v with the call in models", kinds, namespace, want)
	}
}
