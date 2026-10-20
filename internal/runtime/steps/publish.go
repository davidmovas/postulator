package steps

import (
	"context"
	"slices"
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
	Fields              []field.Change     `json:"fields"`
	Blocks              []block.Change     `json:"blocks"`
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
			state, err := publishStateOf(sc)
			if err != nil {
				return run.Result{}, err
			}
			if sc.Page.WPType == pagemap.WPProduct {
				return publishProduct(ctx, deps, sc, &state, body.Blob, draft)
			}
			return publishItem(ctx, deps, sc, &state, string(body.Blob), draft)
		},
	}
}

func publishItem(ctx context.Context, deps Deps, sc *run.StepContext, state *publishState, rendered string,
	draft content.ContentDraft) (run.Result, error) {
	itemType, err := itemTypeOf(sc.Page)
	if err != nil {
		return run.Result{}, err
	}
	client, err := clientFor(ctx, deps, sc.Run.SiteID)
	if err != nil {
		return run.Result{}, err
	}
	if state.Plan == nil {
		stop, planErr := planItem(ctx, deps, sc, client, itemType, state, rendered, draft)
		if planErr != nil || stop != nil {
			return orStopped(sc, stop, planErr)
		}
		return reviewThePlan(ctx, deps, sc, state, "planned what to write to "+sc.Page.Path)
	}
	switch state.next() {
	case unitRaw:
		return writeRawBody(ctx, deps, sc, client, itemType, state)
	case unitItem:
		return writeItemFields(ctx, deps, sc, client, itemType, state)
	case unitSEO:
		return writeSEOMeta(ctx, sc, client, itemType, state)
	default:
		return finishItem(ctx, deps, sc, state)
	}
}

const checkpointPublishReviewed = "publish.reviewed"

func orStopped(sc *run.StepContext, stop *run.Result, err error) (run.Result, error) {
	if err != nil {
		return run.Result{}, err
	}
	if !sc.Accepted() {
		return *stop, nil
	}
	cleared := run.NewCheckpoint()
	if setErr := run.Set(cleared, run.CheckpointAccept, ""); setErr != nil {
		return run.Result{}, setErr
	}
	return withChecks(*stop, cleared), nil
}

func reviewThePlan(ctx context.Context, deps Deps, sc *run.StepContext, state *publishState, message string) (run.Result, error) {
	edit, err := deps.planEdit(ctx, sc, state.Plan)
	if err != nil {
		return run.Result{}, err
	}
	state.Result.Fields, state.Result.Blocks = edit.Fields, edit.Blocks
	hash, err := hashOf(state.Plan, "publish plan")
	if err != nil {
		return run.Result{}, err
	}
	reviewed := EditPlan{Pages: []PageEdit{edit}, Hash: hash}
	if state.Plan.Create {
		reviewed.Pages = nil
	}
	held, err := reviewGate(sc, checkpointPublishReviewed, reviewed, "the changes to "+sc.Page.Path)
	if err != nil || held.held != nil {
		return orHeld(held, err)
	}
	waiting, err := state.wait(message)
	if err != nil {
		return run.Result{}, err
	}
	return withChecks(waiting, held.check), nil
}

func orHeld(held gate, err error) (run.Result, error) {
	if err != nil {
		return run.Result{}, err
	}
	return *held.held, nil
}

func (d Deps) planEdit(ctx context.Context, sc *run.StepContext, plan *publishPlan) (PageEdit, error) {
	written, err := d.ourBlocks(ctx, sc.Page.ID)
	if err != nil {
		return PageEdit{}, err
	}
	return PageEdit{
		PageID: sc.Page.ID, Path: sc.Page.Path, WPID: plan.WPID, Fields: plan.fieldChanges(),
		Blocks: block.Diff(plan.Previous, plan.Final, written),
	}, nil
}

