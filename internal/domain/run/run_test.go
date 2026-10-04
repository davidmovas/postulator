package run_test

import (
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/template"
	kctx "github.com/davidmovas/postulator/internal/kernel/ctx"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

var stamp = time.Date(2026, time.September, 18, 9, 0, 0, 0, time.UTC)

func validRun() run.Run {
	return run.Run{
		ID:         "run-1",
		SiteID:     "site-1",
		Kind:       run.KindGenerate,
		Targets:    []string{"page-1"},
		Recipe:     []template.StepSpec{{Name: "generate_body", Enabled: true}},
		DeadlineAt: stamp.Add(time.Hour),
		CreatedAt:  stamp,
	}
}

func TestNewRunDefaults(t *testing.T) {
	t.Parallel()

	record, err := run.NewRun(validRun())
	if err != nil {
		t.Fatalf("NewRun: %v", err)
	}
	if record.Status != run.StatusPending {
		t.Errorf("Status = %q, want %q", record.Status, run.StatusPending)
	}
	if record.PublishMode != run.PublishDraft {
		t.Errorf("PublishMode = %q, want %q", record.PublishMode, run.PublishDraft)
	}
	if record.CreatedBy != kctx.ActorUser {
		t.Errorf("CreatedBy = %q, want %q", record.CreatedBy, kctx.ActorUser)
	}
}

func TestNewRunRejects(t *testing.T) {
	t.Parallel()

	empty := ""
	cases := []struct {
		name   string
		mutate func(*run.Run)
	}{
		{name: "no id", mutate: func(r *run.Run) { r.ID = "" }},
		{name: "no site", mutate: func(r *run.Run) { r.SiteID = "" }},
		{name: "unknown kind", mutate: func(r *run.Run) { r.Kind = "dance" }},
		{name: "unknown status", mutate: func(r *run.Run) { r.Status = "dancing" }},
		{name: "unknown publish mode", mutate: func(r *run.Run) { r.PublishMode = "broadcast" }},
		{name: "no targets", mutate: func(r *run.Run) { r.Targets = nil }},
		{name: "no recipe", mutate: func(r *run.Run) { r.Recipe = nil }},
		{name: "negative budget", mutate: func(r *run.Run) { r.Budget = run.Budget{MaxUSD: -1} }},
		{name: "negative token budget", mutate: func(r *run.Run) { r.Budget = run.Budget{MaxTokens: -1} }},
		{name: "unknown pause reason", mutate: func(r *run.Run) { r.PauseReason = "tired" }},
		{name: "empty parent", mutate: func(r *run.Run) { r.ParentRunID = &empty }},
		{name: "no deadline", mutate: func(r *run.Run) { r.DeadlineAt = time.Time{} }},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			record := validRun()
			tc.mutate(&record)
			if _, err := run.NewRun(record); !errors.IsCode(err, errors.Invalid) {
				t.Fatalf("NewRun = %v, want an invalid error", err)
			}
		})
	}
}

func TestKindsAndPublishModesAreClosedSets(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		valid bool
		want  bool
	}{
		{name: "generate", valid: run.KindGenerate.Valid(), want: true},
		{name: "relink", valid: run.KindRelink.Valid(), want: true},
		{name: "audit", valid: run.KindAudit.Valid(), want: true},
		{name: "sync", valid: run.KindSync.Valid(), want: true},
		{name: "import", valid: run.KindImport.Valid(), want: true},
		{name: "repair", valid: run.KindRepair.Valid(), want: true},
		{name: "revert", valid: run.KindRevert.Valid(), want: true},
		{name: "custom", valid: run.KindCustom.Valid(), want: true},
		{name: "an unknown kind", valid: run.Kind("dance").Valid()},
		{name: "draft", valid: run.PublishDraft.Valid(), want: true},
		{name: "publish", valid: run.PublishLive.Valid(), want: true},
		{name: "an unknown publish mode", valid: run.PublishMode("broadcast").Valid()},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if tc.valid != tc.want {
				t.Fatalf("%s validates = %t, want %t", tc.name, tc.valid, tc.want)
			}
		})
	}
}
