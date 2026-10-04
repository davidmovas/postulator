package sqlite

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/kernel/clock"
)

func TestSettingsRepoStoresTheClockInstant(t *testing.T) {
	t.Parallel()

	store := openStore(t, nil)
	repo := NewSettingsRepo(store, clock.NewFake(time.Date(2026, time.September, 17, 8, 30, 0, 0, time.UTC)))
	if err := repo.Set(t.Context(), "runs.workers", json.RawMessage(`2`)); err != nil {
		t.Fatalf("Set: %v", err)
	}

	var updated string
	err := store.reader.QueryRowContext(t.Context(), `SELECT updated_at FROM settings WHERE key = 'runs.workers'`).
		Scan(&updated)
	if err != nil {
		t.Fatalf("read updated_at: %v", err)
	}
	if updated != "2026-09-17T08:30:00Z" {
		t.Errorf("updated_at = %q, want %q", updated, "2026-09-17T08:30:00Z")
	}
}
