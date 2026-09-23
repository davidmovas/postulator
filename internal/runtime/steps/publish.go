package steps

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	NamePublish      = string(run.StepPublish)
	ParamRefuseDrift = "refuseDrift"

	CapabilitySEOMeta    = "seo_meta"
	CodeSEOMetaSkipped   = "seo_meta_skipped"
	CodePublishOverDrift = "publish_over_drift"

	FieldParent = "parent"

	publishTimeout = 2 * time.Minute
	lookupPerPage  = 100
)

var editableStatuses = []string{"publish", "future", "draft", "pending", "private"}

type PublishResult struct {
	PreviousMeta        *wp.SEOMeta        `json:"previousMeta,omitempty"`
	URL                 string             `json:"url"`
	Status              string             `json:"status"`
	ContentHash         string             `json:"contentHash"`
	PreviousContent     string             `json:"previousContent"`
	PreviousContentHash string             `json:"previousContentHash"`
	SEOApplied          []string           `json:"seoApplied"`
	Skipped             []string           `json:"skipped"`
	Findings            []content.Finding  `json:"findings"`
	Mismatches          []pagemap.Mismatch `json:"mismatches"`
	WPID                int64              `json:"wpId"`
	Created             bool               `json:"created"`
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

			placement, err := parentOf(ctx, deps, sc.Page)
			if err != nil {
				return run.Result{}, err
			}
			if placement.pending {
				return holdForParent(sc, placement), nil
			}
			parent := placement.wpID

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

			replaced, err := bodyBeingReplaced(ctx, client, existing, found)
			if err != nil {
				return run.Result{}, err
			}

			rendered := string(body.Blob)
			status := string(sc.Run.PublishMode)
			asked := writeRequest{
				existing: existing, found: found, title: draft.Title, content: rendered,
				slug: sc.Page.Slug, status: status, parent: parent, featured: featured.FeaturedID,
			}
			written, err := upsert(ctx, client, itemType, asked)
			if err != nil {
				return run.Result{}, err
			}

			mismatches := compare(sc, asked, written)
			if len(mismatches) > 0 {
				asked.existing, asked.found = written, true
				if written, err = upsert(ctx, client, itemType, asked); err != nil {
					return run.Result{}, err
				}
				mismatches = compare(sc, asked, written)
			}
			if len(mismatches) > 0 {
				return refuseMismatch(sc, mismatches), nil
			}

			result := PublishResult{
				WPID: written.ID, URL: written.Link, Status: written.Status,
				ContentHash: wp.ContentHash(rendered), Created: !found,
				PreviousContent: replaced.Content, PreviousContentHash: replaced.ContentHash,
				SEOApplied: make([]string, 0), Skipped: make([]string, 0), Findings: findings,
				Mismatches: mismatches,
			}
			seo, err := applySEO(ctx, client, sc, written.ID, found)
			if err != nil {
				return run.Result{}, err
			}
			result.SEOApplied = seo.applied
			result.Skipped = seo.skipped
			result.PreviousMeta = seo.previous
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

func compare(sc *run.StepContext, asked writeRequest, written wp.Item) []pagemap.Mismatch {
	checked := sc.Page
	checked.Slug = asked.slug
	checked.Title = asked.title
	checked.H1 = ""
	checked.Status = statusOf(sc.Run.PublishMode)
	checked.Observed = observedOf(written)

	found := checked.Mismatches()
	if written.Parent != asked.parent {
		found = append(found, pagemap.Mismatch{
			Field:   FieldParent,
			Planned: strconv.FormatInt(asked.parent, 10),
			Actual:  strconv.FormatInt(written.Parent, 10),
		})
	}
	return found
}

func observedOf(item wp.Item) pagemap.Observed {
	return pagemap.Observed{
		Link: permalinkOf(item), Slug: item.Slug, Status: item.Status, Title: item.Title,
	}
}

func permalinkOf(item wp.Item) string {
	if strings.Contains(item.Link, "?") {
		return ""
	}
	return item.Link
}

func refuseMismatch(sc *run.StepContext, mismatches []pagemap.Mismatch) run.Result {
	said := make([]string, 0, len(mismatches))
	for i := range mismatches {
		said = append(said, mismatches[i].Field+" was asked for as "+mismatches[i].Planned+
			" and the site answers "+mismatches[i].Actual)
	}
	return run.Result{
		Next:    run.TransitionPause,
		Reason:  run.PauseNeedsHuman,
		Message: "the site did not take " + sc.Page.Path + " as it was asked for: " + strings.Join(said, "; "),
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

type placement struct {
	path    string
	wpID    int64
	pending bool
}

func parentOf(ctx context.Context, deps Deps, page pagemap.Page) (placement, error) {
	wanted := pagemap.ParentPath(page.Path)
	if wanted == "" || wanted == "/" {
		return placement{}, nil
	}

	if page.ParentPageID == nil {
		return placement{}, errors.New(errors.Invalid,
			"the page map holds no page at "+wanted+", so "+page.Path+" has no parent to sit under").
			WithDetail("pageId", page.ID).
			WithDetail("path", page.Path).
			WithDetail("parentPath", wanted)
	}

	parent, err := deps.Pages.Get(ctx, *page.ParentPageID)
	if err != nil {
		if errors.IsCode(err, errors.NotFound) {
			return placement{}, errors.New(errors.Invalid,
				"the page names a parent the page map does not hold").
				WithDetail("pageId", page.ID).
				WithDetail("path", page.Path).
				WithDetail("parentPageId", *page.ParentPageID)
		}
		return placement{}, err
	}
	if parent.Path != wanted {
		return placement{}, errors.New(errors.Invalid,
			"the page is linked to "+parent.Path+" while its path asks for "+wanted).
			WithDetail("pageId", page.ID).
			WithDetail("path", page.Path).
			WithDetail("parentPath", wanted).
			WithDetail("linkedPath", parent.Path)
	}
	if parent.WPID == nil {
		return placement{path: parent.Path, pending: true}, nil
	}
	return placement{path: parent.Path, wpID: *parent.WPID}, nil
}

func holdForParent(sc *run.StepContext, parent placement) run.Result {
	return run.Result{
		Next:   run.TransitionPause,
		Reason: run.PauseAwaitingParent,
		Message: sc.Page.Path + " waits for its parent " + parent.path +
			", which is not on the site yet; it goes on by itself once " + parent.path + " is",
	}
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
	return bySlug(listed.Items, page, parent)
}

func bySlug(items []wp.Item, page pagemap.Page, parent int64) (wp.Item, bool, error) {
	elsewhere := make([]wp.Item, 0, len(items))
	for i := range items {
		if items[i].Parent == parent {
			return items[i], true, nil
		}
		elsewhere = append(elsewhere, items[i])
	}

	switch len(elsewhere) {
	case 0:
		return wp.Item{}, false, nil
	case 1:
		return elsewhere[0], true, nil
	default:
		return wp.Item{}, false, errors.New(errors.Conflict,
			"the site carries several pages under that slug and none of them under the wanted parent").
			WithDetail("slug", page.Slug).
			WithDetail("path", page.Path).
			WithDetail("parent", parent)
	}
}

func bodyBeingReplaced(ctx context.Context, client *wp.Client, existing wp.Item, found bool) (wp.RawContent, error) {
	if !found {
		return wp.RawContent{}, nil
	}

	raw, err := client.GetRaw(ctx, existing.ID)
	switch {
	case err == nil:
		return raw, nil
	case wp.IsPluginMissing(err), errors.IsCode(err, errors.NotFound):
		return wp.RawContent{}, nil
	default:
		return wp.RawContent{}, err
	}
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
	previous *wp.SEOMeta
	applied  []string
	skipped  []string
	findings []content.Finding
}

func applySEO(ctx context.Context, client *wp.Client, sc *run.StepContext, wpID int64, updating bool) (seoWrite, error) {
	meta, found, err := decodeArtifact[Meta](sc, run.ArtifactMeta)
	if err != nil {
		return seoWrite{}, err
	}
	if !found {
		if artifact, held := sc.Artifacts[run.ArtifactMeta]; held && artifact.Purged {
			return metaPurged(sc.Page), nil
		}
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

	previous, err := metaBeingReplaced(ctx, client, capabilities, wpID, updating)
	if err != nil {
		return seoWrite{}, err
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
		previous: previous,
		applied:  append(make([]string, 0, len(result.Applied)), result.Applied...),
		skipped:  []string{},
		findings: nil,
	}, nil
}

func metaBeingReplaced(ctx context.Context, client *wp.Client, capabilities wp.Capabilities,
	wpID int64, updating bool) (*wp.SEOMeta, error) {
	if !updating || !capabilities.Has(wp.CapabilitySEOMetaRead) {
		return nil, nil
	}

	held, err := client.GetSEOMeta(ctx, wpID)
	switch {
	case err == nil:
		return &held, nil
	case wp.IsPluginMissing(err), wp.IsPluginOutdated(err), errors.IsCode(err, errors.NotFound):
		return nil, nil
	default:
		return nil, err
	}
}

func noMetaGenerated() seoWrite {
	return seoWrite{applied: []string{}, skipped: []string{CodeSEOMetaSkipped}}
}

func metaPurged(page pagemap.Page) seoWrite {
	return seoWrite{
		applied: []string{},
		skipped: []string{CodeSEOMetaSkipped},
		findings: []content.Finding{{
			Severity: content.SeverityWarn,
			Code:     CodeArtifactPurged,
			Message: "the meta of " + page.Path + " was dropped by retention before it could be written, " +
				"so the search snippet on the site is whatever was there before",
			Details: map[string]any{
				"pageId": page.ID, "path": page.Path, "kind": string(run.ArtifactMeta),
			},
		}},
	}
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
	next.Observed = observedOf(written)
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
