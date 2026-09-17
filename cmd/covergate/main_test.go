package main

import (
	"os"
	"path/filepath"
	"testing"
)

func writeProfile(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "coverage.out")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write profile: %v", err)
	}
	return path
}

func TestRun(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		profile string
		missing bool
		wantErr bool
	}{
		{
			name:    "gates met",
			profile: profileText(module + "/internal/kernel/id/id.go:1.1,2.2 10 1"),
		},
		{
			name: "total gate missed",
			profile: profileText(
				module+"/internal/kernel/id/id.go:1.1,2.2 1 1",
				module+"/internal/kernel/id/id.go:3.1,4.2 9 0",
			),
			wantErr: true,
		},
		{
			name:    "malformed profile",
			profile: "not a profile\n",
			wantErr: true,
		},
		{
			name:    "missing profile",
			missing: true,
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(t.TempDir(), "absent.out")
			if !tc.missing {
				path = writeProfile(t, tc.profile)
			}

			err := run(path, t.TempDir(), gates{core: 80, total: 70})
			if tc.wantErr && err == nil {
				t.Fatal("run() must report a failure")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("run() error: %v", err)
			}
		})
	}
}
