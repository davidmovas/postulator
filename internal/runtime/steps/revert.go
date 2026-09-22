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
	NameRevert = run.RevertStep

	OutcomeTrashed   = "trashed"
	OutcomeRestored  = "restored"
	OutcomeNeedsHand = "needs_human"

	CodeRevertNeedsHuman = "revert_needs_human"
	CodeRevertMetaKept   = "revert_meta_kept"

	ReasonRevertNoRecord = "the run kept no record of what it published to this page, " +
		"so there is nothing to put back"
	ReasonRevertNoBody = "the run kept no copy of the body it replaced, which needs the companion plugin; " +
		"the WordPress revision history of the page is the way back"
	ReasonRevertNoPlugin = "the companion plugin is not installed, so the body cannot be written back; " +
		"the WordPress revision history of the page is the way back"
	ReasonRevertGone       = "the page is no longer on the site"
	ReasonRevertNoNeighbor = "the neighbor is no longer on the site"
	ReasonRevertNoBefore   = "the run kept no copy of the neighbor content it replaced"

	revertTimeout = 5 * time.Minute
)

type RevertedNeighbor struct {
	PageID  string `json:"pageId"`
	Path    string `json:"path"`
	Outcome string `json:"outcome"`
	Detail  string `json:"detail"`
	WPID    int64  `json:"wpId"`
}

type RevertResult struct {
	PageID    string             `json:"pageId"`
	Path      string             `json:"path"`
	Outcome   string             `json:"outcome"`
	Detail    string             `json:"detail"`
	SourceRun string             `json:"sourceRunId"`
	Neighbors []RevertedNeighbor `json:"neighbors"`
	Findings  []content.Finding  `json:"findings"`
	WPID      int64              `json:"wpId"`
	Created   bool               `json:"created"`
}

func Revert(deps Deps) run.StepDef {
	return run.StepDef{
		Name:     NameRevert,
		Produces: []run.ArtifactKind{run.ArtifactRevertResult},
		Retry:    run.RetryPolicy{Max: 2},
		Timeout:  revertTimeout,
		Run: func(ctx context.Context, sc *run.StepContext) (run.Result, error) {
			source, err := sourceRun(sc)
			if err != nil {
				return run.Result{}, err
			}
			page, err := deps.Pages.Get(ctx, sc.Item.TargetID)
			if err != nil {
				return run.Result{}, err
			}

			result := RevertResult{
				PageID: page.ID, Path: page.Path, SourceRun: source,
				Neighbors: make([]RevertedNeighbor, 0), Findings: make([]content.Finding, 0),
			}

			published, relinked, found, err := recordOf(ctx, deps, source, page.ID)
			if err != nil {
				return run.Result{}, err
			}
			if !found {
				return handBack(result, ReasonRevertNoRecord)
			}
			result.WPID, result.Created = published.WPID, published.Created

			client, err := clientFor(ctx, deps, sc.Run.SiteID)
			if err != nil {
				return run.Result{}, err
			}
			work := revertWork{page: page, client: client, published: published}

			if reason, undone := undoPage(ctx, deps, sc, &result, work); !undone {
				return handBack(result, reason)
			}
			for i := range relinked.Neighbors {
				if reason, undone := undoNeighbor(ctx, deps, sc, &result, work, relinked.Neighbors[i]); !undone {
					return handBack(result, reason)
				}
			}
			return settleRevert(result)
		},
	}
}

type revertWork struct {
	page      pagemap.Page
	client    *wp.Client
	published PublishResult
}

func sourceRun(sc *run.StepContext) (string, error) {
	if sc.Run.ParentRunID == nil || *sc.Run.ParentRunID == "" {
		return "", errors.New(errors.Invalid, "a revert names no run to revert").
			WithDetail("runId", sc.Run.ID)
	}
	return *sc.Run.ParentRunID, nil
}

func recordOf(ctx context.Context, deps Deps, sourceRunID, pageID string) (PublishResult, RelinkResult, bool, error) {
	items, err := deps.Items.ByRun(ctx, sourceRunID)
	if err != nil {
		return PublishResult{}, RelinkResult{}, false, err
	}

	for i := range items {
		if items[i].TargetID != pageID {
			continue
		}
		stored, listErr := deps.Artifacts.ByItem(ctx, items[i].ID)
		if listErr != nil {
			return PublishResult{}, RelinkResult{}, false, listErr
		}

		published, ok, decodeErr := blobOf[PublishResult](stored, run.ArtifactPublishResult)
		if decodeErr != nil {
			return PublishResult{}, RelinkResult{}, false, decodeErr
		}
		if !ok {
			continue
		}
		relinked, _, decodeErr := blobOf[RelinkResult](stored, run.ArtifactRelinkResult)
		if decodeErr != nil {
			return PublishResult{}, RelinkResult{}, false, decodeErr
		}
		return published, relinked, true, nil
	}
	return PublishResult{}, RelinkResult{}, false, nil
}

