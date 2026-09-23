package runtime

import (
	"context"
	"testing"
)

func TestTheEngineSaysItIsStoppingBeforeItCancelsAnyItem(t *testing.T) {
	t.Parallel()

	engine := New(Deps{}, nil, Config{}, nil, nil)
	engine.running.Store(true)

	_, cancel := context.WithCancel(t.Context())
	defer cancel()

	said := make(chan bool, 2)
	engine.cancels = map[string]map[string]context.CancelFunc{
		"run-1": {
			"item-1": func() { said <- engine.stopping() },
			"item-2": func() { said <- engine.stopping() },
		},
	}
	engine.release = func() { cancel() }

	if engine.stopping() {
		t.Fatal("a running engine says it is stopping")
	}

	engine.Stop()

	close(said)
	seen := 0
	for answered := range said {
		seen++
		if !answered {
			t.Fatal("an item context was cancelled while the engine still said it was running, " +
				"so the step it stops would be settled as a failure instead of handed back")
		}
	}
	if seen != 2 {
		t.Fatalf("%d item contexts were cancelled, want both", seen)
	}
	if !engine.stopping() {
		t.Fatal("a stopped engine does not say so")
	}
}
