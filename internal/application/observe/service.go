package observe

import (
	"context"
	"strings"
	"time"

	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const maxSteps = 100000

func invalid(message, field string) *errors.Error {
	return errors.New(errors.Invalid, message).WithDetail("field", field)
}

func (s *Service) Begin(ctx context.Context, siteID string, readers Readers, opts Options) (State, error) {
	siteID = strings.TrimSpace(siteID)
	if siteID == "" {
		return State{}, invalid("an observation needs a site", "siteId")
	}
	if readers.Core == nil {
		return State{}, invalid("an observation needs a core reader", "readers.core")
	}
	owner, err := s.deps.Sites.Get(ctx, siteID)
	if err != nil {
		return State{}, err
	}
	settings := owner.Settings.Normalized()

	source := SourceCore
	if readers.plugin(capabilityBulk) {
		source = SourcePlugin
	}
	observation := settings.Observation
	full := opts.Full || source == SourceCore || observation.LastStartedAt == nil ||
		observation.SinceFullWalk+1 >= settings.Observe.FullWalkEvery

	state := State{
		SiteID: siteID, StartedAt: s.now(), Source: source, Stage: StageInventory, Full: full, Rescan: opts.RenderRescan,
		Types: []string{}, Findings: []content.Finding{},
	}
	if !full {
		since := observation.LastStartedAt.Add(-time.Second)
		state.Since = &since
	}
	return state, nil
}

func (s *Service) Step(ctx context.Context, state State, readers Readers, batch int) (State, error) {
	if batch <= 0 {
		return state, invalid("a batch must hold at least one item", "batch")
	}
	var (
		done bool
		err  error
	)
	switch state.Stage {
	case StageInventory:
		err = s.inventory(ctx, &state, readers)
		done = true
	case StageContent:
		done, err = s.content(ctx, &state, readers, batch)
	case StageStore:
		done, err = s.store(ctx, &state, readers, batch)
	case StageRender:
		done, err = s.render(ctx, &state, readers, batch)
	case StageDerive:
		err = s.derive(ctx, &state, readers)
		done = true
	case StageDone:
		return state, nil
	default:
		return state, invalid("the observation stage is not recognized", "stage")
	}
	if err != nil {
		return state, err
	}
	if done {
		state.Stage = nextStage(state.Stage)
	}
	return state, nil
}

func nextStage(stage Stage) Stage {
	switch stage {
	case StageInventory:
		return StageContent
	case StageContent:
		return StageStore
	case StageStore:
		return StageRender
	case StageRender:
		return StageDerive
	default:
		return StageDone
	}
}

func (s *Service) Run(ctx context.Context, siteID string, readers Readers, opts Options, batch int) (State, error) {
	state, err := s.Begin(ctx, siteID, readers, opts)
	if err != nil {
		return State{}, err
	}
	for i := 0; i < maxSteps && !state.Done(); i++ {
		if state, err = s.Step(ctx, state, readers, batch); err != nil {
			return state, err
		}
	}
	if !state.Done() {
		return state, errors.New(errors.Internal, "the observation did not finish within its step budget")
	}
	return state, nil
}
