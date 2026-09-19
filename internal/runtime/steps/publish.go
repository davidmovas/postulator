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
	NamePublish      = "publish"
	ParamRefuseDrift = "refuseDrift"

	CapabilitySEOMeta    = "seo_meta"
	CodeSEOMetaSkipped   = "seo_meta_skipped"
	CodePublishOverDrift = "publish_over_drift"

	publishTimeout = 2 * time.Minute
	lookupPerPage  = 100
)

var editableStatuses = []string{"publish", "future", "draft", "pending", "private"}

type PublishResult struct {
	URL         string            `json:"url"`
	Status      string            `json:"status"`
	ContentHash string            `json:"contentHash"`
	SEOApplied  []string          `json:"seoApplied"`
	Skipped     []string          `json:"skipped"`
	Findings    []content.Finding `json:"findings"`
	WPID        int64             `json:"wpId"`
	Created     bool              `json:"created"`
}

func Publish(deps Deps) run.StepDef {
	return run.StepDef{
		Name:     NamePublish,
		Requires: []run.ArtifactKind{run.ArtifactDraft, run.ArtifactBodyHTML},
		Produces: []run.ArtifactKind{run.ArtifactPublishResult},
		Retry:    run.RetryPolicy{Max: 3},
		Timeout:  publishTimeout,
		Run: func(ctx context.Context, sc *run.StepContext) (run.Result, error) {
			body, err := sc.Artifact(run.ArtifactBodyHTML)
			if err != nil {
				return run.Result{}, err
			}
			draft, err := draftOf(sc)
			if err != nil {
				return run.Result{}, err
			}
			itemType, err := itemTypeOf(sc.Page)
			if err != nil {
				return run.Result{}, err
			}
			client, err := clientFor(ctx, deps, sc.Run.SiteID)
			if err != nil {
				return run.Result{}, err
			}

			parent, err := parentWPID(ctx, deps, sc.Page)
			if err != nil {
				return run.Result{}, err
			}
			featured, _, err := decodeArtifact[ImagesResult](sc, run.ArtifactImages)
			if err != nil {
				return run.Result{}, err
			}

			existing, found, err := locate(ctx, client, itemType, sc.Page, parent)
			if err != nil {
				return run.Result{}, err
			}

			findings := make([]content.Finding, 0, 1)
			if sc.Page.Drift {
				if sc.BoolParam(ParamRefuseDrift) {
					return run.Result{
						Next:    run.TransitionPause,
						Reason:  run.PauseNeedsHuman,
						Message: "a human edited " + sc.Page.Path + " on the site since it was last published",
					}, nil
				}
				findings = append(findings, driftFinding(sc.Page))
			}

			rendered := string(body.Blob)
			status := string(sc.Run.PublishMode)
			written, err := upsert(ctx, client, itemType, writeRequest{
				existing: existing, found: found, title: draft.Title, content: rendered,
				slug: sc.Page.Slug, status: status, parent: parent, featured: featured.FeaturedID,
			})
			if err != nil {
				return run.Result{}, err
			}

			result := PublishResult{
				WPID: written.ID, URL: written.Link, Status: written.Status,
				ContentHash: wp.ContentHash(rendered), Created: !found,
				SEOApplied: make([]string, 0), Skipped: make([]string, 0), Findings: findings,
			}
			seo, err := applySEO(ctx, client, sc, written.ID)
			if err != nil {
				return run.Result{}, err
			}
			result.SEOApplied = seo.applied
			result.Skipped = seo.skipped
			result.Findings = append(result.Findings, seo.findings...)

			if recordErr := record(ctx, deps, sc, written, result); recordErr != nil {
				return run.Result{}, recordErr
			}

			blob, err := encode(result, "publish result")
			if err != nil {
				return run.Result{}, err
			}
			return run.Result{
				Artifacts: []run.Artifact{{Kind: run.ArtifactPublishResult, Blob: blob}},
				Message:   verb(found) + " " + sc.Page.Path + " as " + strconv.FormatInt(written.ID, 10),
			}, nil
		},
	}
}

func driftFinding(page pagemap.Page) content.Finding {
	return content.Finding{
		Severity: content.SeverityWarn,
		Code:     CodePublishOverDrift,
		Message:  "the page " + page.Path + " changed on the site since it was last published",
		Details:  map[string]any{"class": ClassNeedsHuman, "pageId": page.ID, "path": page.Path},
	}
}

func verb(found bool) string {
	if found {
		return "updated"
	}
	return "created"
}

func itemTypeOf(page pagemap.Page) (wp.ItemType, error) {
	switch page.WPType {
	case pagemap.WPPage:
		return wp.TypePage, nil
	case pagemap.WPPost:
		return wp.TypePost, nil
	default:
		return "", errors.New(errors.Invalid, "only pages and posts are written by the publish step").
			WithDetail("wpType", string(page.WPType)).WithDetail("pageId", page.ID)
	}
}

func clientFor(ctx context.Context, deps Deps, siteID string) (*wp.Client, error) {
	if deps.WordPress == nil {
		return nil, errors.New(errors.Invalid, "no WordPress client is configured for this site").
			WithDetail("siteId", siteID)
	}
	return deps.WordPress.Client(ctx, siteID)
}

