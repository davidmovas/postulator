package sqlite

import (
	"net/url"
	"slices"
	"strings"
	"testing"
)

func TestDSN(t *testing.T) {
	t.Parallel()

	key := make([]byte, keyLength)
	for i := range key {
		key[i] = byte(i)
	}

	cases := []struct {
		name        string
		path        string
		key         []byte
		readOnly    bool
		wantOpaque  string
		wantParams  map[string]string
		wantPragmas []string
		wantAbsent  []string
	}{
		{
			name:       "plain writer",
			path:       `C:\data\postulator.db`,
			key:        nil,
			readOnly:   false,
			wantOpaque: "C:/data/postulator.db",
			wantParams: map[string]string{"_txlock": "immediate"},
			wantPragmas: []string{
				"busy_timeout(5000)",
				"foreign_keys(1)",
				"journal_mode(WAL)",
				"synchronous(NORMAL)",
			},
			wantAbsent: []string{"vfs", "hexkey", "mode"},
		},
		{
			name:       "encrypted writer",
			path:       `C:\data\postulator.db`,
			key:        key,
			readOnly:   false,
			wantOpaque: "C:/data/postulator.db",
			wantParams: map[string]string{
				"_txlock": "immediate",
				"vfs":     "adiantum",
				"hexkey":  "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f",
			},
			wantPragmas: []string{
				"busy_timeout(5000)",
				"foreign_keys(1)",
				"journal_mode(WAL)",
				"synchronous(NORMAL)",
			},
			wantAbsent: []string{"mode"},
		},
		{
			name:        "encrypted reader",
			path:        `C:\Program Files\postulator.db`,
			key:         key,
			readOnly:    true,
			wantOpaque:  "C:/Program%20Files/postulator.db",
			wantParams:  map[string]string{"mode": "ro", "vfs": "adiantum"},
			wantPragmas: []string{"busy_timeout(5000)", "foreign_keys(1)"},
			wantAbsent:  []string{"_txlock"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			raw := dsn(tc.path, tc.key, tc.readOnly)
			if !strings.HasPrefix(raw, "file:") {
				t.Fatalf("dsn = %q, want a file: URI", raw)
			}

			parsed, err := url.Parse(raw)
			if err != nil {
				t.Fatalf("parse %q: %v", raw, err)
			}
			if parsed.Opaque != tc.wantOpaque {
				t.Errorf("opaque = %q, want %q", parsed.Opaque, tc.wantOpaque)
			}

			query := parsed.Query()
			for name, want := range tc.wantParams {
				if query.Get(name) != want {
					t.Errorf("%s = %q, want %q", name, query.Get(name), want)
				}
			}
			for _, name := range tc.wantAbsent {
				if query.Has(name) {
					t.Errorf("%s must be absent, got %q", name, query.Get(name))
				}
			}
			if !slices.Equal(query["_pragma"], tc.wantPragmas) {
				t.Errorf("_pragma = %v, want %v", query["_pragma"], tc.wantPragmas)
			}
		})
	}
}
