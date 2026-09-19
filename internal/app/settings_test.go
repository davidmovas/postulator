package app

import (
	"context"
	"encoding/json"
	"io"
	"slices"
	"testing"

	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/settings"
)

type fakeSource struct {
	stored map[string]json.RawMessage
	err    error
}

func (f fakeSource) All(context.Context) (map[string]json.RawMessage, error) {
	return f.stored, f.err
}

func TestLoadSettings(t *testing.T) {
	t.Parallel()

	registry := settings.New()
	workers := registry.Int("runs.workers", 2, settings.IntRange(1, 16))

	values, unknown, err := LoadSettings(t.Context(), fakeSource{stored: map[string]json.RawMessage{
		"runs.workers": json.RawMessage(`8`),
		"runs.ghost":   json.RawMessage(`1`),
	}}, registry)
	if err != nil {
		t.Fatalf("LoadSettings: %v", err)
	}
	if got := workers.Get(values); got != 8 {
		t.Errorf("runs.workers = %d, want 8", got)
	}
	if !slices.Equal(unknown, []string{"runs.ghost"}) {
		t.Errorf("unknown = %v, want [runs.ghost]", unknown)
	}
}

func TestLoadSettingsDefaults(t *testing.T) {
	t.Parallel()

	registry := settings.New()
	workers := registry.Int("runs.workers", 2, settings.IntRange(1, 16))

	values, unknown, err := LoadSettings(t.Context(), fakeSource{stored: map[string]json.RawMessage{}}, registry)
	if err != nil {
		t.Fatalf("LoadSettings: %v", err)
	}
	if len(unknown) != 0 {
		t.Errorf("unknown = %v, want none", unknown)
	}
	if got := workers.Get(values); got != 2 {
		t.Errorf("runs.workers = %d, want the default 2", got)
	}
}

func TestLoadSettingsRejectsAnOutOfRangeValue(t *testing.T) {
	t.Parallel()

	registry := settings.New()
	registry.Int("runs.workers", 2, settings.IntRange(1, 16))

	_, _, err := LoadSettings(t.Context(), fakeSource{stored: map[string]json.RawMessage{
		"runs.workers": json.RawMessage(`99`),
	}}, registry)
	if !errors.IsCode(err, errors.Invalid) {
		t.Errorf("code = %q, want %q", errors.CodeOf(err), errors.Invalid)
	}
}

func TestLoadSettingsPropagatesTheSourceFailure(t *testing.T) {
	t.Parallel()

	_, _, err := LoadSettings(t.Context(), fakeSource{err: io.ErrUnexpectedEOF}, settings.New())
	if err == nil {
		t.Fatal("the source failure must reach the caller")
	}
}

func TestEverySettingBelongsToADeclaredGroup(t *testing.T) {
	t.Parallel()

	schema, err := settings.Default().Schema()
	if err != nil {
		t.Fatalf("Schema: %v", err)
	}
	if len(schema) == 0 {
		t.Fatal("the composed application declares no setting")
	}

	declared := settings.Groups()
	for _, descriptor := range schema {
		group := settings.GroupOf(descriptor.Key)
		if !slices.Contains(declared, group) {
			t.Fatalf("setting %s belongs to the undeclared group %q; declare it in kernel/settings", descriptor.Key, group)
		}
		if descriptor.Group != string(group) {
			t.Fatalf("descriptor %s reports the group %q", descriptor.Key, descriptor.Group)
		}
	}
}
