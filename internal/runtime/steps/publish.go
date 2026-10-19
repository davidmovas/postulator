package steps

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	NamePublish      = string(run.StepPublish)
	ParamRefuseDrift = "refuseDrift"

	CodePublishOverDrift = "publish_over_drift"

	FieldParent = "parent"

	publishTimeout = 2 * time.Minute
	lookupPerPage  = 100
)

type PublishResult struct {
	PreviousMeta        *wp.SEOMeta        `json:"previousMeta,omitempty"`
	PreviousProduct     *ProductSnapshot   `json:"previousProduct,omitempty"`
	URL                 string             `json:"url"`
	Status              string             `json:"status"`
	ContentHash         string             `json:"contentHash"`
	PreviousContent     string             `json:"previousContent"`
	PreviousContentHash string             `json:"previousContentHash"`
	Excerpt             string             `json:"excerpt,omitempty"`
	PreviousExcerpt     *string            `json:"previousExcerpt,omitempty"`
	SEOApplied          []string           `json:"seoApplied"`
	Skipped             []string           `json:"skipped"`
	Findings            []content.Finding  `json:"findings"`
	Mismatches          []pagemap.Mismatch `json:"mismatches"`
	WPID                int64              `json:"wpId"`
	Created             bool               `json:"created"`
}

func Publish(deps Deps) run.StepDef {
	return run.StepDef{
		Name:      NamePublish,
		Preflight: preflights(pluginPreflight(deps, NamePublish, "writes no SEO meta"), storePreflight(deps)),
		Requires:  []run.ArtifactKind{run.ArtifactDraft, run.ArtifactBodyHTML},
		Produces:  []run.ArtifactKind{run.ArtifactPublishResult},
		Retry:     run.RetryPolicy{Max: 3},
		Timeout:   publishTimeout,
		Run: func(ctx context.Context, sc *run.StepContext) (run.Result, error) {
			body, err := sc.Artifact(run.ArtifactBodyHTML)
			if err != nil {
				return run.Result{}, err
			}
			draft, err := draftOf(sc)
			if err != nil {
				return run.Result{}, err
			}
			if sc.Page.WPType == pagemap.WPProduct {
				return publishProduct(ctx, deps, sc, body.Blob, draft)
			}
			return publishItem(ctx, deps, sc, string(body.Blob), draft)
		},
	}
}

