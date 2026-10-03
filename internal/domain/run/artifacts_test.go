package run_test

import (
	"testing"

	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func TestNewArtifactHashesItsBlob(t *testing.T) {
	t.Parallel()

	artifact, err := run.NewArtifact(run.Artifact{
		ID: "a1", RunID: "r1", ItemID: "i1", Step: "generate_body", Kind: run.ArtifactBodyHTML,
		Blob: []byte("<p>hello</p>"),
	})
	if err != nil {
		t.Fatalf("NewArtifact: %v", err)
	}
	if artifact.Size != len("<p>hello</p>") {
		t.Errorf("Size = %d", artifact.Size)
	}
	if artifact.Hash != run.HashBlob([]byte("<p>hello</p>")) {
		t.Errorf("Hash = %q", artifact.Hash)
	}

	cases := []struct {
		name     string
		artifact run.Artifact
	}{
		{name: "no id", artifact: run.Artifact{RunID: "r", ItemID: "i", Step: "s", Kind: run.ArtifactDraft}},
		{name: "no run", artifact: run.Artifact{ID: "a", ItemID: "i", Step: "s", Kind: run.ArtifactDraft}},
		{name: "no item", artifact: run.Artifact{ID: "a", RunID: "r", Step: "s", Kind: run.ArtifactDraft}},
		{name: "no step", artifact: run.Artifact{ID: "a", RunID: "r", ItemID: "i", Kind: run.ArtifactDraft}},
		{name: "unknown kind", artifact: run.Artifact{ID: "a", RunID: "r", ItemID: "i", Step: "s", Kind: "poem"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if _, err := run.NewArtifact(tc.artifact); !errors.IsCode(err, errors.Invalid) {
				t.Fatalf("NewArtifact = %v, want an invalid error", err)
			}
		})
	}
}

func TestArtifactKindsAreClosed(t *testing.T) {
	t.Parallel()

	for _, kind := range run.ArtifactKinds() {
		if !kind.Valid() {
			t.Errorf("declared kind %q does not validate", kind)
		}
	}
	if run.ArtifactKind("sonnet").Valid() {
		t.Error("an unknown artifact kind must not validate")
	}
}
