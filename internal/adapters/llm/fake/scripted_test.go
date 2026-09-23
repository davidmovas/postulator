package fake_test

import (
	"strings"
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

	client := fake.NewScripted(fake.Reply{Step: "generate_body", Text: `{"h1":"Steaks"}`})

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

	client := fake.NewScripted(fake.Reply{Step: "chat", Text: "hello there"})

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

func TestScriptedAnswersFromThePromptWhenTheReplyIsAFunction(t *testing.T) {
	t.Parallel()

	client := fake.NewScripted(fake.Reply{
		Step: "generate_body",
		Make: func(req port.Request) string {
			return `{"h1":"` + strings.TrimPrefix(req.Messages[len(req.Messages)-1].Text, "write ") + `"}`
		},
	})

	for _, prompt := range []string{"write Steaks", "write Pasta"} {
		req := scriptedRequest("generate_body")
		req.Messages = []port.Message{{Role: port.RoleUser, Text: prompt}}

		got, err := client.Complete(t.Context(), req)
		if err != nil {
			t.Fatalf("Complete %q: %v", prompt, err)
		}
		want := `{"h1":"` + strings.TrimPrefix(prompt, "write ") + `"}`
		if got.Text != want {
			t.Fatalf("%q answered %q, want %q", prompt, got.Text, want)
		}
	}
}

func TestAScriptedTextOutranksItsOwnFunction(t *testing.T) {
	t.Parallel()

	client := fake.NewScripted(fake.Reply{
		Step: "judge",
		Text: `{"score":1}`,
		Make: func(port.Request) string { return `{"score":0}` },
	})

	got, err := client.Complete(t.Context(), scriptedRequest("judge"))
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if got.Text != `{"score":1}` {
		t.Fatalf("the reply answered %q, want the text it carries", got.Text)
	}
}

func TestScriptedPicksTheReplyThatMatchesThePrompt(t *testing.T) {
	t.Parallel()

	client := fake.NewScripted(
		fake.Reply{Step: "generate_body", Match: "/steaks/", Text: `{"h1":"Steaks"}`},
		fake.Reply{Step: "generate_body", Match: "/pasta/", Text: `{"h1":"Pasta"}`},
		fake.Reply{Step: "generate_body", Text: `{"h1":"Anything"}`},
	)

	cases := map[string]string{
		"write the page /steaks/": `{"h1":"Steaks"}`,
		"write the page /pasta/":  `{"h1":"Pasta"}`,
		"write the page /soups/":  `{"h1":"Anything"}`,
	}

	for prompt, want := range cases {
		req := scriptedRequest("generate_body")
		req.Messages = []port.Message{{Role: port.RoleUser, Text: prompt}}

		got, err := client.Complete(t.Context(), req)
		if err != nil {
			t.Fatalf("Complete %q: %v", prompt, err)
		}
		if got.Text != want {
			t.Fatalf("%q answered %q, want %q", prompt, got.Text, want)
		}
	}
}