func undoPage(ctx context.Context, deps Deps, sc *run.StepContext, result *RevertResult,
	work revertWork) (reason string, undone bool) {
	if work.published.Created {
		return takeOffTheSite(ctx, deps, result, work)
	}
	return putTheBodyBack(ctx, deps, sc, result, work)
}

func takeOffTheSite(ctx context.Context, deps Deps, result *RevertResult, work revertWork) (string, bool) {
	itemType, err := itemTypeOf(work.page)
	if err != nil {
		return err.Error(), false
	}

	detail := "moved to the WordPress trash"
	if trashErr := work.client.DeleteItem(ctx, itemType, work.published.WPID, false); trashErr != nil {
		if !errors.IsCode(trashErr, errors.NotFound) {
			return trashErr.Error(), false
		}
		detail = "it was already off the site"
	}

	if planErr := replan(ctx, deps, work.page); planErr != nil {
		return planErr.Error(), false
	}
	result.Outcome = OutcomeTrashed
	result.Detail = detail
	return "", true
}

func putTheBodyBack(ctx context.Context, deps Deps, sc *run.StepContext, result *RevertResult,
	work revertWork) (string, bool) {
	if work.published.PreviousContent == "" {
		return ReasonRevertNoBody, false
	}

	raw, err := work.client.GetRaw(ctx, work.published.WPID)
	switch {
	case wp.IsPluginMissing(err):
		return ReasonRevertNoPlugin, false
	case errors.IsCode(err, errors.NotFound):
		return ReasonRevertGone, false
	case err != nil:
		return err.Error(), false
	}

	hash := raw.ContentHash
	restored := raw.ContentHash == work.published.PreviousContentHash
	switch {
	case restored:
	case raw.ContentHash != work.published.ContentHash:
		return editedSince(work.published.ContentHash, raw.ContentHash), false
	default:
		written, putErr := work.client.PutRaw(ctx, work.published.WPID, work.published.PreviousContent, raw.ContentHash)
		if putErr != nil {
			if errors.IsCode(putErr, errors.Conflict) {
				return writtenUnderUs(raw.ContentHash), false
			}
			return putErr.Error(), false
		}
		hash = written
	}

	if adoptErr := readopt(ctx, deps, sc, work.page, work.published.PreviousContent, hash); adoptErr != nil {
		return adoptErr.Error(), false
	}
	result.Outcome = OutcomeRestored
	result.Detail = "the body the run replaced was written back"
	if len(work.published.SEOApplied) > 0 {
		result.Findings = append(result.Findings, metaKept(work.page, work.published.SEOApplied))
	}
	return "", true
}

func undoNeighbor(ctx context.Context, deps Deps, sc *run.StepContext, result *RevertResult,
	work revertWork, neighbor NeighborResult) (string, bool) {
	if neighbor.Outcome != OutcomeLinked {
		return "", true
	}
	if neighbor.Before.HTML == "" {
		return ReasonRevertNoBefore, false
	}

	raw, err := work.client.GetRaw(ctx, neighbor.WPID)
	switch {
	case wp.IsPluginMissing(err):
		return ReasonRevertNoPlugin, false
	case errors.IsCode(err, errors.NotFound):
		return ReasonRevertNoNeighbor, false
	case err != nil:
		return err.Error(), false
	}

	stored, known := neighborPage(ctx, deps, sc, neighbor.PageID)
	hash := raw.ContentHash
	switch {
	case raw.ContentHash == neighbor.Before.Hash:
	case !known || stored.ContentHash == "":
		return unknownSince(raw.ContentHash), false
	case raw.ContentHash != stored.ContentHash:
		return editedSince(stored.ContentHash, raw.ContentHash), false
	default:
		written, putErr := work.client.PutRaw(ctx, neighbor.WPID, neighbor.Before.HTML, raw.ContentHash)
		if putErr != nil {
			if errors.IsCode(putErr, errors.Conflict) {
				return writtenUnderUs(raw.ContentHash), false
			}
			return putErr.Error(), false
		}
		hash = written
	}

	if known {
		if adoptErr := readopt(ctx, deps, sc, stored, neighbor.Before.HTML, hash); adoptErr != nil {
			return adoptErr.Error(), false
		}
	}
	result.Neighbors = append(result.Neighbors, RevertedNeighbor{
		PageID: neighbor.PageID, Path: neighbor.Path, WPID: neighbor.WPID,
		Outcome: OutcomeRestored, Detail: "the content the relink replaced was written back",
	})
	return "", true
}

