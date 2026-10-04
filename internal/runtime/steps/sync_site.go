package steps

import (
	"context"
	"strconv"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/site"
)

const (
	NameSyncSite = string(run.StepSyncSite)

	CapabilityBulk = "bulk"

	SourcePlugin = "plugin"
	SourceCore   = "core"

	CodePathTakenOnSite = "path_taken_on_site"

	checkpointSync = "sync"

	syncStepTimeout = 10 * time.Minute
)

type SiteSyncResult struct {
	StartedAt time.Time         `json:"startedAt"`
	Source    string            `json:"source"`
	Cursor    string            `json:"cursor"`
	Findings  []content.Finding `json:"findings,omitempty"`
	Batches   int               `json:"batches"`
	Pulled    int               `json:"pulled"`
	Created   int               `json:"created"`
	Updated   int               `json:"updated"`
	Drifted   int               `json:"drifted"`
	Archived  int               `json:"archived"`
	Done      bool              `json:"done"`
}

func SyncSite(deps Deps) run.StepDef {
	return run.StepDef{
		Name:    NameSyncSite,
		Retry:   run.RetryPolicy{Max: 3},
		Timeout: syncStepTimeout,
		Run: func(ctx context.Context, sc *run.StepContext) (run.Result, error) {
			held, _, err := run.Get[SiteSyncResult](sc.Check, checkpointSync)
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

			state, err := batchState(ctx, deps, client, owner, held)
			if err != nil {
				return run.Result{}, err
			}
			batch, next, err := pull(ctx, client, state, batchSize(deps), pagemap.NewSite(owner.BaseURL))
			if err != nil {
				return run.Result{}, err
			}
			if reconcileErr := reconcile(ctx, deps, owner, batch, &state); reconcileErr != nil {
				return run.Result{}, reconcileErr
			}
			state.advance(len(batch), next)
			if state.Done {
				if finishErr := finishSync(ctx, deps, owner.ID, &state); finishErr != nil {
					return run.Result{}, finishErr
				}
			}
			return syncResult(deps, owner, state)
		},
	}
}

func batchState(ctx context.Context, deps Deps, client *wp.Client, owner site.Site,
	state SiteSyncResult) (SiteSyncResult, error) {
	bulk, err := adoptExtensions(ctx, deps, client, owner, state.Batches == 0)
	if err != nil {
		return SiteSyncResult{}, err
	}

	source := SourceCore
	if bulk {
		source = SourcePlugin
	}
	if state.Source != "" && state.Source != source {
		state = SiteSyncResult{}
	}
	state.Source = source
	if state.StartedAt.IsZero() {
		state.StartedAt = deps.now()
	}
	return state, nil
}

func (s *SiteSyncResult) advance(pulled int, next string) {
	s.Batches++
	s.Pulled += pulled
	s.Cursor = next
	s.Done = next == ""
}

func finishSync(ctx context.Context, deps Deps, siteID string, state *SiteSyncResult) error {
	if err := archiveAbsent(ctx, deps, siteID, state); err != nil {
		return err
	}
	if err := linkParents(ctx, deps, siteID); err != nil {
		return err
	}
	if err := resolveLinks(ctx, deps, siteID); err != nil {
		return err
	}
	return announcePages(deps, siteID)
}

func syncResult(deps Deps, owner site.Site, state SiteSyncResult) (run.Result, error) {
	checkpoint := run.NewCheckpoint()
	if err := run.Set(checkpoint, checkpointSync, state); err != nil {
		return run.Result{}, err
	}

	blob, err := encode(state, "site sync result")
	if err != nil {
		return run.Result{}, err
	}

	result := run.Result{
		Artifacts:  []run.Artifact{{Kind: run.ArtifactSyncResult, Blob: blob}},
		Checkpoint: checkpoint,
		Message: "pulled " + strconv.Itoa(state.Pulled) + " items from " + owner.Name +
			" through the " + state.Source,
	}
	if !state.Done {
		result.Next = run.TransitionWait
		result.WakeAt = deps.now()
	}
	return result, nil
}

func adoptExtensions(ctx context.Context, deps Deps, client *wp.Client, owner site.Site, withStore bool) (bool, error) {
	plugin := site.PluginState{Capabilities: []string{}}
	bulk := false
	capabilities, err := client.Capabilities(ctx)
	switch {
	case err == nil:
		bulk = capabilities.Has(CapabilityBulk)
		plugin = site.PluginState{
			Installed:    true,
			Version:      capabilities.Version,
			Capabilities: capabilities.Names,
			SEOPlugin:    capabilities.SEOPlugin,
		}
	case !wp.IsPluginMissing(err):
		return false, err
	}

	commerce := owner.Commerce
	if withStore {
		found, storeErr := client.CommerceOr(ctx, wp.Commerce(owner.Commerce))
		if storeErr != nil {
			return false, storeErr
		}
		commerce = site.Commerce(found)
	}

	if deps.SiteWriter == nil || (samePlugin(owner.Plugin, plugin) && commerce == owner.Commerce) {
		return bulk, nil
	}
	next := owner
	next.Plugin = plugin
	next.Commerce = commerce
	next.UpdatedAt = deps.now()
	return bulk, deps.SiteWriter.Update(ctx, next)
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

func announcePages(deps Deps, siteID string) error {
	if deps.Publisher == nil {
		return nil
	}
	return deps.Publisher.Publish(events.PagesChanged, events.PagesChangedPayload{SiteID: siteID})
}
