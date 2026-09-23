package steps

import (
	"context"
	"strconv"

	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const NameRepairHierarchy = string(run.StepRepairHierarchy)

func RepairHierarchy(deps Deps) run.StepDef {
	return run.StepDef{
		Name:     NameRepairHierarchy,
		Produces: []run.ArtifactKind{run.ArtifactPublishResult},
		Retry:    run.RetryPolicy{Max: 3},
		Timeout:  publishTimeout,
		Run: func(ctx context.Context, sc *run.StepContext) (run.Result, error) {
			itemType, err := itemTypeOf(sc.Page)
			if err != nil {
				return run.Result{}, err
			}
			if sc.Page.WPID == nil {
				return run.Result{}, errors.New(errors.Invalid,
					"the page is not on the site, so there is nothing to move").
					WithDetail("pageId", sc.Page.ID).WithDetail("path", sc.Page.Path)
			}
			client, err := clientFor(ctx, deps, sc.Run.SiteID)
			if err != nil {
				return run.Result{}, err
			}

			placement, err := parentOf(ctx, deps, sc.Page)
			if err != nil {
				return run.Result{}, err
			}
			if placement.pending {
				return holdForParent(sc, placement), nil
			}

			moved, err := client.UpdateItem(ctx, itemType, *sc.Page.WPID, wp.UpdateItem{
				Parent: &placement.wpID, Slug: &sc.Page.Slug,
			})
			if err != nil {
				return run.Result{}, err
			}

			mismatches := placedAs(sc.Page, placement.wpID, moved)
			if len(mismatches) > 0 {
				return refuseMismatch(sc, mismatches), nil
			}

			result := PublishResult{
				WPID: moved.ID, URL: moved.Link, Status: moved.Status,
				ContentHash: sc.Page.ContentHash, Created: false,
				SEOApplied: make([]string, 0), Skipped: make([]string, 0),
				Findings: make([]content.Finding, 0), Mismatches: mismatches,
			}
			if rememberErr := remember(ctx, deps, sc, moved); rememberErr != nil {
				return run.Result{}, rememberErr
			}

			blob, err := encode(result, "publish result")
			if err != nil {
				return run.Result{}, err
			}
			return run.Result{
				Artifacts: []run.Artifact{{Kind: run.ArtifactPublishResult, Blob: blob}},
				Message: "moved " + sc.Page.Path + " under " +
					strconv.FormatInt(placement.wpID, 10),
			}, nil
		},
	}
}

func placedAs(page pagemap.Page, parent int64, item wp.Item) []pagemap.Mismatch {
	checked := pagemap.Page{Path: page.Path, Slug: page.Slug, Observed: observedOf(item)}
	found := checked.Mismatches()
	if item.Parent != parent {
		found = append(found, pagemap.Mismatch{
			Field:   FieldParent,
			Planned: strconv.FormatInt(parent, 10),
			Actual:  strconv.FormatInt(item.Parent, 10),
		})
	}
	return found
}

func remember(ctx context.Context, deps Deps, sc *run.StepContext, item wp.Item) error {
	now := deps.now()
	next := sc.Page
	next.WPID = &item.ID
	next.Observed = observedOf(item)
	next.LastSyncedAt = &now
	next.UpdatedAt = now
	if !item.Modified.IsZero() {
		modified := item.Modified.UTC()
		next.WPModifiedAt = &modified
	}
	return deps.Pages.Update(ctx, next)
}