func publishItem(ctx context.Context, deps Deps, sc *run.StepContext, rendered string,
	draft content.ContentDraft) (run.Result, error) {
	itemType, err := itemTypeOf(sc.Page)
	if err != nil {
		return run.Result{}, err
	}
	client, err := clientFor(ctx, deps, sc.Run.SiteID)
	if err != nil {
		return run.Result{}, err
	}
	placed, err := parentFor(ctx, deps, sc.Page)
	if err != nil {
		return run.Result{}, err
	}
	if placed.pending {
		return holdForParent(sc, placed), nil
	}
	featured, _, err := decodeArtifact[ImagesResult](sc, run.ArtifactImages)
	if err != nil {
		return run.Result{}, err
	}

	existing, found, err := locate(ctx, client, itemType, sc.Page, placed.wpID)
	if err != nil {
		return run.Result{}, err
	}
	if refusesDrift(sc) {
		return driftRefused(sc.Page, "on the site since it was last published"), nil
	}
	var site liveBody
	if found {
		if site, err = readLiveBody(ctx, client, itemType, existing); err != nil {
			return run.Result{}, err
		}
	}
	final, theirs, part, previous := rendered, "", rendered, site.raw
	if found && sc.Page.WPID == nil && !block.SameRender(site.raw, rendered) {
		return needsHuman(sc.Page.Path + ": " + ReasonSlugTaken + " (" + existing.Link + ")"), nil
	}
	keeping, err := deps.keepingOf(ctx, sc, draft, found)
	if err != nil {
		return run.Result{}, err
	}
	if keeping != nil {
		held, same, guardErr := deps.guard(ctx, sc.Page.ID, keeping, site, rendered)
		if guardErr != nil {
			return run.Result{}, guardErr
		}
		if !same {
			return needsHuman(sc.Page.Path + ": " + ReasonClientChanged), nil
		}
		if held.client && !site.plugin {
			return needsHuman(sc.Page.Path + ": " + ReasonKeptNeedsPlugin), nil
		}
		final, theirs, part, previous = held.final, held.theirs, held.part, held.previous
	}
	live := map[field.Key]string{}
	if found {
		live[field.Title], live[field.Excerpt] = existing.Title, existing.Excerpt
		if seoErr := liveSnippet(ctx, client, itemType, existing.ID, true, live); seoErr != nil {
			return run.Result{}, seoErr
		}
	}
	kept, err := deps.keptFieldsOf(ctx, sc, found, live, field.Title, field.Excerpt, field.SEOTitle, field.SEODescription)
	if err != nil {
		return run.Result{}, err
	}
	viaRaw := keeping != nil && site.plugin
	if viaRaw {
		if _, putErr := client.PutRaw(ctx, itemType, existing.ID, final, site.hash); putErr != nil {
			if errors.IsCode(putErr, errors.Conflict) {
				return needsHuman(sc.Page.Path + ": " + ReasonClientChanged), nil
			}
			return run.Result{}, putErr
		}
		if recordErr := recordOurBlocks(ctx, deps, sc, client, itemType, existing.ID, final, theirs, part); recordErr != nil {
			return run.Result{}, recordErr
		}
	}

	answered, _, err := decodeArtifact[Fields](sc, run.ArtifactFields)
	if err != nil {
		return run.Result{}, err
	}
	asked := itemRequest(sc, draft, final, placed.wpID, featured.FeaturedID)
	asked.existing, asked.found = existing, found
	asked.viaRaw, asked.keepTitle = viaRaw, kept[field.Title]
	if excerpt, written := answered.Value(field.Excerpt); written && !kept[field.Excerpt] {
		asked.excerpt = &excerpt
	}
	written, mismatches, err := writeItem(ctx, client, itemType, sc, asked)
	if err != nil {
		return run.Result{}, err
	}
	if len(mismatches) > 0 {
		return refuseMismatch(sc, mismatches), nil
	}

	result := PublishResult{
		WPID: written.ID, URL: written.Link, Status: written.Status,
		ContentHash: wp.ContentHash(final), Created: !found,
		SEOApplied: make([]string, 0), Skipped: make([]string, 0), Findings: driftFindings(sc.Page),
		Mismatches: mismatches,
	}
	if site.plugin {
		result.PreviousContent, result.PreviousContentHash = previous, wp.ContentHash(previous)
	}
	if asked.excerpt != nil {
		excerpt := existing.Excerpt
		result.Excerpt, result.PreviousExcerpt = *asked.excerpt, &excerpt
	}
	seo, err := applySEO(ctx, client, sc, itemType, written.ID, found, kept)
	if err != nil {
		return run.Result{}, err
	}
	result.took(seo)

	if recordErr := record(ctx, deps, sc, written, result); recordErr != nil {
		return run.Result{}, recordErr
	}
	title := written.Title
	if kept[field.Title] {
		title = ""
	}
	if recordErr := recordWritten(ctx, deps, sc, sc.Page.ID, final, title, seo.applied, kept); recordErr != nil {
		return run.Result{}, recordErr
	}
	if !viaRaw {
		if recordErr := recordOurBlocks(ctx, deps, sc, client, itemType, written.ID, final, theirs, part); recordErr != nil {
			return run.Result{}, recordErr
		}
	}
	if asked.excerpt != nil {
		if recordErr := recordField(ctx, deps, sc, sc.Page.ID, field.Excerpt, *asked.excerpt); recordErr != nil {
			return run.Result{}, recordErr
		}
	}
	return publishedAs(result, verb(found)+" "+sc.Page.Path+" as "+strconv.FormatInt(written.ID, 10))
}

func (d Deps) keepingOf(ctx context.Context, sc *run.StepContext, draft content.ContentDraft, found bool) (*content.KeptBody, error) {
	if !found {
		return nil, nil
	}
	if draft.Kept != nil {
		return draft.Kept, nil
	}
	rewriting, err := d.rewriting(ctx, sc)
	if err != nil || rewriting {
		return nil, err
	}
	return unread(), nil
}

type writeRequest struct {
	existing  wp.Item
	title     string
	content   string
	slug      string
	status    string
	parent    int64
	excerpt   *string
	featured  int64
	found     bool
	created   bool
	viaRaw    bool
	keepTitle bool
}

