package steps

import (
	"context"
	"strconv"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	NameSyncBack = string(run.StepSyncBack)

	CodePlanNotKept = "plan_not_kept"

	syncBackTimeout = 2 * time.Minute
)

type SyncResult struct {
	ModifiedAt  time.Time          `json:"modifiedAt"`
	URL         string             `json:"url"`
	Status      string             `json:"status"`
	ContentHash string             `json:"contentHash"`
	Source      string             `json:"source"`
	Findings    []content.Finding  `json:"findings"`
	Mismatches  []pagemap.Mismatch `json:"mismatches"`
	WPID        int64              `json:"wpId"`
	Links       int                `json:"links"`
}

func SyncBack(deps Deps) run.StepDef {
	return run.StepDef{
		Name:      NameSyncBack,
		Preflight: pluginPreflight(deps, NameSyncBack, "reads the page back through core, without its SEO meta"),
		Requires:  []run.ArtifactKind{run.ArtifactPublishResult},
		Produces:  []run.ArtifactKind{run.ArtifactSyncResult},
		Retry:     run.RetryPolicy{Max: 3},
		Timeout:   syncBackTimeout,
		Run: func(ctx context.Context, sc *run.StepContext) (run.Result, error) {
			published, found, err := decodeArtifact[PublishResult](sc, run.ArtifactPublishResult)
			if err != nil {
				return run.Result{}, err
			}
			if !found || published.WPID == 0 {
				return run.Result{}, errors.New(errors.Invalid, "the page was never published, so there is nothing to read back").
					WithDetail("pageId", sc.Page.ID)
			}

			itemType, err := itemTypeOf(sc.Page)
			if err != nil {
				return run.Result{}, err
			}
			client, err := clientFor(ctx, deps, sc.Run.SiteID)
			if err != nil {
				return run.Result{}, err
			}
			owner, err := deps.Sites.Get(ctx, sc.Run.SiteID)
			if err != nil {
				return run.Result{}, err
			}
			pages, err := deps.Pages.ListBySite(ctx, sc.Run.SiteID)
			if err != nil {
				return run.Result{}, err
			}

			item, err := client.GetItem(ctx, itemType, published.WPID)
			if err != nil {
				return run.Result{}, err
			}

			body, source, err := readBack(ctx, client, published.WPID, item.Content)
			if err != nil {
				return run.Result{}, err
			}

			doc, err := content.Parse(body)
			if err != nil {
				return run.Result{}, err
			}

			now := deps.now()
			index := pagemap.NewIndex(pages)
			links := observedOn(sc.Page, index, pagemap.NewSite(owner.BaseURL), doc.Links(), now)

			next := sc.Page
			next.WPID = &item.ID
			next.ContentHash = wp.ContentHash(body)
			next.Observed = pagemap.Observed{
				Link: permalinkOf(item), Slug: item.Slug, Status: item.Status,
				Title: item.Title, H1: headingOne(doc),
			}
			next.Drift = false
			next.LastSyncedAt = &now
			next.UpdatedAt = now
			if !item.Modified.IsZero() {
				modified := item.Modified.UTC()
				next.WPModifiedAt = &modified
			}

			if persistErr := persist(ctx, deps, next, links); persistErr != nil {
				return run.Result{}, persistErr
			}

			mismatches := next.Mismatches()
			result := SyncResult{
				WPID: item.ID, URL: item.Link, Status: item.Status, ContentHash: next.ContentHash,
				Source: source, Links: len(links), ModifiedAt: item.Modified.UTC(),
				Mismatches: mismatches, Findings: planFindings(next, mismatches),
			}
			blob, err := encode(result, "sync result")
			if err != nil {
				return run.Result{}, err
			}
			return run.Result{
				Artifacts: []run.Artifact{{Kind: run.ArtifactSyncResult, Blob: blob}},
				Message:   "read back " + sc.Page.Path + " with " + strconv.Itoa(len(links)) + " internal links",
			}, nil
		},
	}
}

func planFindings(page pagemap.Page, mismatches []pagemap.Mismatch) []content.Finding {
	out := make([]content.Finding, 0, len(mismatches))
	for i := range mismatches {
		out = append(out, content.Finding{
			Severity: content.SeverityWarn,
			Code:     CodePlanNotKept,
			Message: "the " + mismatches[i].Field + " of " + page.Path + " was planned as " +
				mismatches[i].Planned + " and the site holds " + mismatches[i].Actual,
			Details: map[string]any{
				"class": ClassNeedsHuman, "pageId": page.ID, "path": page.Path,
				"field": mismatches[i].Field, "planned": mismatches[i].Planned, "actual": mismatches[i].Actual,
			},
		})
	}
	return out
}

func readBack(ctx context.Context, client *wp.Client, wpID int64, fallback string) (body, source string, err error) {
	raw, err := client.GetRaw(ctx, wpID)
	if err == nil {
		return raw.Content, "plugin", nil
	}
	if wp.IsPluginMissing(err) {
		return fallback, "core", nil
	}
	return "", "", err
}

func persist(ctx context.Context, deps Deps, page pagemap.Page, links []pagemap.PageLink) error {
	apply := func(c context.Context) error {
		if err := deps.Pages.Update(c, page); err != nil {
			return err
		}
		return deps.Links.ReplaceForPage(c, page.ID, links)
	}
	if deps.UnitOfWork == nil {
		return apply(ctx)
	}
	return deps.UnitOfWork.Do(ctx, apply)
}
