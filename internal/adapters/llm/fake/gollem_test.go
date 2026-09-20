package fake_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/gollem-dev/gollem"

	"github.com/davidmovas/postulator/internal/adapters/llm/fake"
	"github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func session(t *testing.T, client *fake.Gollem, options ...gollem.SessionOption) gollem.Session {
	t.Helper()

	opened, err := client.NewSession(t.Context(), options...)
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	return opened
}

func TestTheScriptedClientWalksItsToolDirectives(t *testing.T) {
	t.Parallel()

	client := fake.NewGollem()
	opened := session(t, client, gollem.WithSessionSystemPrompt("you are a fake"))

	first, err := opened.Generate(t.Context(), []gollem.Input{
		gollem.Text(`TOOL:pages_tree{"a":1}` + "\n" + `TOOL:sites_list{}` + "\nFAKE: all done"),
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if len(first.FunctionCalls) != 1 || first.FunctionCalls[0].Name != "pages_tree" {
		t.Fatalf("the first answer is %+v", first)
	}
	if first.FunctionCalls[0].Arguments["a"] == nil {
		t.Fatalf("the arguments are %v", first.FunctionCalls[0].Arguments)
	}

	second, err := opened.Generate(t.Context(), []gollem.Input{
		gollem.FunctionResponse{ID: first.FunctionCalls[0].ID, Name: "pages_tree", Data: map[string]any{"ok": true}},
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if len(second.FunctionCalls) != 1 || second.FunctionCalls[0].Name != "sites_list" {
		t.Fatalf("the second answer is %+v", second)
	}

	third, err := opened.Generate(t.Context(), []gollem.Input{
		gollem.FunctionResponse{ID: second.FunctionCalls[0].ID, Name: "sites_list", Error: errors.New(errors.External, "down")},
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if len(third.Texts) != 1 || third.Texts[0] != "all done" {
		t.Fatalf("the final answer is %+v", third)
	}

	history, err := opened.History()
	if err != nil || len(history.Messages) == 0 {
		t.Fatalf("History = %+v, %v", history, err)
	}
	scripted, ok := opened.(*fake.GollemSession)
	if !ok || scripted.System() != "you are a fake" {
		t.Error("the session forgot its system prompt")
	}
}

func TestTheScriptedClientAnswersWithoutADirective(t *testing.T) {
	t.Parallel()

	opened := session(t, fake.NewGollem())
	answered, err := opened.Generate(t.Context(), []gollem.Input{gollem.Text("hello")})
	if err != nil || len(answered.Texts) != 1 {
		t.Fatalf("Generate = %+v, %v", answered, err)
	}

	withResults, err := opened.Generate(t.Context(), []gollem.Input{
		gollem.FunctionResponse{ID: "c1", Name: "pages_tree", Data: map[string]any{}},
	})
	if err != nil || withResults.Texts[0] != "ran pages_tree" {
		t.Fatalf("Generate after a tool = %+v, %v", withResults, err)
	}
}

func TestTheScriptedClientFailsOnDemand(t *testing.T) {
	t.Parallel()

	opened := session(t, fake.NewGollem())
	if _, err := opened.Generate(t.Context(), []gollem.Input{gollem.Text("ERROR:RATE_LIMITED")}); !errors.IsCode(err, errors.RateLimited) {
		t.Fatalf("Generate = %v, want the scripted failure", err)
	}
	if _, err := opened.Stream(t.Context(), []gollem.Input{gollem.Text("ERROR:EXTERNAL")}); !errors.IsCode(err, errors.External) {
		t.Fatalf("Stream = %v, want the scripted failure", err)
	}
}

func TestTheScriptedClientStreamsAndCounts(t *testing.T) {
	t.Parallel()

	client := fake.NewGollem()
	opened := session(t, client)

	chunks, err := opened.Stream(t.Context(), []gollem.Input{gollem.Text("FAKE: streamed")})
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}

	texts := 0
	for chunk := range chunks {
		texts += len(chunk.Texts)
	}
	if texts != 1 {
		t.Fatalf("the stream carried %d texts", texts)
	}

	counted, err := opened.CountToken(t.Context(), gollem.Text("some words"))
	if err != nil || counted == 0 {
		t.Fatalf("CountToken = %d, %v", counted, err)
	}
	if err = opened.AppendHistory(nil); err != nil {
		t.Fatalf("AppendHistory of nothing: %v", err)
	}
	if err = opened.AppendHistory(&gollem.History{Version: gollem.HistoryVersion}); err != nil {
		t.Fatalf("AppendHistory: %v", err)
	}

	if _, err = client.GenerateEmbedding(t.Context(), 3, []string{"a"}); err == nil {
		t.Error("the scripted client does not embed")
	}
	if len(client.Sessions()) != 1 {
		t.Fatalf("the client opened %d sessions", len(client.Sessions()))
	}
}

func TestTheScriptedClientReplaysAStoredHistory(t *testing.T) {
	t.Parallel()

	loaded := &gollem.History{
		LLType:  gollem.LLMTypeOpenAI,
		Version: gollem.HistoryVersion,
		Messages: []gollem.Message{{
			Role: gollem.RoleUser, Contents: []gollem.MessageContent{mustText(t, "an older question")},
		}},
	}

	opened := session(t, fake.NewGollem(), gollem.WithSessionHistory(loaded))
	if _, err := opened.Generate(t.Context(), []gollem.Input{gollem.Text("FAKE: answered")}); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	history, err := opened.History()
	if err != nil || len(history.Messages) < 2 {
		t.Fatalf("History = %+v, %v", history, err)
	}
}

func mustText(t *testing.T, text string) gollem.MessageContent {
	t.Helper()

	content, err := gollem.NewTextContent(text)
	if err != nil {
		t.Fatalf("NewTextContent: %v", err)
	}
	return content
}

func TestTheScriptedClientAnswersFromAScript(t *testing.T) {
	t.Parallel()

	script := func(prompt string) fake.Turn {
		if strings.Contains(prompt, "rename") {
			return fake.Turn{
				Tool: "pages_update",
				Args: json.RawMessage(`{"pageId":"p-1","title":"Best espresso machines under $500"}`),
				Text: "I retitled the page.",
			}
		}
		return fake.Turn{Text: "Your graph covers eleven espresso topics."}
	}

	cases := []struct {
		name      string
		prompt    string
		wantCalls int
		wantText  string
	}{
		{name: "prose only", prompt: "how does my graph look?", wantText: "Your graph covers eleven espresso topics."},
		{name: "one write call", prompt: "please rename that page", wantCalls: 1, wantText: "I retitled the page."},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			opened := session(t, fake.NewGollem(fake.WithScript(script)))
			answered, err := opened.Generate(t.Context(), []gollem.Input{gollem.Text(tc.prompt)})
			if err != nil {
				t.Fatalf("Generate: %v", err)
			}
			if len(answered.FunctionCalls) != tc.wantCalls {
				t.Fatalf("calls = %+v, want %d", answered.FunctionCalls, tc.wantCalls)
			}

			if tc.wantCalls > 0 {
				if answered.FunctionCalls[0].Name != "pages_update" {
					t.Fatalf("call = %+v", answered.FunctionCalls[0])
				}
				if answered.FunctionCalls[0].Arguments["pageId"] != "p-1" {
					t.Fatalf("arguments = %v", answered.FunctionCalls[0].Arguments)
				}
				answered, err = opened.Generate(t.Context(), []gollem.Input{
					gollem.FunctionResponse{ID: answered.FunctionCalls[0].ID, Name: "pages_update", Data: map[string]any{"ok": true}},
				})
				if err != nil {
					t.Fatalf("Generate: %v", err)
				}
			}
			if len(answered.Texts) != 1 || answered.Texts[0] != tc.wantText {
				t.Fatalf("texts = %v, want %q", answered.Texts, tc.wantText)
			}
		})
	}
}

func TestADirectiveOutranksTheScript(t *testing.T) {
	t.Parallel()

	client := fake.NewGollem(fake.WithScript(func(string) fake.Turn {
		return fake.Turn{Text: "the script answered"}
	}))

	opened := session(t, client)
	answered, err := opened.Generate(t.Context(), []gollem.Input{gollem.Text("FAKE: the directive answered")})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if len(answered.Texts) != 1 || answered.Texts[0] != "the directive answered" {
		t.Fatalf("texts = %v", answered.Texts)
	}
}

func TestTheScriptedClientIsItsOwnFactory(t *testing.T) {
	t.Parallel()

	client := fake.NewGollem()
	built, err := client.New(t.Context(), llm.ModelRef{Provider: "openai", Model: "gpt-5-mini"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if built != gollem.LLMClient(client) {
		t.Fatalf("New returned %T, want the client itself", built)
	}
}
