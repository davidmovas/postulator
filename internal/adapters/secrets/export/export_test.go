package export_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/secrets/export"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const password = "correct horse battery staple"

type database struct {
	body    []byte
	err     error
	skip    bool
	replace bool
}

func (d database) Snapshot(_ context.Context, path string) error {
	switch {
	case d.err != nil:
		return d.err
	case d.skip:
		return nil
	case d.replace:
		return os.Mkdir(path, 0o700)
	}
	return os.WriteFile(path, d.body, 0o600)
}

func body(size int) []byte {
	out := make([]byte, size)
	for i := range out {
		out[i] = byte(i % 251)
	}
	return out
}

func archive(t *testing.T, content []byte) (writer *export.Archive, path string) {
	t.Helper()

	backup := filepath.Join(t.TempDir(), "postulator.pstx")
	written := export.New(database{body: content}, clock.NewFake(clock.System{}.Now()))
	if err := written.Write(t.Context(), backup, password); err != nil {
		t.Fatalf("Write: %v", err)
	}
	return written, backup
}

func TestTheBackupRoundTrips(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		size int
	}{
		{name: "empty", size: 0},
		{name: "one short frame", size: 1024},
		{name: "exactly one frame", size: export.FrameSize},
		{name: "several frames", size: 3*export.FrameSize + 17},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			content := body(tc.size)
			writer, backup := archive(t, content)

			raw, err := os.ReadFile(backup)
			if err != nil {
				t.Fatalf("read the backup: %v", err)
			}
			if string(raw[:len(export.Header)]) != export.Header {
				t.Fatalf("the backup starts with %q, want %q", raw[:len(export.Header)], export.Header)
			}
			if tc.size > 0 && bytes.Contains(raw, content) {
				t.Fatal("the backup carries the database in the clear")
			}

			dir, err := writer.Read(t.Context(), backup, password)
			if err != nil {
				t.Fatalf("Read: %v", err)
			}
			t.Cleanup(func() {
				if removeErr := os.RemoveAll(dir); removeErr != nil {
					t.Errorf("remove the restore directory: %v", removeErr)
				}
			})

			restored, err := os.ReadFile(filepath.Join(dir, export.DatabaseName))
			if err != nil {
				t.Fatalf("read the restored database: %v", err)
			}
			if !bytes.Equal(restored, content) {
				t.Fatalf("the restored database is %d bytes, want %d", len(restored), len(content))
			}

			manifest := readManifest(t, dir)
			if manifest.Version != export.Version || manifest.Database != export.DatabaseName {
				t.Fatalf("manifest = %+v", manifest)
			}
			if manifest.Bytes != int64(len(content)) {
				t.Fatalf("manifest bytes = %d, want %d", manifest.Bytes, len(content))
			}
		})
	}
}

func readManifest(t *testing.T, dir string) export.Manifest {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join(dir, export.ManifestName))
	if err != nil {
		t.Fatalf("read the manifest: %v", err)
	}

	var manifest export.Manifest
	if err = json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("decode the manifest: %v", err)
	}
	return manifest
}

func TestReadRefuses(t *testing.T) {
	t.Parallel()

	writer, backup := archive(t, body(3*export.FrameSize))
	raw, err := os.ReadFile(backup)
	if err != nil {
		t.Fatalf("read the backup: %v", err)
	}

	cases := []struct {
		name     string
		password string
		content  []byte
		want     errors.Code
	}{
		{name: "wrong password", password: "hunter2", content: raw, want: errors.Invalid},
		{name: "empty password", password: "", content: raw, want: errors.Invalid},
		{name: "truncated tail", password: password, content: raw[:len(raw)-1], want: errors.Invalid},
		{name: "truncated frames", password: password, content: raw[:len(raw)/2], want: errors.Invalid},
		{name: "truncated header", password: password, content: raw[:4], want: errors.Invalid},
		{name: "foreign file", password: password, content: bytes.Repeat([]byte{7}, 512), want: errors.Invalid},
		{name: "trailing bytes", password: password, content: append(bytes.Clone(raw), 0), want: errors.Invalid},
		{name: "flipped frame", password: password, content: flip(raw, len(raw)-8), want: errors.Invalid},
		{name: "flipped final flag", password: password, content: flip(raw, len(export.Header)+20), want: errors.Invalid},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(t.TempDir(), "broken.pstx")
			if writeErr := os.WriteFile(path, tc.content, 0o600); writeErr != nil {
				t.Fatalf("write the broken backup: %v", writeErr)
			}

			dir, readErr := writer.Read(t.Context(), path, tc.password)
			if !errors.IsCode(readErr, tc.want) {
				t.Fatalf("Read = %v, want %s", readErr, tc.want)
			}
			if dir != "" {
				t.Fatalf("Read returned the directory %q beside its error", dir)
			}
		})
	}
}

func flip(raw []byte, at int) []byte {
	out := bytes.Clone(raw)
	out[at] ^= 0xFF
	return out
}

func TestWriteRefuses(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		source   database
		path     func(t *testing.T) string
		password string
		want     errors.Code
	}{
		{
			name:     "no path",
			path:     func(*testing.T) string { return "" },
			password: password,
			want:     errors.Invalid,
		},
		{
			name:     "no password",
			path:     func(t *testing.T) string { t.Helper(); return filepath.Join(t.TempDir(), "b.pstx") },
			password: "",
			want:     errors.Invalid,
		},
		{
			name:     "the snapshot fails",
			source:   database{err: errors.New(errors.External, "the database is busy")},
			path:     func(t *testing.T) string { t.Helper(); return filepath.Join(t.TempDir(), "b.pstx") },
			password: password,
			want:     errors.External,
		},
		{
			name:     "the snapshot writes nothing",
			source:   database{skip: true},
			path:     func(t *testing.T) string { t.Helper(); return filepath.Join(t.TempDir(), "b.pstx") },
			password: password,
			want:     errors.Invalid,
		},
		{
			name:     "the snapshot is not a file",
			source:   database{replace: true},
			path:     func(t *testing.T) string { t.Helper(); return filepath.Join(t.TempDir(), "b.pstx") },
			password: password,
			want:     errors.Internal,
		},
		{
			name: "the backup directory cannot be created",
			path: func(t *testing.T) string {
				t.Helper()

				blocker := filepath.Join(t.TempDir(), "blocker")
				if err := os.WriteFile(blocker, []byte("file"), 0o600); err != nil {
					t.Fatalf("write the blocking file: %v", err)
				}
				return filepath.Join(blocker, "b.pstx")
			},
			password: password,
			want:     errors.Internal,
		},
		{
			name:     "the path is a directory",
			path:     func(t *testing.T) string { t.Helper(); return t.TempDir() },
			password: password,
			want:     errors.Internal,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			writer := export.New(tc.source, clock.NewFake(clock.System{}.Now()))
			if err := writer.Write(t.Context(), tc.path(t), tc.password); !errors.IsCode(err, tc.want) {
				t.Fatalf("Write = %v, want %s", err, tc.want)
			}
		})
	}
}

func TestReadReportsAMissingFile(t *testing.T) {
	t.Parallel()

	writer := export.New(database{}, clock.NewFake(clock.System{}.Now()))
	if _, err := writer.Read(t.Context(), filepath.Join(t.TempDir(), "absent.pstx"), password); !errors.IsCode(err, errors.NotFound) {
		t.Fatalf("Read = %v, want %s", err, errors.NotFound)
	}
	if _, err := writer.Read(t.Context(), "", password); !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("Read with no path = %v, want %s", err, errors.Invalid)
	}
}
