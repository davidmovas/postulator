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
	PageID   string `json:"pageId"`
	Path     string `json:"path"`
	Anchor   string `json:"anchor"`
	Sentence string `json:"sentence,omitempty"`
	Outcome  string `json:"outcome"`
	Detail   string `json:"detail"`
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
		Name:      NameRelinkPage,
		Preflight: pluginPreflight(deps, NameRelinkPage, "cannot read or write the page and stands down"),
		Requires:  []run.ArtifactKind{run.ArtifactLinkContext},
		Produces:  []run.ArtifactKind{run.ArtifactPublishResult, run.ArtifactRelinkResult},
		Retry:     run.RetryPolicy{Max: 2},
		Timeout:   relinkTimeout,
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

			pages, err := deps.Pages.ListBySite(ctx, sc.Run.SiteID)
			if err != nil {
				return run.Result{}, err
			}

			work.published.ContentHash = raw.ContentHash
			work.published.PreviousContent = raw.Content
			work.published.PreviousContentHash = raw.ContentHash
			placeTargets(&work, doc, lc, policy, sc.Page, onTheSite(pages))

			if work.result.Linked == 0 {
				return settleRelinkPage(work)
			}

			linked, err := doc.Render()
			if err != nil {
				return run.Result{}, err
			}

			hash, err := client.PutRaw(ctx, *sc.Page.WPID, linked, raw.ContentHash)
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
	policy template.LinkPolicy, page pagemap.Page, live map[string]bool) {
	for i := range lc.Targets {
		target := lc.Targets[i]
		if target.PageID == page.ID || target.PageID == "" {
			continue
		}

		var (
			placement content.InsertResult
			sentence  string
		)
		if live[target.PageID] {
			placement, sentence = backfill(doc, lc, policy, target)
		} else {
			placement = content.InsertTarget(doc, lc, policy, target)
		}

		row := PlacedLink{PageID: target.PageID, Path: target.URL}
		if anchor, inserted := insertedAnchor(placement); inserted {
			row.Outcome, row.Anchor, row.Sentence = OutcomeLinked, anchor, sentence
			work.result.Linked++
			if sentence != "" {
				work.result.Findings = append(work.result.Findings, templatedPageFinding(page, target, sentence))
			}
		} else {
			row.Outcome, row.Detail = OutcomeUnchanged, firstDetail(placement)
			if missing, owed := unplaced(placement, target, live[target.PageID], page); owed {
				work.result.Findings = append(work.result.Findings, missing)
			} else if !live[target.PageID] && !alreadyLinked(placement) {
				row.Detail = "it is linked once " + target.URL + " is published"
			}
		}
		work.result.Placed = append(work.result.Placed, row)
	}
	work.result.Findings = append(work.result.Findings, content.Unpublished(doc, lc, live, page.ID)...)
}

func onTheSite(pages []pagemap.Page) map[string]bool {
	live := make(map[string]bool, len(pages))
	for i := range pages {
		if pages[i].WPID != nil {
			live[pages[i].ID] = true
		}
	}
	return live
}

func unplaced(placement content.InsertResult, target content.LinkTarget, live bool,
	page pagemap.Page) (content.Finding, bool) {
	if len(placement.Decisions) == 0 || alreadyLinked(placement) {
		return content.Finding{}, false
	}
	budget := placement.Decisions[0].Outcome == content.OutcomeCapReached
	if !budget && !live {
		return content.Finding{}, false
	}
	return content.Finding{
		Severity: content.SeverityWarn,
		Code:     content.CodeTargetMissing,
		Message:  "the page does not link to " + target.URL + ": " + placement.Decisions[0].Detail,
		Details: map[string]any{
			"pageId": page.ID, "targetPageId": target.PageID, "relation": string(target.Relation),
			"reason": string(placement.Decisions[0].Outcome),
		},
	}, true
}

func templatedPageFinding(page pagemap.Page, target content.LinkTarget, sentence string) content.Finding {
	return content.Finding{
		Severity: content.SeverityWarn,
		Code:     CodeRelinkPhraseTemplated,
		Message: "the page " + page.Path + " carried no anchor for " + target.URL + ", so a plain sentence was added " +
			"to it; rewrite it on the site if it reads poorly",
		Details: map[string]any{"pageId": page.ID, "targetPageId": target.PageID, "sentence": sentence},
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