func (p *publishPlan) fieldChanges() []field.Change {
	out := make([]field.Change, 0, 5)
	switch {
	case p.Product && p.Store != nil && p.Store.ShortDescription != nil:
		out = append(out, field.NewChange(field.ProductShortDescription, p.Store.Snapshot.ShortDescription, *p.Store.ShortDescription))
	case !p.Product:
		if !p.Request.KeepTitle {
			out = append(out, field.NewChange(field.Title, p.SiteTitle, p.Request.Title))
		}
		if p.Request.Excerpt != nil {
			out = append(out, field.NewChange(field.Excerpt, p.Excerpt, *p.Request.Excerpt))
		}
	}
	if !p.SEO.Send {
		return out
	}
	previous := wp.SEOMeta{}
	if p.SEO.Previous != nil {
		previous = *p.SEO.Previous
	}
	for _, sent := range []struct {
		key           field.Key
		before, after string
	}{
		{key: field.SEOTitle, before: previous.Title, after: p.SEO.Meta.Title},
		{key: field.SEODescription, before: previous.Description, after: p.SEO.Meta.Description},
	} {
		if sent.after != "" {
			out = append(out, field.NewChange(sent.key, sent.before, sent.after))
		}
	}
	return out
}

func planItem(ctx context.Context, deps Deps, sc *run.StepContext, client *wp.Client, itemType wp.ItemType,
	state *publishState, rendered string, draft content.ContentDraft) (*run.Result, error) {
	placed, err := parentFor(ctx, deps, sc.Page)
	if err != nil {
		return nil, err
	}
	if placed.pending {
		held := holdForParent(sc, placed)
		return &held, nil
	}
	featured, _, err := decodeArtifact[ImagesResult](sc, run.ArtifactImages)
	if err != nil {
		return nil, err
	}

	existing, found, err := locate(ctx, client, itemType, sc.Page, placed.wpID)
	if err != nil {
		return nil, err
	}
	if refusesDrift(sc) {
		refused := driftRefused(sc.Page, "on the site since it was last published")
		return &refused, nil
	}
	var site liveBody
	if found {
		if site, err = readLiveBody(ctx, client, itemType, existing); err != nil {
			return nil, err
		}
	}
	final, theirs, part, previous := rendered, "", rendered, site.raw
	if found && sc.Page.WPID == nil && !block.SameRender(site.raw, rendered) {
		taken := needsHuman(sc.Page.Path + ": " + ReasonSlugTaken + " (" + existing.Link + ")")
		return &taken, nil
	}
	keeping, err := deps.keepingOf(ctx, sc, draft, found)
	if err != nil {
		return nil, err
	}
	if keeping != nil {
		held, same, guardErr := deps.guard(ctx, sc.Page.ID, keeping, site, rendered)
		if guardErr != nil {
			return nil, guardErr
		}
		if !same {
			changed := needsHuman(sc.Page.Path + ": " + ReasonClientChanged)
			return &changed, nil
		}
		if held.client && !site.plugin {
			needsPlugin := needsHuman(sc.Page.Path + ": " + ReasonKeptNeedsPlugin)
			return &needsPlugin, nil
		}
		final, theirs, part, previous = held.final, held.theirs, held.part, held.previous
	}
	live := map[field.Key]string{}
	if found {
		live[field.Title], live[field.Excerpt] = existing.Title, existing.Excerpt
		if seoErr := liveSnippet(ctx, client, itemType, existing.ID, true, live); seoErr != nil {
			return nil, seoErr
		}
	}
	kept, err := deps.keptFieldsOf(ctx, sc, found, live, field.Title, field.Excerpt, field.SEOTitle, field.SEODescription)
	if err != nil {
		return nil, err
	}
	answered, _, err := decodeArtifact[Fields](sc, run.ArtifactFields)
	if err != nil {
		return nil, err
	}
	viaRaw := keeping != nil && site.plugin
	request := itemPlan{
		Title: draft.Title, Slug: sc.Page.Slug, Status: string(sc.Run.PublishMode), Parent: placed.wpID,
		Featured: featured.FeaturedID, ViaRaw: viaRaw, KeepTitle: kept[field.Title],
	}
	if excerpt, written := answered.Value(field.Excerpt); written && !kept[field.Excerpt] {
		request.Excerpt = &excerpt
	}
	canonical, err := deps.canonicalWritten(ctx, sc, !found)
	if err != nil {
		return nil, err
	}
	seo, err := planSEO(ctx, client, sc, itemType, existing.ID, found, kept, canonical)
	if err != nil {
		return nil, err
	}

	plan := &publishPlan{
		Writes: make([]string, 0, 3), Kept: kept.keys(), Request: request, SEO: seo, Final: final, Theirs: theirs,
		Part: part, Previous: previous, SiteHash: site.hash, SiteTitle: existing.Title, Excerpt: existing.Excerpt,
		WPID: existing.ID, Create: !found, Plugin: site.plugin,
	}
	if viaRaw {
		plan.Writes = append(plan.Writes, unitRaw)
	}
	if !found || request.carries() {
		plan.Writes = append(plan.Writes, unitItem)
	} else {
		read := answerOf(existing)
		plan.Item = &read
	}
	if seo.Send {
		plan.Writes = append(plan.Writes, unitSEO)
	}
	state.Plan = plan
	state.Result = PublishResult{
		WPID: existing.ID, URL: existing.Link, Status: existing.Status, SEOApplied: make([]string, 0),
		Skipped: make([]string, 0), Findings: driftFindings(sc.Page), Mismatches: make([]pagemap.Mismatch, 0),
	}
	if !seo.Send {
		state.Result.took(seoWrite{applied: []string{}, skipped: seo.Skipped, findings: seo.Findings})
	}
	return nil, nil
}

