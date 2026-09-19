package steps

import (
	"context"

	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/run"
)

const NameInsertLinks = string(run.StepInsertLinks)

func InsertLinks(deps Deps) run.StepDef {
	return run.StepDef{
		Name:     NameInsertLinks,
		Requires: []run.ArtifactKind{run.ArtifactLinkContext, run.ArtifactBodyHTML},
		Produces: []run.ArtifactKind{run.ArtifactBodyHTML},
		Retry:    run.RetryPolicy{Max: 2},
		Timeout:  pureStepTimeout,
		Run: func(ctx context.Context, sc *run.StepContext) (run.Result, error) {
			lc, err := linkContextOf(sc)
			if err != nil {
				return run.Result{}, err
			}
			doc, err := bodyOf(sc)
			if err != nil {
				return run.Result{}, err
			}
			policy, err := effectivePolicy(ctx, deps, sc)
			if err != nil {
				return run.Result{}, err
			}

			result := content.InsertLinks(doc, lc, policy)

			checkpoint := run.NewCheckpoint()
			if setErr := run.Set(checkpoint, checkpointLinks, result); setErr != nil {
				return run.Result{}, setErr
			}

			return run.Result{
				Artifacts:  []run.Artifact{{Kind: run.ArtifactBodyHTML, Blob: []byte(doc.HTML())}},
				Checkpoint: checkpoint,
				Message:    "placed " + plural(len(result.Placed)),
			}, nil
		},
	}
}
