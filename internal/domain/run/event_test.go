package run_test

import (
	"testing"

	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func TestNewEvent(t *testing.T) {
	t.Parallel()

	event, err := run.NewEvent(run.Event{RunID: "r1", Seq: 1, Type: "run.started", At: stamp})
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}
	if string(event.Payload) != "{}" {
		t.Errorf("Payload = %s, want an empty object", event.Payload)
	}

	cases := []struct {
		name  string
		event run.Event
	}{
		{name: "no run", event: run.Event{Seq: 1, Type: "run.started", At: stamp}},
		{name: "no sequence", event: run.Event{RunID: "r", Type: "run.started", At: stamp}},
		{name: "no type", event: run.Event{RunID: "r", Seq: 1, At: stamp}},
		{name: "no timestamp", event: run.Event{RunID: "r", Seq: 1, Type: "run.started"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if _, err := run.NewEvent(tc.event); !errors.IsCode(err, errors.Invalid) {
				t.Fatalf("NewEvent = %v, want an invalid error", err)
			}
		})
	}
}