func neighborPage(ctx context.Context, deps Deps, sc *run.StepContext, pageID string) (pagemap.Page, bool) {
	stored, err := deps.Pages.Get(ctx, pageID)
	if err != nil || stored.SiteID != sc.Run.SiteID {
		return pagemap.Page{}, false
	}
	return stored, true
}

func editedSince(left, right string) string {
	return "the page was edited on the site since the run wrote it: the run left " + left +
		" and the site now holds " + right
}

func unknownSince(read string) string {
	return "nothing here records what the relink left on the neighbor, so a human edit since cannot " +
		"be told apart from it; the site holds " + read
}

func writtenUnderUs(read string) string {
	return "the page changed on the site while the revert was writing it: it held " + read +
		" when the revert read it"
}

func replan(ctx context.Context, deps Deps, page pagemap.Page) error {
	now := deps.now()
	next := page
	next.WPID = nil
	next.Status = pagemap.StatusPlanned
	next.ContentHash = ""
	next.Observed = pagemap.Observed{}
	next.WPModifiedAt = nil
	next.LastSyncedAt = nil
	next.Drift = false
	next.UpdatedAt = now
	return persist(ctx, deps, next, []pagemap.PageLink{})
}

func readopt(ctx context.Context, deps Deps, sc *run.StepContext, page pagemap.Page, body, hash string) error {
	owner, err := deps.Sites.Get(ctx, sc.Run.SiteID)
	if err != nil {
		return err
	}
	pages, err := deps.Pages.ListBySite(ctx, sc.Run.SiteID)
	if err != nil {
		return err
	}
	doc, err := content.Parse(body)
	if err != nil {
		return errors.Wrap(err, errors.Internal, "read the body the revert wrote back")
	}

	now := deps.now()
	next := page
	next.ContentHash = hash
	next.Drift = false
	next.LastSyncedAt = &now
	next.UpdatedAt = now
	links := observedOn(page, pagemap.NewIndex(pages), pagemap.NewSite(owner.BaseURL), doc.Links(), now)
	return persist(ctx, deps, next, links)
}

func metaKept(page pagemap.Page, applied []string) content.Finding {
	return content.Finding{
		Severity: content.SeverityWarn,
		Code:     CodeRevertMetaKept,
		Message: "the SEO meta the run wrote to " + page.Path +
			" stays as it is: the companion plugin offers no read of what was there before",
		Details: map[string]any{"pageId": page.ID, "path": page.Path, "fields": applied},
	}
}

func handBack(result RevertResult, reason string) (run.Result, error) {
	result.Outcome = OutcomeNeedsHand
	result.Detail = reason
	result.Findings = append(result.Findings, content.Finding{
		Severity: content.SeverityError,
		Code:     CodeRevertNeedsHuman,
		Message:  "the revert of " + result.Path + " needs a human: " + reason,
		Details:  map[string]any{"class": ClassNeedsHuman, "pageId": result.PageID, "reason": reason},
	})

	blob, err := encode(result, "revert result")
	if err != nil {
		return run.Result{}, err
	}
	return run.Result{
		Artifacts: []run.Artifact{{Kind: run.ArtifactRevertResult, Blob: blob}},
		Next:      run.TransitionPause,
		Reason:    run.PauseNeedsHuman,
		Message:   "the revert of " + result.Path + " needs a human: " + reason,
	}, nil
}

func settleRevert(result RevertResult) (run.Result, error) {
	blob, err := encode(result, "revert result")
	if err != nil {
		return run.Result{}, err
	}
	return run.Result{
		Artifacts: []run.Artifact{{Kind: run.ArtifactRevertResult, Blob: blob}},
		Message: "reverted " + result.Path + " and " + strconv.Itoa(len(result.Neighbors)) +
			" of its neighbors",
	}, nil
}