func writeRawBody(ctx context.Context, deps Deps, sc *run.StepContext, client *wp.Client, itemType wp.ItemType,
	state *publishState) (run.Result, error) {
	plan := state.Plan
	hash, err := client.PutRaw(ctx, itemType, plan.WPID, plan.Final, plan.SiteHash)
	if err != nil {
		if !errors.IsCode(err, errors.Conflict) {
			return state.failed(err)
		}
		landed, readErr := landedRaw(ctx, client, itemType, plan.WPID, plan.Final)
		if readErr != nil {
			return state.failed(readErr)
		}
		if landed == "" {
			return state.stopped(rawConflict(sc, plan, err))
		}
		hash = landed
	}
	if recordErr := recordOurBlocks(ctx, deps, sc, client, itemType, plan.WPID, plan.Final, plan.Theirs, plan.Part); recordErr != nil {
		return state.failed(recordErr)
	}
	state.Result.ContentHash = hash
	if plan.Plugin {
		state.Result.PreviousContent, state.Result.PreviousContentHash = plan.Previous, wp.ContentHash(plan.Previous)
	}
	state.wrote(unitRaw)
	return state.wait("wrote the body of " + sc.Page.Path)
}

func landedRaw(ctx context.Context, client *wp.Client, itemType wp.ItemType, id int64, sent string) (string, error) {
	held, err := client.GetRaw(ctx, itemType, id)
	if err != nil {
		return "", err
	}
	if !block.SameRender(held.Content, sent) {
		return "", nil
	}
	return held.ContentHash, nil
}

func rawConflict(sc *run.StepContext, plan *publishPlan, cause error) run.Result {
	if plan.Product {
		return needsHuman("the product " + sc.Page.Path + " changed in the store while its description was written: " +
			"it held " + plan.SiteHash + " and now holds " + currentHashOf(cause))
	}
	return needsHuman(sc.Page.Path + ": " + ReasonClientChanged)
}

func writeItemFields(ctx context.Context, deps Deps, sc *run.StepContext, client *wp.Client, itemType wp.ItemType,
	state *publishState) (run.Result, error) {
	plan := state.Plan
	asked := plan.request()
	var (
		written    wp.Item
		mismatches []pagemap.Mismatch
	)
	claimed, taken, err := createdEarlier(ctx, client, itemType, sc.Page, plan)
	switch {
	case err != nil:
		return state.failed(err)
	case taken != nil:
		return state.stopped(*taken)
	case claimed != nil:
		asked.created = true
		written, mismatches = *claimed, compare(sc, asked, *claimed)
	default:
		if written, mismatches, err = writeItem(ctx, client, itemType, sc, asked); err != nil {
			return state.failed(err)
		}
	}

	answer := answerOf(written)
	plan.Item = &answer
	state.Result.WPID, state.Result.URL, state.Result.Status = written.ID, written.Link, written.Status
	state.Result.Created, state.Result.Mismatches = plan.Create, mismatches
	if plan.Request.Excerpt != nil {
		excerpt := plan.Excerpt
		state.Result.Excerpt, state.Result.PreviousExcerpt = *plan.Request.Excerpt, &excerpt
	}
	if !plan.Request.ViaRaw {
		state.Result.ContentHash = wp.ContentHash(plan.Final)
		if plan.Plugin {
			state.Result.PreviousContent, state.Result.PreviousContentHash = plan.Previous, wp.ContentHash(plan.Previous)
		}
		if recordErr := recordOurBlocks(ctx, deps, sc, client, itemType, written.ID, plan.Final, plan.Theirs, plan.Part); recordErr != nil {
			return state.failed(recordErr)
		}
	}
	if recordErr := record(ctx, deps, sc, written, state.Result); recordErr != nil {
		return state.failed(recordErr)
	}
	state.wrote(unitItem)
	if len(mismatches) > 0 {
		return state.stopped(refuseMismatch(sc, mismatches))
	}
	return state.wait(verb(!plan.Create) + " " + sc.Page.Path + " as " + strconv.FormatInt(written.ID, 10))
}

