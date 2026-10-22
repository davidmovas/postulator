package runs_test

import (
	"slices"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/application/runs"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/run"
	kctx "github.com/davidmovas/postulator/internal/kernel/ctx"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

const (
	reviewedFields = `{"pages":[{"pageId":"page-1","path":"/espresso/","fields":[{"key":"seo_title","before":"","beforeHash":"",` +
		`"after":"Espresso","afterHash":"h"},{"key":"excerpt","before":"","beforeHash":"","after":"Pull a shot.","afterHash":"h"}],` +
		`"blocks":[{"kind":"paragraph","hash":"h","raw":"<p>Ours.</p>","owner":"ours","change":"added"}],"body":"body","wpId":7}],"hash":"plan"}`
	reviewedBodies = `{"pages":[{"pageId":"page-1","path":"/espresso/","fields":[],` +
		`"blocks":[{"kind":"paragraph","hash":"h","raw":"<p>Ours.</p>","owner":"ours","change":"added"}],"wpId":7}],"hash":"plan"}`
)

func (f *fixture) seedReview(t *testing.T, reason run.PauseReason, plan string) run.Item {
	t.Helper()

	record, _ := f.seedRun(t, run.StatusPaused)
	item := run.Item{
		ID: id.New(), RunID: record.ID, SiteID: record.SiteID, TargetID: f.pages[1], Status: run.StatusPaused,
		CurrentStep: string(run.StepPublish), PauseReason: reason, Seq: 5,
		Checkpoint: run.NewCheckpoint(), CreatedAt: sqlitetest.Stamp.Add(time.Second), UpdatedAt: sqlitetest.Stamp,
	}
	if err := f.items.Insert(t.Context(), item); err != nil {
		t.Fatalf("insert the reviewed item: %v", err)
	}
	artifact, err := run.NewArtifact(run.Artifact{
		ID: id.New(), RunID: record.ID, ItemID: item.ID, Step: string(run.StepPublish), Kind: run.ArtifactEditPlan,
		Blob: []byte(plan), CreatedAt: sqlitetest.Stamp,
	})
	if err != nil {
		t.Fatalf("NewArtifact: %v", err)
	}
	if err = f.blobs.ReplaceStep(t.Context(), item.ID, string(run.StepPublish), []run.Artifact{artifact}); err != nil {
		t.Fatalf("seed the edit plan: %v", err)
	}
	return item
}

func TestAPersonAcceptsTheFieldsAReviewShows(t *testing.T) {
	t.Parallel()

	fixture := newFixture(t)
	item := fixture.seedReview(t, run.PauseAwaitingReview, reviewedFields)
	asPerson := kctx.WithActor(t.Context(), kctx.ActorUser)

	_, err := fixture.service.RetryStep(asPerson, runs.RetryStepRequest{
		ItemID: item.ID, AcceptFindings: true, Fields: []string{"seo_title", "body"},
	})
	if err != nil {
		t.Fatalf("accept two fields: %v", err)
	}
	if fixture.engine.accepted != item.ID || !slices.Equal(fixture.engine.acceptedFields, []field.Key{field.SEOTitle, field.Body}) {
		t.Fatalf("the engine accepted %q with %v, want the item with the SEO title and the body", fixture.engine.accepted, fixture.engine.acceptedFields)
	}

	if _, err = fixture.service.RetryStep(asPerson, runs.RetryStepRequest{ItemID: item.ID, AcceptFindings: true}); err != nil {
		t.Fatalf("accept the whole review: %v", err)
	}
	if fixture.engine.acceptedFields != nil {
		t.Fatalf("a whole acceptance handed the engine %v, want no narrowing", fixture.engine.acceptedFields)
	}
}

func TestAcceptingAFieldTheReviewDoesNotShowIsRefused(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		reason run.PauseReason
		plan   string
		req    func(itemID string) runs.RetryStepRequest
	}{
		{
			name: "a field the review does not show", reason: run.PauseAwaitingReview, plan: reviewedFields,
			req: func(itemID string) runs.RetryStepRequest {
				return runs.RetryStepRequest{ItemID: itemID, AcceptFindings: true, Fields: []string{"seo_description"}}
			},
		},
		{
			name: "a field the deny list names", reason: run.PauseAwaitingReview, plan: reviewedFields,
			req: func(itemID string) runs.RetryStepRequest {
				return runs.RetryStepRequest{ItemID: itemID, AcceptFindings: true, Fields: []string{"price"}}
			},
		},
		{
			name: "a field the catalog does not know", reason: run.PauseAwaitingReview, plan: reviewedFields,
			req: func(itemID string) runs.RetryStepRequest {
				return runs.RetryStepRequest{ItemID: itemID, AcceptFindings: true, Fields: []string{"nickname"}}
			},
		},
		{
			name: "no field at all", reason: run.PauseAwaitingReview, plan: reviewedFields,
			req: func(itemID string) runs.RetryStepRequest {
				return runs.RetryStepRequest{ItemID: itemID, AcceptFindings: true, Fields: []string{}}
			},
		},
		{
			name: "fields on a plain retry", reason: run.PauseAwaitingReview, plan: reviewedFields,
			req: func(itemID string) runs.RetryStepRequest {
				return runs.RetryStepRequest{ItemID: itemID, Fields: []string{"seo_title"}}
			},
		},
		{
			name: "fields on an item no review holds", reason: run.PauseNeedsHuman, plan: reviewedFields,
			req: func(itemID string) runs.RetryStepRequest {
				return runs.RetryStepRequest{ItemID: itemID, AcceptFindings: true, Fields: []string{"seo_title"}}
			},
		},
		{
			name: "fields on a review that changes bodies only", reason: run.PauseAwaitingReview, plan: reviewedBodies,
			req: func(itemID string) runs.RetryStepRequest {
				return runs.RetryStepRequest{ItemID: itemID, AcceptFindings: true, Fields: []string{"body"}}
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			fixture := newFixture(t)
			item := fixture.seedReview(t, tc.reason, tc.plan)
			_, err := fixture.service.RetryStep(kctx.WithActor(t.Context(), kctx.ActorUser), tc.req(item.ID))
			if !errors.IsCode(err, errors.Invalid) {
				t.Fatalf("RetryStep = %v, want it refused as invalid", err)
			}
			if fixture.engine.accepted != "" || fixture.engine.retried != "" {
				t.Fatalf("the engine accepted %q and retried %q, want nothing requeued", fixture.engine.accepted, fixture.engine.retried)
			}
		})
	}
}
