package runtime

import (
	"slices"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/run"
)

func TestAStepHashesWhatItReadsOnlyWhenItIsThere(t *testing.T) {
	t.Parallel()

	writer := run.StepDef{
		Name: "generate_body", Requires: []run.ArtifactKind{run.ArtifactLinkContext}, Reads: []run.ArtifactKind{run.ArtifactResearch},
	}
	held := func(kinds ...run.ArtifactKind) map[run.ArtifactKind]run.Artifact {
		out := map[run.ArtifactKind]run.Artifact{}
		for _, kind := range kinds {
			out[kind] = run.Artifact{Kind: kind, Hash: string(kind) + "-hash"}
		}
		return out
	}
	cases := []struct {
		name      string
		available map[run.ArtifactKind]run.Artifact
		want      []run.ArtifactKind
	}{
		{name: "a researched page", available: held(run.ArtifactLinkContext, run.ArtifactResearch, run.ArtifactDraft),
			want: []run.ArtifactKind{run.ArtifactLinkContext, run.ArtifactResearch}},
		{name: "a page nobody researched", available: held(run.ArtifactLinkContext),
			want: []run.ArtifactKind{run.ArtifactLinkContext}},
		{name: "a research purged since", available: map[run.ArtifactKind]run.Artifact{
			run.ArtifactLinkContext: {Kind: run.ArtifactLinkContext},
			run.ArtifactResearch:    {Kind: run.ArtifactResearch, Purged: true},
		}, want: []run.ArtifactKind{run.ArtifactLinkContext}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := make([]run.ArtifactKind, 0)
			for _, artifact := range required(writer, nil, tc.available) {
				got = append(got, artifact.Kind)
			}
			if !slices.Equal(got, tc.want) {
				t.Errorf("required = %v, want %v", got, tc.want)
			}
		})
	}
}
