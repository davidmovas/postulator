package aesgcm_test

import (
	"bytes"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/secrets/aesgcm"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func key(fill byte) []byte {
	return bytes.Repeat([]byte{fill}, aesgcm.KeyLength)
}

func TestSealOpen(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		plaintext []byte
	}{
		{name: "empty", plaintext: []byte{}},
		{name: "application password", plaintext: []byte("abcd EFGH 1234 ijkl MNOP 5678")},
		{name: "binary", plaintext: bytes.Repeat([]byte{0x00, 0xff}, 64)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			sealed, err := aesgcm.Seal(key(0x11), tc.plaintext)
			if err != nil {
				t.Fatalf("Seal: %v", err)
			}
			if !bytes.HasPrefix(sealed, []byte(aesgcm.Version)) {
				t.Fatalf("envelope = %x, want the %q prefix", sealed, aesgcm.Version)
			}
			if len(tc.plaintext) > 0 && bytes.Contains(sealed, tc.plaintext) {
				t.Fatal("the envelope must not contain the plaintext")
			}

			recovered, err := aesgcm.Open(key(0x11), sealed)
			if err != nil {
				t.Fatalf("Open: %v", err)
			}
			if !bytes.Equal(recovered, tc.plaintext) {
				t.Errorf("Open = %x, want %x", recovered, tc.plaintext)
			}
		})
	}
}

func TestNonceIsFresh(t *testing.T) {
	t.Parallel()

	first, err := aesgcm.Seal(key(0x22), []byte("same"))
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	second, err := aesgcm.Seal(key(0x22), []byte("same"))
	if err != nil {
		t.Fatalf("Seal again: %v", err)
	}
	if bytes.Equal(first, second) {
		t.Error("two seals of the same plaintext must differ")
	}
}

func TestOpenRejects(t *testing.T) {
	t.Parallel()

	sealed, err := aesgcm.Seal(key(0x33), []byte("secret"))
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}

	tampered := bytes.Clone(sealed)
	tampered[len(tampered)-1] ^= 0xff

	badVersion := bytes.Clone(sealed)
	badVersion[1] = '9'

	cases := []struct {
		name     string
		key      []byte
		envelope []byte
		want     errors.Code
	}{
		{name: "short key", key: []byte("short"), envelope: sealed, want: errors.Invalid},
		{name: "wrong key", key: key(0x44), envelope: sealed, want: errors.Invalid},
		{name: "tampered", key: key(0x33), envelope: tampered, want: errors.Invalid},
		{name: "wrong version", key: key(0x33), envelope: badVersion, want: errors.Invalid},
		{name: "truncated", key: key(0x33), envelope: sealed[:4], want: errors.Invalid},
		{name: "empty", key: key(0x33), envelope: nil, want: errors.Invalid},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if _, openErr := aesgcm.Open(tc.key, tc.envelope); errors.CodeOf(openErr) != tc.want {
				t.Errorf("code = %q, want %q", errors.CodeOf(openErr), tc.want)
			}
		})
	}
}

func TestSealRejectsAShortKey(t *testing.T) {
	t.Parallel()

	if _, err := aesgcm.Seal([]byte("short"), []byte("secret")); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("code = %q, want %q", errors.CodeOf(err), errors.Invalid)
	}
}