func itemRequest(sc *run.StepContext, draft content.ContentDraft, rendered string, parent, featured int64) writeRequest {
	return writeRequest{
		title: draft.Title, content: rendered, slug: sc.Page.Slug, status: string(sc.Run.PublishMode),
		parent: parent, featured: featured,
	}
}

func writeItem(ctx context.Context, client *wp.Client, itemType wp.ItemType, sc *run.StepContext,
	asked writeRequest) (wp.Item, []pagemap.Mismatch, error) {
	asked.created = !asked.found
	written, err := upsert(ctx, client, itemType, asked)
	if err != nil {
		return wp.Item{}, nil, err
	}

	mismatches := compare(sc, asked, written)
	if len(mismatches) > 0 {
		asked.existing, asked.found = written, true
		if written, err = upsert(ctx, client, itemType, asked); err != nil {
			return wp.Item{}, nil, err
		}
		mismatches = compare(sc, asked, written)
	}
	return written, mismatches, nil
}

func upsert(ctx context.Context, client *wp.Client, itemType wp.ItemType, req writeRequest) (wp.Item, error) {
	if !req.found {
		in := wp.CreateItem{
			Title: req.title, Content: req.content, Slug: req.slug, Status: req.status,
		}
		if req.excerpt != nil {
			in.Excerpt = *req.excerpt
		}
		if itemType == wp.TypePage {
			in.Parent = &req.parent
		}
		if req.featured != 0 {
			in.FeaturedMedia = &req.featured
		}
		return client.CreateItem(ctx, itemType, in)
	}

	in := wp.UpdateItem{Status: &req.status, Excerpt: req.excerpt}
	if !req.keepTitle {
		in.Title = &req.title
	}
	if !req.viaRaw {
		in.Content = &req.content
	}
	if req.featured != 0 {
		in.FeaturedMedia = &req.featured
	}
	return client.UpdateItem(ctx, itemType, req.existing.ID, in)
}

func compare(sc *run.StepContext, asked writeRequest, written wp.Item) []pagemap.Mismatch {
	checked := sc.Page
	checked.Title = asked.title
	if asked.keepTitle && !asked.created {
		checked.Title = written.Title
	}
	checked.H1 = ""
	checked.Status = statusOf(sc.Run.PublishMode)
	checked.Observed = observedOf(written)
	if !asked.created {
		checked.Slug = ""
		checked.Observed.Link = ""
		return checked.Mismatches()
	}

	checked.Slug = asked.slug
	found := checked.Mismatches()
	if hierarchical(sc.Page) && written.Parent != asked.parent {
		found = append(found, pagemap.Mismatch{
			Field:   FieldParent,
			Planned: strconv.FormatInt(asked.parent, 10),
			Actual:  strconv.FormatInt(written.Parent, 10),
		})
	}
	return found
}

func refuseMismatch(sc *run.StepContext, mismatches []pagemap.Mismatch) run.Result {
	said := make([]string, 0, len(mismatches))
	for i := range mismatches {
		said = append(said, mismatches[i].Field+" was asked for as "+mismatches[i].Planned+
			" and the site answers "+mismatches[i].Actual)
	}
	return needsHuman("the site did not take " + sc.Page.Path + " as it was asked for: " + strings.Join(said, "; "))
}

func refusesDrift(sc *run.StepContext) bool {
	return sc.Page.Drift && sc.BoolParam(ParamRefuseDrift)
}

func driftRefused(page pagemap.Page, since string) run.Result {
	return needsHuman("a human edited " + page.Path + " " + since)
}

func driftFindings(page pagemap.Page) []content.Finding {
	findings := make([]content.Finding, 0, 2)
	if page.Drift {
		findings = append(findings, driftFinding(page))
	}
	return findings
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

func hierarchical(page pagemap.Page) bool {
	return page.WPType == pagemap.WPPage
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

func publishedAs(result PublishResult, message string) (run.Result, error) {
	blob, err := encode(result, "publish result")
	if err != nil {
		return run.Result{}, err
	}
	return run.Result{
		Artifacts: []run.Artifact{{Kind: run.ArtifactPublishResult, Blob: blob}},
		Message:   message,
	}, nil
}
