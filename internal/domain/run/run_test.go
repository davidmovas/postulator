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

func TestNewItem(t *testing.T) {
	t.Parallel()

	item, err := run.NewItem(run.Item{ID: "item-1", RunID: "run-1", SiteID: "site-1", TargetID: "page-1", CurrentStep: "resolve_context"})
	if err != nil {
		t.Fatalf("NewItem: %v", err)
	}
	if item.Status != run.StatusPending || item.Checkpoint == nil {
		t.Fatalf("NewItem = %+v", item)
	}

	cases := []struct {
		name string
		item run.Item
	}{
		{name: "no id", item: run.Item{RunID: "r", SiteID: "s1", TargetID: "p", CurrentStep: "s"}},
		{name: "no run", item: run.Item{ID: "i", SiteID: "s1", TargetID: "p", CurrentStep: "s"}},
		{name: "no site", item: run.Item{ID: "i", RunID: "r", TargetID: "p", CurrentStep: "s"}},
		{name: "no target", item: run.Item{ID: "i", RunID: "r", SiteID: "s1", CurrentStep: "s"}},
		{name: "no step", item: run.Item{ID: "i", RunID: "r", SiteID: "s1", TargetID: "p"}},
		{name: "unknown status", item: run.Item{ID: "i", RunID: "r", SiteID: "s1", TargetID: "p", CurrentStep: "s", Status: "hmm"}},
		{name: "negative attempts", item: run.Item{ID: "i", RunID: "r", SiteID: "s1", TargetID: "p", CurrentStep: "s", Attempts: -1}},
		{name: "unknown pause", item: run.Item{ID: "i", RunID: "r", SiteID: "s1", TargetID: "p", CurrentStep: "s", PauseReason: "hmm"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if _, err := run.NewItem(tc.item); !errors.IsCode(err, errors.Invalid) {
				t.Fatalf("NewItem = %v, want an invalid error", err)
			}
		})
	}
}

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

func TestNewEvent(t *testing.T) {
	t.Parallel()

	event, err := run.NewEvent(run.Event{RunID: "r1", Seq: 1, Type: "run.started", At: stamp})
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}
	if string(event.Payload) != "{}" {
		t.Errorf("Payload = %s, want an empty object", event.Payload)
	}

	cases := []struct {
		name  string
		event run.Event
	}{
		{name: "no run", event: run.Event{Seq: 1, Type: "run.started", At: stamp}},
		{name: "no sequence", event: run.Event{RunID: "r", Type: "run.started", At: stamp}},
		{name: "no type", event: run.Event{RunID: "r", Seq: 1, At: stamp}},
		{name: "no timestamp", event: run.Event{RunID: "r", Seq: 1, Type: "run.started"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if _, err := run.NewEvent(tc.event); !errors.IsCode(err, errors.Invalid) {
				t.Fatalf("NewEvent = %v, want an invalid error", err)
			}
		})
	}
}

func TestExecStatusIsAClosedSet(t *testing.T) {
	t.Parallel()

	for _, status := range []run.ExecStatus{run.ExecStarted, run.ExecDone, run.ExecFailed} {
		if !status.Valid() {
			t.Errorf("ExecStatus(%q).Valid() = false", status)
		}
	}
	if run.ExecStatus("pending").Valid() {
		t.Error("an unknown exec status must not validate")
	}
	if !run.KindRelink.Valid() || !run.KindAudit.Valid() || !run.KindSync.Valid() || !run.KindImport.Valid() || !run.KindCustom.Valid() {
		t.Error("every declared run kind must validate")
	}
	if !run.PublishLive.Valid() {
		t.Error("publish must validate")
	}
}
