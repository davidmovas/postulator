package sqlite_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func secretsRepo(t *testing.T) *sqlite.SecretsRepo {
	t.Helper()

	store := sqlitetest.Open(t)
	return sqlite.NewSecretsRepo(store, clock.NewFake(time.Date(2026, time.September, 17, 8, 30, 0, 0, time.UTC)))
}

func TestSecretsRepoRoundTrip(t *testing.T) {
	t.Parallel()

	repo := secretsRepo(t)
	sealed := []byte{0x76, 0x31, 0x3a, 0x01, 0x02, 0x03}

	if err := repo.Put(t.Context(), "wp.site.token", sealed); err != nil {
		t.Fatalf("Put: %v", err)
	}

	got, err := repo.Get(t.Context(), "wp.site.token")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !bytes.Equal(got, sealed) {
		t.Errorf("Get = %x, want %x", got, sealed)
	}

	replacement := []byte{0x76, 0x31, 0x3a, 0x09}
	if err = repo.Put(t.Context(), "wp.site.token", replacement); err != nil {
		t.Fatalf("Put again: %v", err)
	}
	got, err = repo.Get(t.Context(), "wp.site.token")
	if err != nil {
		t.Fatalf("Get after replace: %v", err)
	}
	if !bytes.Equal(got, replacement) {
		t.Errorf("Get = %x, want %x", got, replacement)
	}

	if err = repo.Delete(t.Context(), "wp.site.token"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err = repo.Get(t.Context(), "wp.site.token"); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("code = %q, want %q", errors.CodeOf(err), errors.NotFound)
	}
}

func TestSecretsRepoRejectsBadInput(t *testing.T) {
	t.Parallel()

	repo := secretsRepo(t)

	cases := []struct {
		name string
		call func() error
	}{
		{name: "empty ref", call: func() error { return repo.Put(t.Context(), "", []byte{1}) }},
		{name: "empty ciphertext", call: func() error { return repo.Put(t.Context(), "a", nil) }},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if err := tc.call(); !errors.IsCode(err, errors.Invalid) {
				t.Errorf("code = %q, want %q", errors.CodeOf(err), errors.Invalid)
			}
		})
	}
}

func TestSecretsRepoDeleteReportsMissing(t *testing.T) {
	t.Parallel()

	repo := secretsRepo(t)
	if err := repo.Delete(t.Context(), "absent"); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("code = %q, want %q", errors.CodeOf(err), errors.NotFound)
	}
}