func parentWPID(ctx context.Context, deps Deps, page pagemap.Page) (int64, error) {
	if page.ParentPageID == nil {
		return 0, nil
	}

	parent, err := deps.Pages.Get(ctx, *page.ParentPageID)
	if err != nil {
		if errors.IsCode(err, errors.NotFound) {
			return 0, nil
		}
		return 0, err
	}
	if parent.WPID == nil {
		return 0, nil
	}
	return *parent.WPID, nil
}

func locate(ctx context.Context, client *wp.Client, itemType wp.ItemType, page pagemap.Page, parent int64) (wp.Item, bool, error) {
	if page.WPID != nil {
		item, err := client.GetItem(ctx, itemType, *page.WPID)
		if err == nil {
			return item, true, nil
		}
		if !errors.IsCode(err, errors.NotFound) {
			return wp.Item{}, false, err
		}
	}
	if page.Slug == "" {
		return wp.Item{}, false, nil
	}

	listed, err := client.ListItems(ctx, itemType, wp.ListQuery{
		Slug: page.Slug, Status: editableStatuses, PerPage: lookupPerPage,
	})
	if err != nil {
		return wp.Item{}, false, err
	}
	for i := range listed.Items {
		if listed.Items[i].Parent == parent {
			return listed.Items[i], true, nil
		}
	}
	return wp.Item{}, false, nil
}

type writeRequest struct {
	existing wp.Item
	title    string
	content  string
	slug     string
	status   string
	parent   int64
	featured int64
	found    bool
}

func upsert(ctx context.Context, client *wp.Client, itemType wp.ItemType, req writeRequest) (wp.Item, error) {
	if !req.found {
		in := wp.CreateItem{
			Title: req.title, Content: req.content, Slug: req.slug, Status: req.status,
			Parent: &req.parent,
		}
		if req.featured != 0 {
			in.FeaturedMedia = &req.featured
		}
		return client.CreateItem(ctx, itemType, in)
	}

	in := wp.UpdateItem{
		Title: &req.title, Content: &req.content, Slug: &req.slug, Status: &req.status,
		Parent: &req.parent,
	}
	if req.featured != 0 {
		in.FeaturedMedia = &req.featured
	}
	return client.UpdateItem(ctx, itemType, req.existing.ID, in)
}

type seoWrite struct {
	applied  []string
	skipped  []string
	findings []content.Finding
}

func applySEO(ctx context.Context, client *wp.Client, sc *run.StepContext, wpID int64) (seoWrite, error) {
	meta, found, err := decodeArtifact[Meta](sc, run.ArtifactMeta)
	if err != nil {
		return seoWrite{}, err
	}
	if !found {
		return noMetaGenerated(), nil
	}

	capabilities, err := client.Capabilities(ctx)
	if err != nil {
		if wp.IsPluginMissing(err) {
			return metaNotWritten(sc.Page, ReasonNoPlugin), nil
		}
		return seoWrite{}, err
	}
	if !capabilities.Has(CapabilitySEOMeta) {
		return metaNotWritten(sc.Page, ReasonNoSEOWriter), nil
	}

	result, err := client.SetSEOMeta(ctx, wpID, wp.SEOMeta{
		Title:         meta.Title,
		Description:   meta.Description,
		Canonical:     meta.Canonical,
		OGTitle:       meta.OGTitle,
		OGDescription: meta.OGDescription,
	})
	if err != nil {
		if wp.IsPluginMissing(err) {
			return metaNotWritten(sc.Page, ReasonNoPlugin), nil
		}
		return seoWrite{}, err
	}
	return seoWrite{
		applied:  append(make([]string, 0, len(result.Applied)), result.Applied...),
		skipped:  []string{},
		findings: nil,
	}, nil
}

func noMetaGenerated() seoWrite {
	return seoWrite{applied: []string{}, skipped: []string{CodeSEOMetaSkipped}}
}

func metaNotWritten(page pagemap.Page, reason string) seoWrite {
	return seoWrite{
		applied: []string{},
		skipped: []string{CodeSEOMetaSkipped},
		findings: []content.Finding{{
			Severity: content.SeverityWarn,
			Code:     CodeSEOMetaSkipped,
			Message:  "the SEO meta of " + page.Path + " was generated but not written: " + reason,
			Details:  map[string]any{"pageId": page.ID, "path": page.Path, "reason": reason},
		}},
	}
}

func record(ctx context.Context, deps Deps, sc *run.StepContext, written wp.Item, result PublishResult) error {
	now := deps.now()
	next := sc.Page
	next.WPID = &written.ID
	next.Status = statusOf(sc.Run.PublishMode)
	next.ContentHash = result.ContentHash
	next.Drift = false
	next.LastSyncedAt = &now
	next.UpdatedAt = now
	if !written.Modified.IsZero() {
		modified := written.Modified.UTC()
		next.WPModifiedAt = &modified
	}
	return deps.Pages.Update(ctx, next)
}

func statusOf(mode run.PublishMode) pagemap.Status {
	if mode == run.PublishLive {
		return pagemap.StatusPublished
	}
	return pagemap.StatusExists
}

func (d Deps) now() time.Time {
	if d.Clock == nil {
		return time.Time{}
	}
	return d.Clock.Now().UTC().Truncate(time.Second)
}
