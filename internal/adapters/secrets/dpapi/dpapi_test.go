package dpapi_test

import (
	"bytes"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/davidmovas/postulator/internal/adapters/secrets/dpapi"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const distinctiveLength = 8

func TestRoundTrip(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		plaintext []byte
	}{
		{name: "one byte", plaintext: []byte{0x00}},
		{name: "master key", plaintext: bytes.Repeat([]byte{0xab}, 32)},
		{name: "text", plaintext: []byte("correct horse battery staple")},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			protected, err := dpapi.Protect(tc.plaintext)
			if err != nil {
				t.Fatalf("Protect: %v", err)
			}
			if len(tc.plaintext) >= distinctiveLength && bytes.Contains(protected, tc.plaintext) {
				t.Fatal("the protected blob must not contain the plaintext")
			}

			recovered, err := dpapi.Unprotect(protected)
			if err != nil {
				t.Fatalf("Unprotect: %v", err)
			}
			if !bytes.Equal(recovered, tc.plaintext) {
				t.Errorf("Unprotect = %x, want %x", recovered, tc.plaintext)
			}
		})
	}
}

func TestEmptyInput(t *testing.T) {
	t.Parallel()

	if _, err := dpapi.Protect(nil); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("Protect code = %q, want %q", errors.CodeOf(err), errors.Invalid)
	}
	if _, err := dpapi.Unprotect(nil); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("Unprotect code = %q, want %q", errors.CodeOf(err), errors.Invalid)
	}
}

func TestTamperedBlobIsRejected(t *testing.T) {
	t.Parallel()

	protected, err := dpapi.Protect([]byte("secret"))
	if err != nil {
		t.Fatalf("Protect: %v", err)
	}

	protected[len(protected)-1] ^= 0xff
	if _, err = dpapi.Unprotect(protected); err == nil {
		t.Fatal("a tampered blob must not unprotect")
	}
	if errors.CodeOf(err) != errors.Invalid {
		t.Errorf("code = %q, want %q", errors.CodeOf(err), errors.Invalid)
	}
}

func TestEntropyIsRequired(t *testing.T) {
	t.Parallel()

	plaintext := []byte("a secret worth protecting")

	in := windows.DataBlob{Size: uint32(len(plaintext)), Data: &plaintext[0]}
	var out windows.DataBlob
	if err := windows.CryptProtectData(&in, nil, nil, 0, nil, windows.CRYPTPROTECT_UI_FORBIDDEN, &out); err != nil {
		t.Fatalf("protect without entropy: %v", err)
	}

	foreign := make([]byte, out.Size)
	copy(foreign, unsafe.Slice(out.Data, out.Size))
	if _, err := windows.LocalFree(windows.Handle(unsafe.Pointer(out.Data))); err != nil {
		t.Fatalf("free: %v", err)
	}

	recovered, err := dpapi.Unprotect(foreign)
	if err == nil {
		t.Fatalf("a blob protected without our entropy must not unprotect, got %q", recovered)
	}
	if !errors.IsCode(err, errors.Invalid) {
		t.Errorf("code = %q, want %q", errors.CodeOf(err), errors.Invalid)
	}
}
