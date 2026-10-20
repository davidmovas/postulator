package steps

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const ReasonAwaitingReview = "this is a draft run, so nothing is written to a page that is on the site until a person " +
	"reviews the changes and accepts them; accept them to write them as they are shown"

type PageEdit struct {
	PageID string         `json:"pageId"`
	Path   string         `json:"path"`
	Fields []field.Change `json:"fields"`
	Blocks []block.Change `json:"blocks"`
	WPID   int64          `json:"wpId"`
}

type EditPlan struct {
	Pages []PageEdit `json:"pages"`
	Hash  string     `json:"hash"`
}

func reviewing(sc *run.StepContext) bool {
	return sc.Run.PublishMode == run.PublishDraft && sc.Run.Kind != run.KindRevert
}

func hashOf(value any, what string) (string, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", errors.Wrap(err, errors.Internal, "encode the "+what+" to hash it")
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:]), nil
}

type gate struct {
	held  *run.Result
	check run.Checkpoint
}

func reviewGate(sc *run.StepContext, key string, plan EditPlan, message string) (gate, error) {
	check := run.NewCheckpoint()
	accepted := sc.Accepted()
	if accepted {
		if err := run.Set(check, run.CheckpointAccept, ""); err != nil {
			return gate{}, err
		}
	}
	if !reviewing(sc) || len(plan.Pages) == 0 {
		return gate{check: check}, nil
	}
	reviewed, _, err := run.Get[string](sc.Check, key)
	if err != nil {
		return gate{}, err
	}
	if accepted && reviewed == plan.Hash {
		return gate{check: check}, nil
	}
	if setErr := run.Set(check, key, plan.Hash); setErr != nil {
		return gate{}, setErr
	}
	blob, err := encode(plan, "edit plan")
	if err != nil {
		return gate{}, err
	}
	held := run.Result{
		Next: run.TransitionPause, Reason: run.PauseAwaitingReview, Message: message + ": " + ReasonAwaitingReview,
		Checkpoint: check, Artifacts: []run.Artifact{{Kind: run.ArtifactEditPlan, Blob: blob}},
	}
	return gate{held: &held, check: check}, nil
}

func sealed(pages []PageEdit) (EditPlan, error) {
	hash, err := hashOf(pages, "edit plan")
	if err != nil {
		return EditPlan{}, err
	}
	return EditPlan{Pages: pages, Hash: hash}, nil
}

func (d Deps) ourBlocks(ctx context.Context, pageID string) ([]string, error) {
	if d.Bodies == nil {
		return nil, nil
	}
	held, err := d.Bodies.ListWritten(ctx, pageID)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(held))
	for i := range held {
		out = append(out, held[i].BlockHash)
	}
	return out, nil
}

const (
	checkpointRelinkPageReviewed      = "relink_page.reviewed"
	checkpointRelinkNeighborsReviewed = "relink_neighbors.reviewed"
)

type relinkedBody struct {
	page   pagemap.Page
	before string
	after  string
}

func (d Deps) reviewRelinks(ctx context.Context, sc *run.StepContext, key, message string, bodies []relinkedBody) (gate, error) {
	pages := make([]PageEdit, 0, len(bodies))
	for i := range bodies {
		written, err := d.ourBlocks(ctx, bodies[i].page.ID)
		if err != nil {
			return gate{}, err
		}
		wpID := int64(0)
		if bodies[i].page.WPID != nil {
			wpID = *bodies[i].page.WPID
		}
		pages = append(pages, PageEdit{
			PageID: bodies[i].page.ID, Path: bodies[i].page.Path, WPID: wpID, Fields: []field.Change{},
			Blocks: block.Diff(bodies[i].before, bodies[i].after, written),
		})
	}
	plan, err := sealed(pages)
	if err != nil {
		return gate{}, err
	}
	return reviewGate(sc, key, plan, message)
}

func withChecks(result run.Result, checks ...run.Checkpoint) run.Result {
	merged := result.Checkpoint
	if merged == nil {
		merged = run.NewCheckpoint()
	}
	for _, check := range checks {
		merged = merged.MergedWith(check)
	}
	result.Checkpoint = merged
	return result
}
