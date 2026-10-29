package steps

import (
	"context"
	"strconv"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/application/observe"
	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/site"
)

const (
	NameSyncSite = string(run.StepSyncSite)

	ParamRenderRescan = run.ParamRenderRescan
	ParamFullWalk     = run.ParamFullWalk

	syncStepTimeout = 10 * time.Minute
)

type SiteSyncResult struct {
	StartedAt   time.Time         `json:"startedAt"`
	Source      string            `json:"source"`
	Stage       string            `json:"stage"`
	Findings    []content.Finding `json:"findings,omitempty"`
	Summary     *observe.Summary  `json:"summary,omitempty"`
	Batches     int               `json:"batches"`
	Pulled      int               `json:"pulled"`
	Created     int               `json:"created"`
	Updated     int               `json:"updated"`
	Products    int               `json:"products"`
	Rendered    int               `json:"rendered"`
	Unreachable int               `json:"unreachable"`
	Archived    int               `json:"archived"`
	Closed      int               `json:"closed"`
	Entities    int               `json:"entities"`
	Placed      int               `json:"placed"`
	Unplaced    int               `json:"unplaced"`
	UnplacedBy  map[string]int    `json:"unplacedBy,omitempty"`
	Full        bool              `json:"full"`
	Done        bool              `json:"done"`
}

func SyncSite(deps Deps) run.StepDef {
	return run.StepDef{
		Name:    NameSyncSite,
		Retry:   run.RetryPolicy{Max: 3},
		Timeout: syncStepTimeout,
		Run: func(ctx context.Context, sc *run.StepContext) (run.Result, error) {
			held, found, err := run.Get[observe.Progress](sc.Check, observe.CheckpointKey)
			if err != nil {
				return run.Result{}, err
			}
			owner, err := deps.Sites.Get(ctx, sc.Run.SiteID)
			if err != nil {
				return run.Result{}, err
			}
			client, err := clientFor(ctx, deps, sc.Run.SiteID)
			if err != nil {
				return run.Result{}, err
			}

			if !found || held.State.Stage == "" {
				if owner, err = adoptExtensions(ctx, deps, client, owner); err != nil {
					return run.Result{}, err
				}
				readers, readersErr := observeReaders(ctx, client, owner)
				if readersErr != nil {
					return run.Result{}, readersErr
				}
				state, beginErr := deps.Observe.Begin(ctx, owner.ID, readers, optionsOf(sc))
				if beginErr != nil {
					return run.Result{}, beginErr
				}
				held = observe.Progress{State: state}
			}
			readers, err := observeReaders(ctx, client, owner)
			if err != nil {
				return run.Result{}, err
			}
			if held.State, err = deps.Observe.Step(ctx, held.State, readers, batchSize(deps)); err != nil {
				return run.Result{}, err
			}
			held.Batches++
			return syncResult(ctx, deps, owner, held)
		},
	}
}

func optionsOf(sc *run.StepContext) observe.Options {
	return observe.Options{RenderRescan: sc.BoolParam(ParamRenderRescan), Full: sc.BoolParam(ParamFullWalk)}
}

func syncResult(ctx context.Context, deps Deps, owner site.Site, held observe.Progress) (run.Result, error) {
	checkpoint := run.NewCheckpoint()
	if err := run.Set(checkpoint, observe.CheckpointKey, held); err != nil {
		return run.Result{}, err
	}

	state := held.State
	report := SiteSyncResult{
		StartedAt: state.StartedAt, Source: state.Source, Stage: string(state.Stage), Findings: state.Findings,
		Batches: held.Batches, Pulled: state.Counts.Pulled, Created: state.Counts.Created, Updated: state.Counts.Updated,
		Products: state.Counts.Products, Rendered: state.Counts.Rendered, Unreachable: state.Counts.Unreachable,
		Archived: state.Counts.Archived, Closed: state.Counts.Closed, Entities: state.Counts.Entities, Placed: state.Counts.Placed,
		Unplaced: state.Counts.Unplaced, UnplacedBy: state.Counts.UnplacedBy, Full: state.Full, Done: state.Done(),
	}
	message := "observed " + strconv.Itoa(state.Counts.Pulled) + " items of " + owner.Name + " through the " + state.Source +
		" (" + string(state.Stage) + ")"
	if report.Done {
		summary, err := deps.Observe.Summary(ctx, owner.ID)
		if err != nil {
			return run.Result{}, err
		}
		report.Summary = &summary
		message = "observed " + strconv.Itoa(summary.Observed) + " pages of " + owner.Name + ": " +
			strconv.Itoa(summary.Rendered) + " rendered, " + strconv.Itoa(summary.TrailsResolved) + " trails resolved, " +
			strconv.Itoa(openFindings(summary)) + " open findings"
	}

	blob, err := encode(report, "site sync result")
	if err != nil {
		return run.Result{}, err
	}
	result := run.Result{
		Artifacts:  []run.Artifact{{Kind: run.ArtifactSyncResult, Blob: blob}},
		Checkpoint: checkpoint,
		Message:    message,
	}
	if !report.Done {
		result.Next = run.TransitionWait
		result.WakeAt = deps.now()
	}
	return result, nil
}

func openFindings(summary observe.Summary) int {
	total := 0
	for _, count := range summary.Findings {
		total += count
	}
	return total
}

func adoptExtensions(ctx context.Context, deps Deps, client *wp.Client, owner site.Site) (site.Site, error) {
	plugin := site.PluginState{Capabilities: []string{}}
	capabilities, err := client.Capabilities(ctx)
	switch {
	case err == nil:
		plugin = site.PluginState{
			Installed:    true,
			Version:      capabilities.Version,
			Capabilities: capabilities.Names,
			SEOPlugin:    capabilities.SEOPlugin,
		}
	case !wp.IsPluginMissing(err):
		return site.Site{}, err
	}

	found, err := client.CommerceOr(ctx, wp.Commerce(owner.Commerce))
	if err != nil {
		return site.Site{}, err
	}
	commerce := site.Commerce(found)

	if deps.SiteWriter == nil || (samePlugin(owner.Plugin, plugin) && commerce == owner.Commerce) {
		owner.Plugin, owner.Commerce = plugin, commerce
		return owner, nil
	}
	next := owner
	next.Plugin = plugin
	next.Commerce = commerce
	next.UpdatedAt = deps.now()
	if err := deps.SiteWriter.Update(ctx, next); err != nil {
		return site.Site{}, err
	}
	return next, nil
}

func samePlugin(current, next site.PluginState) bool {
	if current.Installed != next.Installed || current.Version != next.Version ||
		current.SEOPlugin != next.SEOPlugin || len(current.Capabilities) != len(next.Capabilities) {
		return false
	}
	for i := range current.Capabilities {
		if current.Capabilities[i] != next.Capabilities[i] {
			return false
		}
	}
	return true
}
