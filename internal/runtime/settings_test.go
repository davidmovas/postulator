package runtime_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/settings"
	"github.com/davidmovas/postulator/internal/runtime"
)

func storedValues(t *testing.T, stored map[string]json.RawMessage) *settings.Values {
	t.Helper()

	values := settings.Default().NewValues()
	unknown, err := settings.Default().Apply(values, stored)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if len(unknown) != 0 {
		t.Fatalf("Apply reported %v as undeclared", unknown)
	}
	return values
}

func TestTheRunDeadlineIsADeclaredSetting(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		stored map[string]json.RawMessage
		want   time.Duration
	}{
		{name: "nothing stored", stored: map[string]json.RawMessage{}, want: runtime.DefaultRunDeadline},
		{
			name:   "a stored deadline",
			stored: map[string]json.RawMessage{"runs.deadline": json.RawMessage(`"90m"`)},
			want:   90 * time.Minute,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := runtime.Settings(storedValues(t, tc.stored)).RunDeadline; got != tc.want {
				t.Fatalf("RunDeadline = %s, want %s", got, tc.want)
			}
		})
	}
}

func TestTheRunDeadlineIsBounded(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		value   string
		refused bool
	}{
		{name: "inside the range", value: `"12h"`},
		{name: "below the minimum", value: `"30s"`, refused: true},
		{name: "above the maximum", value: `"1000h"`, refused: true},
		{name: "not a duration", value: `"soon"`, refused: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := settings.Default().Validate("runs.deadline", json.RawMessage(tc.value))
			if tc.refused && err == nil {
				t.Fatalf("Validate(%s) was accepted", tc.value)
			}
			if !tc.refused && err != nil {
				t.Fatalf("Validate(%s): %v", tc.value, err)
			}
		})
	}
}

func TestEnqueueTakesItsDeadlineFromTheSetting(t *testing.T) {
	t.Parallel()

	h := newHarness(t, 1)
	engine := runtime.New(runtime.Deps{
		Runs: h.runs, Items: h.items, Artifacts: h.blobs, Execs: h.execs, Events: h.log,
		Pages: pageRepoOf(h), Specs: h.specs, Spend: h.spend,
		Catalog:    stubCatalog{info: llm.ModelInfo{InputUSDPerM: 1, OutputUSDPerM: 2}},
		Profiles:   stubProfiles{ref: llm.ModelRef{Provider: "openai", Model: "test"}},
		UnitOfWork: h.store, Publisher: h.bus,
	}, mustRegister(t, bodyStep(newCounter())),
		runtime.Settings(storedValues(t, map[string]json.RawMessage{"runs.deadline": json.RawMessage(`"90m"`)})),
		systemClock(), h.logger)

	queued, err := engine.Enqueue(t.Context(), h.newRun([]template.StepSpec{{Name: "generate_body", Enabled: true}}))
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}
	if want := queued.CreatedAt.Add(90 * time.Minute); !queued.DeadlineAt.Equal(want) {
		t.Fatalf("DeadlineAt = %s, want %s", queued.DeadlineAt, want)
	}
}
