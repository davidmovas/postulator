package masterpassword_test

import (
	"bytes"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/secrets/masterpassword"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func TestSealAndOpenRoundTrip(t *testing.T) {
	t.Parallel()

	secret := []byte("the protected master key blob")

	envelope, err := masterpassword.Seal("correct horse battery staple", secret)
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}

	opened, err := masterpassword.Open("correct horse battery staple", envelope)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if !bytes.Equal(opened, secret) {
		t.Fatalf("Open returned %q, want %q", opened, secret)
	}
}

func TestSealHidesThePlaintext(t *testing.T) {
	t.Parallel()

	secret := bytes.Repeat([]byte{0xA7}, 32)

	envelope, err := masterpassword.Seal("hunter2", secret)
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	if bytes.Contains(envelope, secret) {
		t.Fatal("the envelope carries the plaintext")
	}
	if string(envelope[:len(masterpassword.Version)]) != masterpassword.Version {
		t.Fatalf("the envelope starts with %q, want %q", envelope[:len(masterpassword.Version)], masterpassword.Version)
	}
}

func TestSealSaltsEveryEnvelope(t *testing.T) {
	t.Parallel()

	first, err := masterpassword.Seal("hunter2", []byte("blob"))
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	second, err := masterpassword.Seal("hunter2", []byte("blob"))
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	if bytes.Equal(first, second) {
		t.Fatal("two envelopes of the same plaintext are identical")
	}
}

func TestOpenRefuses(t *testing.T) {
	t.Parallel()

	envelope, err := masterpassword.Seal("hunter2", []byte("the protected master key blob"))
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}

	cases := []struct {
		name     string
		password string
		envelope []byte
		want     errors.Code
	}{
		{name: "wrong password", password: "hunter3", envelope: envelope, want: errors.Locked},
		{name: "empty password", password: "", envelope: envelope, want: errors.Invalid},
		{name: "truncated", password: "hunter2", envelope: envelope[:len(envelope)-1], want: errors.Locked},
		{name: "too short", password: "hunter2", envelope: []byte("v1:"), want: errors.Invalid},
		{name: "foreign prefix", password: "hunter2", envelope: bytes.Repeat([]byte{7}, 80), want: errors.Invalid},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if _, openErr := masterpassword.Open(tc.password, tc.envelope); errors.CodeOf(openErr) != tc.want {
				t.Fatalf("Open = %v, want %s", openErr, tc.want)
			}
		})
	}
}

func TestSealRefusesAnEmptyPassword(t *testing.T) {
	t.Parallel()

	if _, err := masterpassword.Seal("", []byte("blob")); errors.CodeOf(err) != errors.Invalid {
		t.Fatalf("Seal = %v, want %s", err, errors.Invalid)
	}
	if _, err := masterpassword.Seal("hunter2", nil); errors.CodeOf(err) != errors.Invalid {
		t.Fatalf("Seal = %v, want %s", err, errors.Invalid)
	}
}
