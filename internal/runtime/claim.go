package runtime

import (
	"context"
	"time"

	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/application/templates"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type claim struct {
	item      run.Item
	record    run.Run
	page      pagemap.Page
	spec      template.TemplateSpec
	def       run.Definition
	step      run.StepDef
	artifacts map[run.ArtifactKind]run.Artifact
	params    map[string]any
	index     int
	version   int
	inputHash string
	timeout   time.Duration
	expectSeq int64
}

func (e *Engine) claim(parent context.Context, itemID string) (*claim, error) {
	var held *claim

	err := e.transact(parent, func(ctx context.Context, box *outbox) error {
		item, err := e.deps.Items.Get(ctx, itemID)
		if err != nil {
			return err
		}
		record, err := e.deps.Runs.Get(ctx, item.RunID)
		if err != nil {
			return err
		}

		now := e.now()
		if !claimable(record, item, now) {
			return nil
		}
		if now.After(record.DeadlineAt) {
			return e.expire(ctx, box, record, now)
		}

		candidate := &claim{item: item, record: record}
		if fault := e.locate(ctx, candidate); fault != nil {
			return e.abandon(ctx, box, record, item, *fault, now)
		}
		leased, err := e.lease(ctx, candidate, now)
		if err != nil || !leased {
			return err
		}
		if gatherErr := e.gather(ctx, candidate); gatherErr != nil {
			return gatherErr
		}
		if beginErr := e.begin(ctx, box, candidate, now); beginErr != nil {
			return beginErr
		}
		held = candidate
		return nil
	})
	if err != nil {
		return nil, err
	}
	return held, nil
}

func claimable(record run.Run, item run.Item, now time.Time) bool {
	switch {
	case atRest(record.Status), !item.Status.Advanceable():
		return false
	case item.Status == run.StatusWaiting && item.WakeAt != nil && now.Before(*item.WakeAt):
		return false
	default:
		return true
	}
}

func (e *Engine) locate(ctx context.Context, held *claim) *run.Fault {
	scoped := held.record.Kind.PageScoped()

	if scoped {
		resolved, err := e.deps.Specs.ResolveForPage(ctx, templates.ResolveForPageRequest{PageID: held.item.TargetID})
		if err != nil {
			return classified(err)
		}
		held.spec, held.version = resolved.Spec, resolved.Version
	}

	def, err := run.Plan(e.registry, held.record.Kind, held.record.Recipe, e.cfg.RunDeadline)
	if err != nil {
		return classified(err)
	}
	index, step, found := def.StepByName(held.item.CurrentStep)
	if !found {
		return &run.Fault{
			Class: run.FaultFatal, Code: run.CodeUnknownStep, Action: run.ActionFail,
			Message: "step " + held.item.CurrentStep + " is not part of the recipe of this run",
		}
	}
	held.def, held.index, held.step = def, index, step

	if scoped {
		page, err := e.deps.Pages.Get(ctx, held.item.TargetID)
		if err != nil {
			return classified(err)
		}
		held.page = page
	}
	return nil
}

func classified(err error) *run.Fault {
	fault := run.Classify(err)
	return &fault
}

func (e *Engine) lease(ctx context.Context, held *claim, now time.Time) (bool, error) {
	held.timeout = held.step.Timeout
	if held.timeout <= 0 {
		held.timeout = e.cfg.StepTimeout
	}

	leaseUntil := now.Add(held.timeout + e.cfg.LeaseDuration)
	claimed, err := e.deps.Items.Claim(ctx, held.item.ID, held.item.AdvanceSeq, leaseUntil, now)
	if errors.IsCode(err, errors.Conflict) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	held.item = claimed
	held.expectSeq = claimed.AdvanceSeq
	return true, nil
}

func (e *Engine) gather(ctx context.Context, held *claim) error {
	stored, err := e.deps.Artifacts.ByItem(ctx, held.item.ID)
	if err != nil {
		return err
	}

	held.artifacts = artifactsByKind(held.def, held.index, stored)
	held.params = run.ParamsFor(held.record.Recipe, held.step.Name)
	held.inputHash, err = run.InputHash(held.step.Name, held.params, required(held.step, held.artifacts),
		held.version, held.item.Checkpoint)
	return err
}

func (e *Engine) begin(ctx context.Context, box *outbox, held *claim, now time.Time) error {
	runID, itemID := held.record.ID, held.item.ID

	if held.record.Status == run.StatusPending {
		held.record.Status = run.StatusRunning
		held.record.StartedAt = &now
		if err := e.deps.Runs.Update(ctx, held.record); err != nil {
			return err
		}
		box.add(ctx, runID, events.RunStarted, events.RunStartedPayload{RunID: runID})
	}
	if held.item.AdvanceSeq == 1 {
		box.add(ctx, runID, events.ItemStarted, events.ItemStartedPayload{RunID: runID, ItemID: itemID})
	}
	box.add(ctx, runID, events.StepStarted, events.StepStartedPayload{RunID: runID, ItemID: itemID, Step: held.step.Name})
	return nil
}

func required(step run.StepDef, available map[run.ArtifactKind]run.Artifact) []run.Artifact {
	out := make([]run.Artifact, 0, len(step.Requires))
	for _, kind := range step.Requires {
		if artifact, ok := available[kind]; ok {
			out = append(out, artifact)
		}
	}
	return out
}

func artifactsByKind(def run.Definition, upto int, stored []run.Artifact) map[run.ArtifactKind]run.Artifact {
	order := make(map[string]int, len(def.Steps))
	for i := range def.Steps {
		order[def.Steps[i].Name] = i
	}

	byKind := make(map[run.ArtifactKind]run.Artifact, len(stored))
	rank := make(map[run.ArtifactKind]int, len(stored))
	for i := range stored {
		artifact := stored[i]
		position, known := order[artifact.Step]
		if !known || position > upto {
			continue
		}
		if seen, ok := rank[artifact.Kind]; ok && seen >= position {
			continue
		}
		byKind[artifact.Kind] = artifact
		rank[artifact.Kind] = position
	}
	return byKind
}
