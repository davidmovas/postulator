package agent

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestAResultThatArrivesMidTurnIsDeliveredWhenTheTurnEnds(t *testing.T) {
	t.Parallel()

	turns := NewTurns(func() time.Duration { return time.Minute })

	var (
		mu        sync.Mutex
		resumed   []string
		answered  = make(chan struct{})
		delivered = make(chan struct{})
	)
	turns.Resuming(func(conversationID, text string) {
		mu.Lock()
		resumed = append(resumed, conversationID+": "+text)
		mu.Unlock()
		close(delivered)
	})

	if err := turns.Start(t.Context(), "c1", "m1", time.Now(), func(context.Context) {
		<-answered
	}); err != nil {
		t.Fatalf("Start: %v", err)
	}

	if !turns.Queue("c1", "the confirmed tool pages_create ran") {
		t.Fatal("a result that arrived while the turn was answering was not queued")
	}
	if !turns.Queue("c1", "the confirmed tool pages_update ran") {
		t.Fatal("the second result was not queued")
	}
	if turns.Queue("c2", "another conversation") {
		t.Fatal("a conversation that is not answering must start its turn at once")
	}

	close(answered)
	select {
	case <-delivered:
	case <-time.After(5 * time.Second):
		t.Fatal("the queued results never reached the conversation")
	}

	mu.Lock()
	defer mu.Unlock()
	want := "c1: the confirmed tool pages_create ran\n\nthe confirmed tool pages_update ran"
	if len(resumed) != 1 || resumed[0] != want {
		t.Fatalf("the conversation was told %q, want %q", resumed, want)
	}
}

func TestAClosedRegistryDeliversNothing(t *testing.T) {
	t.Parallel()

	turns := NewTurns(func() time.Duration { return time.Minute })
	turns.Resuming(func(string, string) {
		t.Error("a shutting down agent must not start another turn")
	})

	if err := turns.Start(t.Context(), "c1", "m1", time.Now(), func(context.Context) {}); err != nil {
		t.Fatalf("Start: %v", err)
	}
	turns.Close()

	if turns.Queue("c1", "too late") {
		t.Fatal("a closed registry must refuse to queue")
	}
}
