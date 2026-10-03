package steps

import (
	"context"
	stderrors "errors"
	"time"

	"github.com/davidmovas/postulator/internal/domain/run"
)

const (
	checkpointLinks = "links"

	pureStepTimeout = 30 * time.Second
	writerTimeout   = 15 * time.Minute
	editorTimeout   = 3 * time.Minute
)

func all(deps Deps) []run.StepDef {
	return []run.StepDef{
		ResolveContext(deps),
		GenerateBody(deps),
		GenerateMeta(deps),
		InsertLinks(deps),
		RepairLinks(deps),
		GenerateImages(deps),
		Validate(deps),
		Judge(deps),
		Publish(deps),
		RelinkNeighbors(deps),
		SyncBack(deps),
		Report(deps),
		RepairHierarchy(deps),
		SyncSite(deps),
		RelinkPage(deps),
		Revert(deps),
	}
}

func Register(registry *run.Registry, deps Deps) error {
	defs := all(deps)
	for i := range defs {
		if err := registry.Register(defs[i]); err != nil {
			return err
		}
	}
	return nil
}

func stopped(ctx context.Context) bool {
	return stderrors.Is(ctx.Err(), context.Canceled)
}

func needsHuman(message string) run.Result {
	return run.Result{Next: run.TransitionPause, Reason: run.PauseNeedsHuman, Message: message}
}
