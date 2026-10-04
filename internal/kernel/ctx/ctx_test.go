package ctx_test

import (
	"context"
	"testing"

	"github.com/davidmovas/postulator/internal/kernel/ctx"
)

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

func TestTheParentKeepsNoActor(t *testing.T) {
	t.Parallel()

	base := context.Background()
	carried := ctx.WithActor(base, ctx.ActorAgent)
	if actor, _ := ctx.ActorFrom(carried); actor != ctx.ActorAgent {
		t.Fatalf("Actor() = %q", actor)
	}
	if _, ok := ctx.ActorFrom(base); ok {
		t.Fatal("the parent context must not be modified")
	}
}
