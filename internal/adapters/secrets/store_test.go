package secrets

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/secrets/aesgcm"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type memoryVault struct {
	rows map[string][]byte
}

func newMemoryVault() *memoryVault {
	return &memoryVault{rows: make(map[string][]byte)}
}

func (v *memoryVault) Put(_ context.Context, ref string, ciphertext []byte) error {
	v.rows[ref] = bytes.Clone(ciphertext)
	return nil
}

func (v *memoryVault) Get(_ context.Context, ref string) ([]byte, error) {
	ciphertext, ok := v.rows[ref]
	if !ok {
		return nil, errors.New(errors.NotFound, "secret "+ref+" is not stored")
	}
	return ciphertext, nil
}

func (v *memoryVault) Delete(_ context.Context, ref string) error {
	if _, ok := v.rows[ref]; !ok {
		return errors.New(errors.NotFound, "secret "+ref+" is not stored")
	}
	delete(v.rows, ref)
	return nil
}

func testKey() []byte {
	return bytes.Repeat([]byte{0x5a}, aesgcm.KeyLength)
}

func TestStoreRoundTrip(t *testing.T) {
	t.Parallel()

	vault := newMemoryVault()
	store := NewStore(vault, testKey())

	if err := store.Put(t.Context(), "wp.site.token", "abcd EFGH 1234"); err != nil {
		t.Fatalf("Put: %v", err)
	}

	stored := vault.rows["wp.site.token"]
	if !bytes.HasPrefix(stored, []byte(aesgcm.Version)) {
		t.Fatalf("stored = %x, want the %q prefix", stored, aesgcm.Version)
	}
	if strings.Contains(string(stored), "abcd EFGH 1234") {
		t.Fatal("the column must not hold the plaintext")
	}

	value, err := store.Get(t.Context(), "wp.site.token")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if value != "abcd EFGH 1234" {
		t.Errorf("Get = %q, want %q", value, "abcd EFGH 1234")
	}

	if err = store.Delete(t.Context(), "wp.site.token"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err = store.Get(t.Context(), "wp.site.token"); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("code = %q, want %q", errors.CodeOf(err), errors.NotFound)
	}
}

func TestStoreReportsWhetherASecretIsHeldWithoutOpeningIt(t *testing.T) {
	t.Parallel()

	vault := newMemoryVault()
	store := NewStore(vault, testKey())
	if err := store.Put(t.Context(), "llm:openai:api_key", "sk-secret"); err != nil {
		t.Fatalf("Put: %v", err)
	}

	cases := []struct {
		name string
		ref  string
		want bool
	}{
		{name: "a reference the vault holds", ref: "llm:openai:api_key", want: true},
		{name: "a reference the vault does not hold", ref: "llm:anthropic:api_key"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			held, err := store.Has(t.Context(), tc.ref)
			if err != nil {
				t.Fatalf("Has: %v", err)
			}
			if held != tc.want {
				t.Fatalf("Has(%q) = %v, want %v", tc.ref, held, tc.want)
			}
		})
	}

	if _, err := NewStore(vault, bytes.Repeat([]byte{0x01}, aesgcm.KeyLength)).Has(t.Context(), "llm:openai:api_key"); err != nil {
		t.Fatalf("Has must not open the secret it reports: %v", err)
	}
}

func TestStoreRejects(t *testing.T) {
	t.Parallel()

	vault := newMemoryVault()

	cases := []struct {
		name string
		call func() error
		want errors.Code
	}{
		{
			name: "empty reference",
			call: func() error { return NewStore(vault, testKey()).Put(t.Context(), "", "x") },
			want: errors.Invalid,
		},
		{
			name: "short key",
			call: func() error { return NewStore(vault, []byte("short")).Put(t.Context(), "a", "x") },
			want: errors.Invalid,
		},
		{
			name: "missing secret",
			call: func() error { return NewStore(vault, testKey()).Delete(t.Context(), "absent") },
			want: errors.NotFound,
		},
		{
			name: "missing secret on read",
			call: func() error {
				_, err := NewStore(vault, testKey()).Get(t.Context(), "absent")
				return err
			},
			want: errors.NotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if err := tc.call(); errors.CodeOf(err) != tc.want {
				t.Errorf("code = %q, want %q", errors.CodeOf(err), tc.want)
			}
		})
	}
}

func TestStoreRejectsAnotherKey(t *testing.T) {
	t.Parallel()

	vault := newMemoryVault()
	if err := NewStore(vault, testKey()).Put(t.Context(), "a", "x"); err != nil {
		t.Fatalf("Put: %v", err)
	}

	other := bytes.Repeat([]byte{0x01}, aesgcm.KeyLength)
	if _, err := NewStore(vault, other).Get(t.Context(), "a"); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("code = %q, want %q", errors.CodeOf(err), errors.Invalid)
	}
}
