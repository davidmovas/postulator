package ctx_test

import (
	"context"
	"testing"

	"github.com/davidmovas/postulator/internal/kernel/ctx"
)

func TestRunIDRoundTrip(t *testing.T) {
	t.Parallel()

	base := context.Background()
	if got, ok := ctx.RunID(base); ok || got != "" {
		t.Fatalf("RunID(empty) = (%q, %v), want (\"\", false)", got, ok)
	}

	carried := ctx.WithRunID(base, "run-1")
	got, ok := ctx.RunID(carried)
	if !ok || got != "run-1" {
		t.Fatalf("RunID() = (%q, %v), want (\"run-1\", true)", got, ok)
	}

	if _, ok = ctx.RunID(base); ok {
		t.Fatal("the parent context must not be modified")
	}
}

func TestConversationIDRoundTrip(t *testing.T) {
	t.Parallel()

	base := context.Background()
	if got, ok := ctx.ConversationID(base); ok || got != "" {
		t.Fatalf("ConversationID(empty) = (%q, %v), want (\"\", false)", got, ok)
	}

	carried := ctx.WithConversationID(base, "conv-1")
	got, ok := ctx.ConversationID(carried)
	if !ok || got != "conv-1" {
		t.Fatalf("ConversationID() = (%q, %v), want (\"conv-1\", true)", got, ok)
	}
}

func TestActorRoundTrip(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		actor ctx.Actor
		text  string
	}{
		{name: "user", actor: ctx.ActorUser, text: "user"},
		{name: "agent", actor: ctx.ActorAgent, text: "agent"},
		{name: "schedule", actor: ctx.ActorSchedule, text: "schedule"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if tc.actor.String() != tc.text {
				t.Fatalf("String() = %q, want %q", tc.actor.String(), tc.text)
			}
			got, ok := ctx.ActorFrom(ctx.WithActor(context.Background(), tc.actor))
			if !ok || got != tc.actor {
				t.Fatalf("Actor() = (%q, %v), want (%q, true)", got, ok, tc.actor)
			}
		})
	}
}

func TestActorAbsent(t *testing.T) {
	t.Parallel()

	got, ok := ctx.ActorFrom(context.Background())
	if ok || got != "" {
		t.Fatalf("ActorFrom(empty) = (%q, %v), want (\"\", false)", got, ok)
	}
}

func TestValuesDoNotCollide(t *testing.T) {
	t.Parallel()

	full := ctx.WithActor(ctx.WithConversationID(ctx.WithRunID(context.Background(), "run"), "conv"), ctx.ActorAgent)

	if run, _ := ctx.RunID(full); run != "run" {
		t.Fatalf("RunID() = %q", run)
	}
	if conv, _ := ctx.ConversationID(full); conv != "conv" {
		t.Fatalf("ConversationID() = %q", conv)
	}
	if actor, _ := ctx.ActorFrom(full); actor != ctx.ActorAgent {
		t.Fatalf("Actor() = %q", actor)
	}
}
