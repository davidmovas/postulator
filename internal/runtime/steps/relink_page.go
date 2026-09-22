package steps

import (
	"context"
	stderrors "errors"
	"strconv"

	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const NameRelinkPage = string(run.StepRelinkPage)

type PlacedLink struct {
	PageID  string `json:"pageId"`
	Path    string `json:"path"`
	Anchor  string `json:"anchor"`
	Outcome string `json:"outcome"`
	Detail  string `json:"detail"`
}

type RelinkPageResult struct {
	PageID   string            `json:"pageId"`
	Path     string            `json:"path"`
	Placed   []PlacedLink      `json:"placed"`
	Findings []content.Finding `json:"findings"`
	WPID     int64             `json:"wpId"`
	Linked   int               `json:"linked"`
	Skipped  int               `json:"skipped"`
}

func RelinkPage(deps Deps) run.StepDef {
	return run.StepDef{
		Name:     NameRelinkPage,
		Requires: []run.ArtifactKind{run.ArtifactLinkContext},
		Produces: []run.ArtifactKind{run.ArtifactPublishResult, run.ArtifactRelinkResult},
		Retry:    run.RetryPolicy{Max: 2},
		Timeout:  relinkTimeout,
		Run: func(ctx context.Context, sc *run.StepContext) (run.Result, error) {
			lc, err := linkContextOf(sc)
			if err != nil {
				return run.Result{}, err
			}
			policy, err := effectivePolicy(ctx, deps, sc)
			if err != nil {
				return run.Result{}, err
			}
			if sc.Page.WPID == nil {
				return run.Result{
					Next:    run.TransitionPause,
					Reason:  run.PauseNeedsHuman,
					Message: "the page " + sc.Page.Path + " " + ReasonPageOffTheSite,
				}, nil
			}
			client, err := clientFor(ctx, deps, sc.Run.SiteID)
			if err != nil {
				return run.Result{}, err
			}

			work := pageRelink{
				result: RelinkPageResult{
					PageID: sc.Page.ID, Path: sc.Page.Path, WPID: *sc.Page.WPID,
					Placed: make([]PlacedLink, 0, len(lc.Targets)), Findings: make([]content.Finding, 0),
				},
				published: PublishResult{
					WPID: *sc.Page.WPID, URL: sc.Page.Observed.Link, Status: string(sc.Page.Status),
					ContentHash: sc.Page.ContentHash, SEOApplied: make([]string, 0),
					Skipped: make([]string, 0), Findings: make([]content.Finding, 0),
					Mismatches: make([]pagemap.Mismatch, 0),
				},
			}

			raw, err := client.GetRaw(ctx, *sc.Page.WPID)
			if err != nil {
				switch {
				case wp.IsPluginMissing(err):
					return standDown(work, sc, ReasonNoPlugin)
				case errors.IsCode(err, errors.NotFound):
					return standDown(work, sc, ReasonPageGone)
				default:
					return run.Result{}, err
				}
			}

			doc, err := content.Parse(raw.Content)
			if err != nil {
				return standDown(work, sc, ReasonPageUnreadable)
			}

			work.published.ContentHash = raw.ContentHash
			work.published.PreviousContent = raw.Content
			work.published.PreviousContentHash = raw.ContentHash
			placeTargets(&work, doc, lc, policy, sc.Page.ID)

			if work.result.Linked == 0 {
				return settleRelinkPage(work)
			}

			hash, err := client.PutRaw(ctx, *sc.Page.WPID, doc.HTML(), raw.ContentHash)
			if err != nil {
				if errors.IsCode(err, errors.Conflict) {
					return run.Result{
						Next:   run.TransitionPause,
						Reason: run.PauseNeedsHuman,
						Message: "the page " + sc.Page.Path + " changed on the site while it was being relinked: " +
							"it held " + raw.ContentHash + " and now holds " + currentHashOf(err),
					}, nil
				}
				return run.Result{}, err
			}
			work.published.ContentHash = hash

			pages, err := deps.Pages.ListBySite(ctx, sc.Run.SiteID)
			if err != nil {
				return run.Result{}, err
			}
			owner, err := deps.Sites.Get(ctx, sc.Run.SiteID)
			if err != nil {
				return run.Result{}, err
			}
			if adoptErr := adopt(ctx, deps, sc.Page, pagemap.NewIndex(pages),
				pagemap.NewSite(owner.BaseURL), doc, hash); adoptErr != nil {
				return run.Result{}, adoptErr
			}
			return settleRelinkPage(work)
		},
	}
}

type pageRelink struct {
	result    RelinkPageResult
	published PublishResult
}

func placeTargets(work *pageRelink, doc *content.Document, lc content.LinkContext,
	policy template.LinkPolicy, pageID string) {
	for i := range lc.Targets {
		if lc.Targets[i].PageID == pageID || lc.Targets[i].PageID == "" {
			continue
		}

		placement := content.InsertTarget(doc, lc, policy, lc.Targets[i])
		row := PlacedLink{PageID: lc.Targets[i].PageID, Path: lc.Targets[i].URL}
		if anchor, inserted := insertedAnchor(placement); inserted {
			row.Outcome, row.Anchor = OutcomeLinked, anchor
			work.result.Linked++
		} else {
			row.Outcome, row.Detail = OutcomeUnchanged, firstDetail(placement)
		}
		work.result.Placed = append(work.result.Placed, row)
	}
}

func standDown(work pageRelink, sc *run.StepContext, reason string) (run.Result, error) {
	work.result.Skipped++
	work.result.Findings = append(work.result.Findings, content.Finding{
		Severity: content.SeverityWarn,
		Code:     CodeRelinkSkipped,
		Message:  "the page " + sc.Page.Path + " was not relinked: " + reason,
		Details:  map[string]any{"pageId": sc.Page.ID, "path": sc.Page.Path, "reason": reason},
	})
	return settleRelinkPage(work)
}

func settleRelinkPage(work pageRelink) (run.Result, error) {
	published, err := encode(work.published, "publish result")
	if err != nil {
		return run.Result{}, err
	}
	relinked, err := encode(work.result, "relink result")
	if err != nil {
		return run.Result{}, err
	}

	return run.Result{
		Artifacts: []run.Artifact{
			{Kind: run.ArtifactPublishResult, Blob: published},
			{Kind: run.ArtifactRelinkResult, Blob: relinked},
		},
		Message: "placed " + strconv.Itoa(work.result.Linked) + " links on " + work.result.Path,
	}, nil
}

func currentHashOf(err error) string {
	var kernel *errors.Error
	if !stderrors.As(err, &kernel) || kernel == nil {
		return ""
	}
	value, ok := kernel.Details["currentHash"].(string)
	if !ok {
		return ""
	}
	return value
}
