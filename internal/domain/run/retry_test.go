package run_test

import (
	"slices"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/run"
)

func TestOnlyTheThreeHeavyArtifactsArePurgeable(t *testing.T) {
	t.Parallel()

	cases := []struct {
		kind run.ArtifactKind
		want bool
	}{
		{kind: run.ArtifactLinkContext, want: false},
		{kind: run.ArtifactDraft, want: true},
		{kind: run.ArtifactBodyHTML, want: true},
		{kind: run.ArtifactMeta, want: false},
		{kind: run.ArtifactImages, want: true},
		{kind: run.ArtifactValidationReport, want: false},
		{kind: run.ArtifactJudgeReport, want: false},
		{kind: run.ArtifactPublishResult, want: false},
		{kind: run.ArtifactRelinkResult, want: false},
		{kind: run.ArtifactSyncResult, want: false},
		{kind: run.ArtifactFinalReport, want: false},
		{kind: run.ArtifactKind("nonsense"), want: false},
	}

	for _, tc := range cases {
		t.Run(string(tc.kind), func(t *testing.T) {
			t.Parallel()

			if got := tc.kind.Purgeable(); got != tc.want {
				t.Fatalf("%s.Purgeable() = %v, want %v", tc.kind, got, tc.want)
			}
		})
	}

	purgeable := run.PurgeableArtifactKinds()
	want := []run.ArtifactKind{run.ArtifactDraft, run.ArtifactBodyHTML, run.ArtifactImages}
	if !slices.Equal(purgeable, want) {
		t.Fatalf("PurgeableArtifactKinds() = %v, want %v", purgeable, want)
	}

	purgeable[0] = run.ArtifactMeta
	if run.PurgeableArtifactKinds()[0] != run.ArtifactDraft {
		t.Fatal("PurgeableArtifactKinds hands out the declaration itself")
	}
}

func TestPurgedKindsReportsWhatIsGone(t *testing.T) {
	t.Parallel()

	artifacts := []run.Artifact{
		{Kind: run.ArtifactLinkContext},
		{Kind: run.ArtifactDraft, Purged: true},
		{Kind: run.ArtifactBodyHTML, Purged: true},
		{Kind: run.ArtifactPublishResult},
	}

	got := run.PurgedKinds(artifacts)
	want := []run.ArtifactKind{run.ArtifactDraft, run.ArtifactBodyHTML}
	if !slices.Equal(got, want) {
		t.Fatalf("PurgedKinds = %v, want %v", got, want)
	}
	if len(run.PurgedKinds(nil)) != 0 {
		t.Fatal("PurgedKinds of nothing must be empty")
	}
}

func TestExpiredInputsNamesTheRequiredKindsThatWerePurged(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		requires []run.ArtifactKind
		purged   []run.ArtifactKind
		expired  []run.ArtifactKind
		reason   run.RetryBlockedReason
	}{
		{
			name:     "nothing required",
			requires: nil,
			purged:   []run.ArtifactKind{run.ArtifactBodyHTML},
			expired:  []run.ArtifactKind{},
			reason:   "",
		},
		{
			name:     "nothing purged",
			requires: []run.ArtifactKind{run.ArtifactBodyHTML, run.ArtifactDraft},
			purged:   nil,
			expired:  []run.ArtifactKind{},
			reason:   "",
		},
		{
			name:     "a purged kind the step does not consume",
			requires: []run.ArtifactKind{run.ArtifactPublishResult},
			purged:   []run.ArtifactKind{run.ArtifactBodyHTML},
			expired:  []run.ArtifactKind{},
			reason:   "",
		},
		{
			name:     "one required kind purged",
			requires: []run.ArtifactKind{run.ArtifactLinkContext, run.ArtifactBodyHTML},
			purged:   []run.ArtifactKind{run.ArtifactBodyHTML, run.ArtifactImages},
			expired:  []run.ArtifactKind{run.ArtifactBodyHTML},
			reason:   run.RetryBlockedInputsExpired,
		},
		{
			name:     "every required kind purged, in the order the step requires them",
			requires: []run.ArtifactKind{run.ArtifactDraft, run.ArtifactBodyHTML},
			purged:   []run.ArtifactKind{run.ArtifactBodyHTML, run.ArtifactDraft},
			expired:  []run.ArtifactKind{run.ArtifactDraft, run.ArtifactBodyHTML},
			reason:   run.RetryBlockedInputsExpired,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := run.ExpiredInputs(tc.requires, tc.purged); !slices.Equal(got, tc.expired) {
				t.Fatalf("ExpiredInputs = %v, want %v", got, tc.expired)
			}
			if got := run.RetryBlocked(tc.requires, tc.purged); got != tc.reason {
				t.Fatalf("RetryBlocked = %q, want %q", got, tc.reason)
			}
		})
	}
}

func TestTheStepNamesAreDeclaredInPipelineOrder(t *testing.T) {
	t.Parallel()

	names := run.StepNames()
	want := []run.StepName{
		run.StepResolveContext, run.StepGenerateBody, run.StepGenerateMeta, run.StepInsertLinks,
		run.StepRepairLinks, run.StepGenerateImages, run.StepValidate, run.StepJudge,
		run.StepRepairHierarchy, run.StepPublish, run.StepRelinkNeighbors, run.StepSyncBack,
		run.StepReport, run.StepSyncSite,
	}
	if !slices.Equal(names, want) {
		t.Fatalf("StepNames() = %v, want %v", names, want)
	}

	seen := make(map[run.StepName]struct{}, len(names))
	for _, name := range names {
		if _, duplicate := seen[name]; duplicate {
			t.Fatalf("%s is declared twice", name)
		}
		seen[name] = struct{}{}
	}

	names[0] = run.StepReport
	if run.StepNames()[0] != run.StepResolveContext {
		t.Fatal("StepNames hands out the declaration itself")
	}
}
