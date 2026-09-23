package localfile_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/images"
	"github.com/davidmovas/postulator/internal/adapters/images/localfile"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func folder(t *testing.T, names ...string) string {
	t.Helper()

	dir := t.TempDir()
	for _, name := range names {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(name), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	return dir
}

func TestPickMatchesTheFilename(t *testing.T) {
	t.Parallel()

	dir := folder(t, "espresso-machine.png", "espresso-cup.jpg", "kettle.png", "notes.txt")

	cases := []struct {
		name  string
		query images.Query
		want  []string
	}{
		{
			name:  "one keyword matches two files",
			query: images.Query{Term: "espresso brewing"},
			want:  []string{"espresso-cup.jpg", "espresso-machine.png"},
		},
		{
			name:  "the limit is obeyed",
			query: images.Query{Term: "espresso", Limit: 1},
			want:  []string{"espresso-cup.jpg"},
		},
		{
			name:  "short words are ignored",
			query: images.Query{Term: "a of to"},
			want:  []string{"espresso-cup.jpg", "espresso-machine.png", "kettle.png"},
		},
		{
			name:  "nothing matches",
			query: images.Query{Term: "matcha"},
			want:  []string{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			picked, err := localfile.New(dir).Pick(t.Context(), tc.query)
			if err != nil {
				t.Fatalf("Pick: %v", err)
			}
			if len(picked) != len(tc.want) {
				t.Fatalf("Pick returned %d images, want %d: %+v", len(picked), len(tc.want), picked)
			}
			for i, name := range tc.want {
				if picked[i].Filename != name {
					t.Errorf("image %d = %q, want %q", i, picked[i].Filename, name)
				}
				if len(picked[i].Bytes) == 0 || picked[i].ContentType == "" {
					t.Errorf("image %d = %+v", i, picked[i])
				}
			}
		})
	}
}

func TestPickReportsWhatItCannotDo(t *testing.T) {
	t.Parallel()

	cancelled, cancel := context.WithCancel(t.Context())
	cancel()

	cases := []struct {
		name   string
		source *localfile.Source
		ctx    context.Context
		want   errors.Code
	}{
		{name: "no folder is configured", source: localfile.New("  "), want: errors.Invalid},
		{name: "the folder is gone", source: localfile.New(filepath.Join(t.TempDir(), "absent")), want: errors.External},
		{name: "the caller left", source: localfile.New(t.TempDir()), ctx: cancelled, want: errors.Cancelled},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := tc.ctx
			if ctx == nil {
				ctx = t.Context()
			}
			if _, err := tc.source.Pick(ctx, images.Query{Term: "espresso"}); !errors.IsCode(err, tc.want) {
				t.Fatalf("code = %q, want %q (err %v)", errors.CodeOf(err), tc.want, err)
			}
		})
	}
}
