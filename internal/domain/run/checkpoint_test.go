package run_test

import (
	"encoding/json"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type placement struct {
	Anchor string `json:"anchor"`
	Index  int    `json:"index"`
}

func TestCheckpointRoundTrip(t *testing.T) {
	t.Parallel()

	cp := run.NewCheckpoint()
	if err := run.Set(cp, "placement", placement{Anchor: "espresso", Index: 2}); err != nil {
		t.Fatalf("Set: %v", err)
	}

	encoded, err := cp.Encode()
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}

	decoded, err := run.DecodeCheckpoint(encoded)
	if err != nil {
		t.Fatalf("DecodeCheckpoint: %v", err)
	}

	value, found, err := run.Get[placement](decoded, "placement")
	if err != nil || !found {
		t.Fatalf("Get = %+v, %v, %v", value, found, err)
	}
	if value.Anchor != "espresso" || value.Index != 2 {
		t.Fatalf("Get = %+v", value)
	}
}

func TestCheckpointMissingKey(t *testing.T) {
	t.Parallel()

	cp := run.NewCheckpoint()
	value, found, err := run.Get[placement](cp, "absent")
	if err != nil || found || value != (placement{}) {
		t.Fatalf("Get of a missing key = %+v, %v, %v", value, found, err)
	}
}

func TestCheckpointRejectsBadInput(t *testing.T) {
	t.Parallel()

	if err := run.Set(nil, "placement", 1); !errors.IsCode(err, errors.Internal) {
		t.Fatalf("Set on a nil checkpoint = %v, want an internal error", err)
	}
	if _, err := run.DecodeCheckpoint("not json"); !errors.IsCode(err, errors.Internal) {
		t.Fatalf("DecodeCheckpoint of junk = %v, want an internal error", err)
	}
	if err := run.Set(run.NewCheckpoint(), "cycle", make(chan int)); !errors.IsCode(err, errors.Internal) {
		t.Fatalf("Set of an unencodable value = %v, want an internal error", err)
	}

	cp := run.Checkpoint{"placement": json.RawMessage(`"text"`)}
	if _, _, err := run.Get[placement](cp, "placement"); !errors.IsCode(err, errors.Internal) {
		t.Fatalf("Get of a mistyped value = %v, want an internal error", err)
	}
}

func TestCheckpointEncodeOfNothing(t *testing.T) {
	t.Parallel()

	var cp run.Checkpoint
	encoded, err := cp.Encode()
	if err != nil || encoded != "{}" {
		t.Fatalf("Encode of a nil checkpoint = %q, %v", encoded, err)
	}

	empty, err := run.DecodeCheckpoint("")
	if err != nil || len(empty) != 0 {
		t.Fatalf("DecodeCheckpoint of an empty column = %v, %v", empty, err)
	}
}

func TestCheckpointMergedWithDoesNotMutate(t *testing.T) {
	t.Parallel()

	base := run.Checkpoint{"a": json.RawMessage(`1`), "b": json.RawMessage(`2`)}
	merged := base.MergedWith(run.Checkpoint{"b": json.RawMessage(`3`), "c": json.RawMessage(`4`)})

	if string(base["b"]) != "2" || len(base) != 2 {
		t.Fatalf("MergedWith mutated the receiver: %v", base)
	}
	if string(merged["a"]) != "1" || string(merged["b"]) != "3" || string(merged["c"]) != "4" {
		t.Fatalf("MergedWith = %v", merged)
	}
}
