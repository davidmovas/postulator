package fake_test

import (
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/llm/fake"
	port "github.com/davidmovas/postulator/internal/application/llm"
	"github.com/davidmovas/postulator/internal/domain/llm"
)

func scriptedRequest(step string) port.Request {
	return port.Request{
		Ref:       llm.ModelRef{Provider: "openai", Model: "test"},
		System:    "you write pages",
		Messages:  []port.Message{{Role: port.RoleUser, Text: "write the page"}},
		MaxTokens: 128,
		Meta:      port.CallMeta{Step: step},
	}
}

func TestScriptedAnswersPerStep(t *testing.T) {
	t.Parallel()

	client := fake.NewScripted(map[string]string{"generate_body": `{"h1":"Steaks"}`})

	scripted, err := client.Complete(t.Context(), scriptedRequest("generate_body"))
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if scripted.Text != `{"h1":"Steaks"}` {
		t.Fatalf("text = %q, want the scripted reply", scripted.Text)
	}

	unscripted, err := client.Complete(t.Context(), scriptedRequest("judge"))
	if err != nil {
		t.Fatalf("Complete an unscripted step: %v", err)
	}
	if unscripted.Text == `{"h1":"Steaks"}` {
		t.Fatal("an unscripted step got the scripted reply")
	}

	if client.CallsTo("generate_body") != 1 || client.CallsTo("judge") != 1 || client.CallsTo("publish") != 0 {
		t.Fatalf("the call counts are %d, %d and %d",
			client.CallsTo("generate_body"), client.CallsTo("judge"), client.CallsTo("publish"))
	}
	if len(client.Requests()) != 2 {
		t.Fatalf("the client recorded %d requests, want 2", len(client.Requests()))
	}
}

func TestScriptedStreams(t *testing.T) {
	t.Parallel()

	client := fake.NewScripted(map[string]string{"chat": "hello there"})

	deltas, err := client.Stream(t.Context(), scriptedRequest("chat"))
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}

	text := ""
	for delta := range deltas {
		text += delta.Text
	}
	if text != "hello there" {
		t.Fatalf("the stream carried %q", text)
	}
}
