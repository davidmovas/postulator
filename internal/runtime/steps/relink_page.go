package steps

import (
	"context"
	"strconv"

	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	NameRelinkPage = string(run.StepRelinkPage)

	ReasonPageOffTheSite = "is not on the site, so there is nothing to relink; publish it first"
	ReasonPageIsATerm    = "is a product category, whose description the companion plugin does not write"
	ReasonPageGone       = "the page is no longer on the site"
	ReasonPageUnreadable = "the stored content of the page could not be read as HTML"
)

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
			policy, err := effectivePolicy(ctx, deps, sc.Run.SiteID, sc.Spec)
			if err != nil {
				return run.Result{}, err
			}
			if reason, held := offLimits(sc.Page); held {
				return needsHuman("the page " + sc.Page.Path + " " + reason), nil
			}
			client, err := clientFor(ctx, deps, sc.Run.SiteID)
			if err != nil {
				return run.Result{}, err
			}

			work := newPageRelink(sc.Page, len(lc.Targets))
			body, reason, err := readSiteBody(ctx, client, sc.Page, ReasonPageGone, ReasonPageUnreadable)
			if err != nil {
				return run.Result{}, err
			}
			if reason != "" {
				return standDown(work, sc, reason)
			}

			pages, err := deps.Pages.ListBySite(ctx, sc.Run.SiteID)
			if err != nil {
				return run.Result{}, err
			}
			work.read(body.raw)
			work.place(body.doc, lc, policy, sc.Page, onTheSite(pages))
			if work.result.Linked == 0 {
				return settleRelinkPage(work)
			}
			return writeRelinkedPage(ctx, deps, client, sc, work, body, pages)
		},
	}
}

func offLimits(page pagemap.Page) (reason string, held bool) {
	switch {
	case page.WPID == nil:
		return ReasonPageOffTheSite, true
	case page.WPType == pagemap.WPProductCategory:
		return ReasonPageIsATerm, true
	default:
		return "", false
	}
}

type pageRelink struct {
	result    RelinkPageResult
	published PublishResult
}

func newPageRelink(page pagemap.Page, targets int) pageRelink {
	return pageRelink{
		result: RelinkPageResult{
			PageID: page.ID, Path: page.Path, WPID: *page.WPID,
			Placed: make([]PlacedLink, 0, targets), Findings: make([]content.Finding, 0),
		},
		published: PublishResult{
			WPID: *page.WPID, URL: page.Observed.Link, Status: string(page.Status),
			ContentHash: page.ContentHash, SEOApplied: make([]string, 0),
			Skipped: make([]string, 0), Findings: make([]content.Finding, 0),
			Mismatches: make([]pagemap.Mismatch, 0),
		},
	}
}

func (w *pageRelink) read(raw wp.RawContent) {
	w.published.ContentHash = raw.ContentHash
	w.published.PreviousContent = raw.Content
	w.published.PreviousContentHash = raw.ContentHash
}

func (w *pageRelink) place(doc *content.Document, lc content.LinkContext,
	policy template.LinkPolicy, page pagemap.Page, live map[string]bool) {
	for i := range lc.Targets {
		target := lc.Targets[i]
		if target.PageID == page.ID || target.PageID == "" {
			continue
		}
		w.result.Placed = append(w.result.Placed, w.placeTarget(doc, lc, policy, page, target, live[target.PageID]))
	}
	w.result.Findings = append(w.result.Findings, content.Unpublished(doc, lc, live, page.ID)...)
}

func (w *pageRelink) placeTarget(doc *content.Document, lc content.LinkContext, policy template.LinkPolicy,
	page pagemap.Page, target content.LinkTarget, live bool) PlacedLink {
	var (
		placement content.InsertResult
		sentence  string
	)
	if live {
		placement, sentence = backfill(doc, lc, policy, target)
	} else {
		placement = content.InsertTarget(doc, lc, policy, target)
	}

	row := PlacedLink{PageID: target.PageID, Path: target.URL}
	if anchor, inserted := insertedAnchor(placement); inserted {
		row.Outcome, row.Anchor, row.Sentence = OutcomeLinked, anchor, sentence
		w.result.Linked++
		if sentence != "" {
			w.result.Findings = append(w.result.Findings, templatedPageFinding(page, target, sentence))
		}
		return row
	}

	row.Outcome, row.Detail = OutcomeUnchanged, firstDetail(placement)
	if missing, owed := unplaced(placement, target, live, page); owed {
		w.result.Findings = append(w.result.Findings, missing)
	} else if !live && !alreadyLinked(placement) {
		row.Detail = "it is linked once " + target.URL + " is published"
	}
	return row
}

func writeRelinkedPage(ctx context.Context, deps Deps, client *wp.Client, sc *run.StepContext, work pageRelink,
	body siteBody, pages []pagemap.Page) (run.Result, error) {
	linked, err := body.doc.Render()
	if err != nil {
		return run.Result{}, err
	}
	hash, err := body.put(ctx, client, sc.Page, linked)
	if err != nil {
		if errors.IsCode(err, errors.Conflict) {
			return needsHuman("the page " + sc.Page.Path + " changed on the site while it was being relinked: " +
				"it held " + body.raw.ContentHash + " and now holds " + currentHashOf(err)), nil
		}
		return run.Result{}, err
	}
	work.published.ContentHash = hash

	owner, err := deps.Sites.Get(ctx, sc.Run.SiteID)
	if err != nil {
		return run.Result{}, err
	}
	if adoptErr := adopt(ctx, deps, sc.Page, pagemap.NewIndex(pages),
		pagemap.NewSite(owner.BaseURL), body.doc, hash); adoptErr != nil {
		return run.Result{}, adoptErr
	}
	return settleRelinkPage(work)
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