func (p *publishPlan) request() writeRequest {
	return writeRequest{
		existing: wp.Item{ID: p.WPID, Excerpt: p.Excerpt}, title: p.Request.Title, content: p.Final,
		slug: p.Request.Slug, status: p.Request.Status, parent: p.Request.Parent, excerpt: p.Request.Excerpt,
		featured: p.Request.Featured, found: !p.Create, viaRaw: p.Request.ViaRaw, keepTitle: p.Request.KeepTitle,
	}
}

func createdEarlier(ctx context.Context, client *wp.Client, itemType wp.ItemType, page pagemap.Page,
	plan *publishPlan) (*wp.Item, *run.Result, error) {
	if !plan.Create {
		return nil, nil, nil
	}
	held, found, err := locate(ctx, client, itemType, page, plan.Request.Parent)
	if err != nil || !found {
		return nil, nil, err
	}
	site, err := readLiveBody(ctx, client, itemType, held)
	if err != nil {
		return nil, nil, err
	}
	if !block.SameRender(site.raw, plan.Final) {
		taken := needsHuman(page.Path + ": " + ReasonSlugTaken + " (" + held.Link + ")")
		return nil, &taken, nil
	}
	return &held, nil, nil
}

func writeSEOMeta(ctx context.Context, sc *run.StepContext, client *wp.Client, itemType wp.ItemType,
	state *publishState) (run.Result, error) {
	plan := state.Plan
	wpID := plan.WPID
	if plan.Item != nil {
		wpID = plan.Item.ID
		if plan.SEO.Canonical {
			plan.SEO.Meta.Canonical = permalinkOf(plan.Item.item())
		}
	}
	seo, err := sendSEO(ctx, client, sc, itemType, wpID, plan.SEO)
	if err != nil {
		return state.failed(err)
	}
	state.Result.took(seo)
	state.wrote(unitSEO)
	return state.wait("wrote the SEO meta of " + sc.Page.Path)
}

func finishItem(ctx context.Context, deps Deps, sc *run.StepContext, state *publishState) (run.Result, error) {
	plan := state.Plan
	written := plan.Item.item()
	if err := record(ctx, deps, sc, written, state.Result); err != nil {
		return state.failed(err)
	}
	kept := plan.kept()
	title := written.Title
	if kept[field.Title] {
		title = ""
	}
	if err := recordWritten(ctx, deps, sc, sc.Page.ID, plan.Final, title, state.Result.SEOApplied, kept); err != nil {
		return state.failed(err)
	}
	if plan.Request.Excerpt != nil {
		if err := recordField(ctx, deps, sc, sc.Page.ID, field.Excerpt, *plan.Request.Excerpt); err != nil {
			return state.failed(err)
		}
	}
	if canonical := plan.SEO.Meta.Canonical; canonical != "" && slices.Contains(state.Result.SEOApplied, "canonical") {
		if err := recordField(ctx, deps, sc, sc.Page.ID, field.SEOCanonical, canonical); err != nil {
			return state.failed(err)
		}
	}
	return state.finish(verb(!plan.Create) + " " + sc.Page.Path + " as " + strconv.FormatInt(written.ID, 10))
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

	in := wp.UpdateItem{Excerpt: req.excerpt}
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
	checked.Status = pagemap.StatusFromWordPress(written.Status)
	if asked.created {
		checked.Status = statusOf(sc.Run.PublishMode)
	}
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
	next.Status = pagemap.StatusFromWordPress(written.Status)
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
