package sqlite_test

import (
	"context"
	"encoding/json"
	"maps"
	"slices"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func settingsRepo(t *testing.T) (*sqlite.SettingsRepo, *sqlite.Store) {
	t.Helper()

	store := sqlitetest.Open(t)
	return sqlite.NewSettingsRepo(store, clock.NewFake(time.Date(2026, time.September, 17, 8, 30, 0, 0, time.UTC))), store
}

func TestSettingsRepoRoundTrip(t *testing.T) {
	t.Parallel()

	repo, _ := settingsRepo(t)

	value, found, err := repo.Get(t.Context(), "runs.workers")
	if err != nil {
		t.Fatalf("Get on an empty table: %v", err)
	}
	if found || value != nil {
		t.Fatalf("Get = %s, %v, want nil, false", value, found)
	}

	if err = repo.Set(t.Context(), "runs.workers", json.RawMessage(`2`)); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err = repo.Set(t.Context(), "runs.workers", json.RawMessage(`4`)); err != nil {
		t.Fatalf("Set again: %v", err)
	}

	value, found, err = repo.Get(t.Context(), "runs.workers")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !found || string(value) != "4" {
		t.Errorf("Get = %s, %v, want 4, true", value, found)
	}
}

func TestSettingsRepoAll(t *testing.T) {
	t.Parallel()

	repo, _ := settingsRepo(t)

	stored, err := repo.All(t.Context())
	if err != nil {
		t.Fatalf("All on an empty table: %v", err)
	}
	if len(stored) != 0 {
		t.Fatalf("All = %v, want an empty map", stored)
	}

	cases := map[string]string{
		"runs.workers":  `2`,
		"llm.provider":  `"openai"`,
		"agent.useFake": `true`,
	}
	for key, raw := range cases {
		if err = repo.Set(t.Context(), key, json.RawMessage(raw)); err != nil {
			t.Fatalf("Set %s: %v", key, err)
		}
	}

	stored, err = repo.All(t.Context())
	if err != nil {
		t.Fatalf("All: %v", err)
	}
	if !slices.Equal(slices.Sorted(maps.Keys(stored)), []string{"agent.useFake", "llm.provider", "runs.workers"}) {
		t.Fatalf("keys = %v", slices.Sorted(maps.Keys(stored)))
	}
	for key, raw := range cases {
		if string(stored[key]) != raw {
			t.Errorf("%s = %s, want %s", key, stored[key], raw)
		}
	}
}

func TestSettingsRepoRejectsBadInput(t *testing.T) {
	t.Parallel()

	repo, _ := settingsRepo(t)

	cases := []struct {
		name  string
		key   string
		value json.RawMessage
	}{
		{name: "empty key", key: "", value: json.RawMessage(`1`)},
		{name: "broken json", key: "runs.workers", value: json.RawMessage(`{`)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if err := repo.Set(t.Context(), tc.key, tc.value); !errors.IsCode(err, errors.Invalid) {
				t.Errorf("code = %q, want %q", errors.CodeOf(err), errors.Invalid)
			}
		})
	}
}

func TestSettingsRepoWritesInsideTheUnitOfWork(t *testing.T) {
	t.Parallel()

	repo, store := settingsRepo(t)
	sentinel := errors.New(errors.Invalid, "abandon")

	err := store.Do(t.Context(), func(c context.Context) error {
		if setErr := repo.Set(c, "runs.workers", json.RawMessage(`2`)); setErr != nil {
			return setErr
		}
		return sentinel
	})
	if err == nil {
		t.Fatal("Do must return the callback error")
	}

	_, found, err := repo.Get(t.Context(), "runs.workers")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if found {
		t.Error("a rolled back write must not be visible")
	}
}

func TestSettingsRepoStoresTheClockInstant(t *testing.T) {
	t.Parallel()

	repo, store := settingsRepo(t)
	if err := repo.Set(t.Context(), "runs.workers", json.RawMessage(`2`)); err != nil {
		t.Fatalf("Set: %v", err)
	}

	var updated string
	if err := sqlite.ScanUpdatedAt(t.Context(), store, "runs.workers", &updated); err != nil {
		t.Fatalf("read updated_at: %v", err)
	}
	if updated != "2026-09-17T08:30:00Z" {
		t.Errorf("updated_at = %q, want %q", updated, "2026-09-17T08:30:00Z")
	}
}
