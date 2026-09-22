package runs_test

import (
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/application/runs"
	"github.com/davidmovas/postulator/internal/domain/run"
	kctx "github.com/davidmovas/postulator/internal/kernel/ctx"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

func (f *fixture) seedRevert(t *testing.T, parent run.Run, status run.Status) run.Run {
	t.Helper()

	record := run.Run{
		ID: id.New(), SiteID: f.siteID, Kind: run.KindRevert, Status: status, Targets: f.pages[:1],
		Recipe: run.RevertRecipe(), PublishMode: run.PublishDraft, ParentRunID: &parent.ID,
		CreatedBy: kctx.ActorUser, DeadlineAt: sqlitetest.Stamp.Add(time.Hour), CreatedAt: sqlitetest.Stamp,
	}
	if err := f.runs.Insert(t.Context(), record); err != nil {
		t.Fatalf("insert the revert run: %v", err)
	}
	return record
}

func TestRevertEnqueuesOneItemPerPageTheRunPublished(t *testing.T) {
	t.Parallel()

	fixture := newFixture(t)
	record, item := fixture.seedRun(t, run.StatusCompleted)
	fixture.seedArtifacts(t, record, item, []run.ArtifactKind{run.ArtifactPublishResult}, nil)

	answered, err := fixture.service.Revert(t.Context(), runs.RevertRequest{RunID: record.ID})
	if err != nil {
		t.Fatalf("Revert: %v", err)
	}
	if answered.RunID == "" {
		t.Fatal("Revert answered no run id")
	}

	queued := fixture.engine.queued
	if queued.Kind != run.KindRevert {
		t.Fatalf("the revert run is a %q", queued.Kind)
	}
	if queued.ParentRunID == nil || *queued.ParentRunID != record.ID {
		t.Fatalf("the revert names %v as its parent, want %s", queued.ParentRunID, record.ID)
	}
	if len(queued.Targets) != 1 || queued.Targets[0] != item.TargetID {
		t.Fatalf("the revert targets %v, want the page the run published", queued.Targets)
	}
	if len(queued.Recipe) != 1 || queued.Recipe[0].Name != run.RevertStep {
		t.Fatalf("the revert recipe is %+v", queued.Recipe)
	}
	if queued.Budget != (run.Budget{}) {
		t.Fatalf("the revert carries a budget of %+v, want none: it spends nothing", queued.Budget)
	}
}

func TestRevertRefusesWhatCannotBePutBack(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		setup func(t *testing.T, f *fixture) string
		code  errors.Code
	}{
		{
			name:  "no run",
			setup: func(*testing.T, *fixture) string { return "  " },
			code:  errors.Invalid,
		},
		{
			name: "a run still in flight",
			setup: func(t *testing.T, f *fixture) string {
				t.Helper()
				record, item := f.seedRun(t, run.StatusRunning)
				f.seedArtifacts(t, record, item, []run.ArtifactKind{run.ArtifactPublishResult}, nil)
				return record.ID
			},
			code: errors.Conflict,
		},
		{
			name: "a run that published nothing",
			setup: func(t *testing.T, f *fixture) string {
				t.Helper()
				record, item := f.seedRun(t, run.StatusCompleted)
				f.seedArtifacts(t, record, item, []run.ArtifactKind{run.ArtifactBodyHTML}, nil)
				return record.ID
			},
			code: errors.Invalid,
		},
		{
			name: "a run whose publish record was purged",
			setup: func(t *testing.T, f *fixture) string {
				t.Helper()
				record, item := f.seedRun(t, run.StatusCompleted)
				f.seedArtifacts(t, record, item, nil, []run.ArtifactKind{run.ArtifactPublishResult})
				return record.ID
			},
			code: errors.Invalid,
		},
		{
			name: "a revert of a revert",
			setup: func(t *testing.T, f *fixture) string {
				t.Helper()
				record, _ := f.seedRun(t, run.StatusCompleted)
				return f.seedRevert(t, record, run.StatusCompleted).ID
			},
			code: errors.Conflict,
		},
		{
			name: "a second revert of the same run",
			setup: func(t *testing.T, f *fixture) string {
				t.Helper()
				record, item := f.seedRun(t, run.StatusCompleted)
				f.seedArtifacts(t, record, item, []run.ArtifactKind{run.ArtifactPublishResult}, nil)
				f.seedRevert(t, record, run.StatusCompleted)
				return record.ID
			},
			code: errors.Conflict,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			fixture := newFixture(t)
			runID := tc.setup(t, fixture)
			_, err := fixture.service.Revert(t.Context(), runs.RevertRequest{RunID: runID})
			if !errors.IsCode(err, tc.code) {
				t.Fatalf("Revert = %v, want %s", err, tc.code)
			}
		})
	}
}

func TestRevertIsOfferedAgainAfterOneThatFailed(t *testing.T) {
	t.Parallel()

	fixture := newFixture(t)
	record, item := fixture.seedRun(t, run.StatusCompleted)
	fixture.seedArtifacts(t, record, item, []run.ArtifactKind{run.ArtifactPublishResult}, nil)
	fixture.seedRevert(t, record, run.StatusFailed)

	if _, err := fixture.service.Revert(t.Context(), runs.RevertRequest{RunID: record.ID}); err != nil {
		t.Fatalf("Revert after a failed revert: %v", err)
	}
}
