package steps

import (
	"context"
	"slices"
	"strconv"
	"strings"

	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/site"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	CodeProductNameDiffers        = "product_name_differs"
	CodeProductTypeKept           = "product_type_kept"
	CodeCommerceUnknown           = "commerce_unknown"
	CodeCommerceAbsent            = "commerce_absent"
	CodeCommerceForbidden         = "commerce_forbidden"
	CodeProductNeedsPlugin        = "product_needs_plugin"
	CodeProductNotInStore         = "product_not_in_store"
	CodeProductEditedLive         = "product_edited_live"
	CodeProductCategoryUnwritable = "product_category_unwritable"
	CodeProductOutputsMissing     = "product_outputs_missing"
	CodeProductOutputsIgnored     = "product_outputs_ignored"

	FieldShortDescription = "shortDescription"
	FieldAttributes       = "attributes"

	simpleProduct = "simple"
)

type SnapshotAttribute struct {
	Name      string   `json:"name"`
	Options   []string `json:"options"`
	ID        int64    `json:"id"`
	Position  int      `json:"position"`
	Visible   bool     `json:"visible"`
	Variation bool     `json:"variation"`
}

type ProductSnapshot struct {
	ShortDescription string              `json:"shortDescription"`
	WrittenShort     string              `json:"writtenShort"`
	Attributes       []SnapshotAttribute `json:"attributes"`
	Written          []SnapshotAttribute `json:"written"`
	Images           []int64             `json:"images"`
	Added            []string            `json:"added"`
	ImageID          int64               `json:"imageId"`
	ShortWritten     bool                `json:"shortWritten"`
	AttributesSent   bool                `json:"attributesSent"`
}

func (s *ProductSnapshot) wrote(written wp.Product) {
	if s.ShortWritten {
		s.WrittenShort = written.ShortDescription
	}
	for _, name := range s.Added {
		if at := attributeNamed(written.Attributes, name); at >= 0 {
			s.Written = append(s.Written, snapshotOf(written.Attributes[at:at+1])...)
		}
	}
}

func attributeNamed(attributes []wp.ProductAttribute, name string) int {
	return slices.IndexFunc(attributes, func(attribute wp.ProductAttribute) bool {
		return strings.EqualFold(strings.TrimSpace(attribute.Name), strings.TrimSpace(name))
	})
}

func snapshotOf(attributes []wp.ProductAttribute) []SnapshotAttribute {
	out := make([]SnapshotAttribute, 0, len(attributes))
	for _, attribute := range attributes {
		out = append(out, SnapshotAttribute{
			Name: attribute.Name, Options: slices.Clone(attribute.Options), ID: attribute.ID,
			Position: attribute.Position, Visible: attribute.Visible, Variation: attribute.Variation,
		})
	}
	return out
}

func (s ProductSnapshot) StoreAttributes() []wp.ProductAttribute {
	out := make([]wp.ProductAttribute, 0, len(s.Attributes))
	for _, attribute := range s.Attributes {
		out = append(out, wp.ProductAttribute{
			Name: attribute.Name, Options: slices.Clone(attribute.Options), ID: attribute.ID,
			Position: attribute.Position, Visible: attribute.Visible, Variation: attribute.Variation,
		})
	}
	return out
}

func imageIDs(images []wp.ProductImage) []int64 {
	out := make([]int64, 0, len(images))
	for _, image := range images {
		out = append(out, image.ID)
	}
	return out
}

func mergedAttributes(held []wp.ProductAttribute, wanted []content.Specification) (merged []wp.ProductAttribute, added []string) {
	merged = make([]wp.ProductAttribute, 0, len(held)+len(wanted))
	for _, attribute := range held {
		attribute.Options = slices.Clone(attribute.Options)
		merged = append(merged, attribute)
	}
	added = make([]string, 0, len(wanted))
	for _, specification := range wanted {
		at := attributeNamed(merged, specification.Name)
		switch {
		case at < 0:
			merged = append(merged, wp.ProductAttribute{
				Name: specification.Name, Options: []string{specification.Value}, Position: len(merged), Visible: true,
			})
		case filled(merged[at]) || merged[at].ID != 0 || merged[at].Variation:
			continue
		default:
			merged[at].Options = []string{specification.Value}
		}
		added = append(added, specification.Name)
	}
	return merged, added
}

func filled(attribute wp.ProductAttribute) bool {
	return slices.ContainsFunc(attribute.Options, func(option string) bool { return strings.TrimSpace(option) != "" })
}

func publishProduct(ctx context.Context, deps Deps, sc *run.StepContext, state *publishState, body []byte,
	draft content.ContentDraft) (run.Result, error) {
	if sc.Page.WPID == nil {
		return needsHuman(sc.Page.Path + " " + notInStore), nil
	}
	client, err := clientFor(ctx, deps, sc.Run.SiteID)
	if err != nil {
		return run.Result{}, err
	}
	if state.Plan == nil {
		stop, planErr := planProduct(ctx, deps, sc, client, state, body, draft)
		if planErr != nil || stop != nil {
			return orStopped(stop, planErr)
		}
		return state.wait("planned what to write to the product " + sc.Page.Path)
	}
	switch state.next() {
	case unitStore:
		return writeStoreFields(ctx, deps, sc, client, state, draft)
	case unitRaw:
		return writeRawBody(ctx, deps, sc, client, wp.TypeProduct, state)
	case unitSEO:
		return writeSEOMeta(ctx, sc, client, wp.TypeProduct, state)
	default:
		return finishProduct(ctx, deps, sc, state)
	}
}

func planProduct(ctx context.Context, deps Deps, sc *run.StepContext, client *wp.Client, state *publishState,
	body []byte, draft content.ContentDraft) (*run.Result, error) {
	held, err := client.GetProduct(ctx, *sc.Page.WPID)
	if err != nil {
		refused, refusedErr := refusedByStore(ctx, deps, sc, err)
		if refusedErr != nil {
			return nil, refusedErr
		}
		return &refused, nil
	}
	if refusesDrift(sc) {
		refused := driftRefused(sc.Page, "in the store since it was last written")
		return &refused, nil
	}
	description, err := descriptionOf(body)
	if err != nil {
		return nil, err
	}
	raw, err := client.GetRaw(ctx, wp.TypeProduct, held.ID)
	switch {
	case wp.IsPluginMissing(err):
		missing := needsHuman(sc.Page.Path + " is a product and the site has no companion plugin to write its description")
		return &missing, nil
	case err != nil:
		return nil, err
	}
	featured, _, err := decodeArtifact[ImagesResult](sc, run.ArtifactImages)
	if err != nil {
		return nil, err
	}
	theirs, part, previous := "", description, raw.Content
	keeping, err := deps.keepingOf(ctx, sc, draft, true)
	if err != nil {
		return nil, err
	}
	if keeping != nil {
		onSite := liveBody{raw: raw.Content, hash: raw.ContentHash, plugin: true}
		guarded, same, guardErr := deps.guard(ctx, sc.Page.ID, keeping, onSite, description)
		if guardErr != nil {
			return nil, guardErr
		}
		if !same {
			changed := needsHuman(sc.Page.Path + ": " + ReasonClientChanged)
			return &changed, nil
		}
		description, theirs, part, previous = guarded.final, guarded.theirs, guarded.part, guarded.previous
	}
	live := map[field.Key]string{field.ProductShortDescription: held.ShortDescription}
	if seoErr := liveSnippet(ctx, client, wp.TypeProduct, held.ID, true, live); seoErr != nil {
		return nil, seoErr
	}
	kept, err := deps.keptFieldsOf(ctx, sc, true, live, field.SEOTitle, field.SEODescription, field.ProductShortDescription)
	if err != nil {
		return nil, err
	}
	written := draft.Product
	if written != nil && kept[field.ProductShortDescription] {
		copied := *written
		copied.ShortDescription = ""
		written = &copied
	}
	update, snapshot, typed := productUpdate(held, written, featured.FeaturedID)
	seo, err := planSEO(ctx, client, sc, wp.TypeProduct, held.ID, true, kept)
	if err != nil {
		return nil, err
	}

	saved := productAnswerOf(held)
	plan := &publishPlan{
		Writes: make([]string, 0, 3), Kept: kept.keys(), SEO: seo, Final: description, Theirs: theirs, Part: part,
		Previous: previous, SiteHash: raw.ContentHash, WPID: held.ID, Plugin: true, Product: true,
		Store: storePlanOf(update, snapshot, &saved),
	}
	if hasFields(update) {
		plan.Writes = append(plan.Writes, unitStore)
	}
	plan.Writes = append(plan.Writes, unitRaw)
	if seo.Send {
		plan.Writes = append(plan.Writes, unitSEO)
	}
	state.Plan = plan
	state.Result = PublishResult{
		WPID: held.ID, URL: permalinkOf(wp.Item{Link: held.Permalink}), Status: held.Status, PreviousProduct: &snapshot,
		SEOApplied: make([]string, 0), Skipped: make([]string, 0), Findings: findingsOfProduct(sc.Page, held, typed),
		Mismatches: make([]pagemap.Mismatch, 0),
	}
	if !seo.Send {
		state.Result.took(seoWrite{applied: []string{}, skipped: seo.Skipped, findings: seo.Findings})
	}
	return nil, nil
}

func storePlanOf(update wp.UpdateProduct, snapshot ProductSnapshot, saved *productAnswer) *storePlan {
	plan := &storePlan{ShortDescription: update.ShortDescription, Snapshot: snapshot, Saved: saved}
	if update.Attributes != nil {
		plan.Attributes, plan.AttributesSent = snapshotOf(*update.Attributes), true
	}
	if update.Images != nil {
		plan.Images, plan.ImagesSent = slices.Clone(*update.Images), true
	}
	return plan
}

func writeStoreFields(ctx context.Context, deps Deps, sc *run.StepContext, client *wp.Client, state *publishState,
	draft content.ContentDraft) (run.Result, error) {
	plan := state.Plan
	store := productWrite{deps: deps, sc: sc, client: client, held: wp.Product{ID: plan.WPID}, draft: draft.Product}
	snapshot := plan.Store.Snapshot
	saved, refused, err := store.saveFields(ctx, plan.Store.update(), &snapshot, plan.SiteHash)
	if err != nil {
		return state.failed(err)
	}
	if refused != nil {
		return state.stopped(*refused)
	}
	answer := productAnswerOf(saved.written)
	plan.Store.Snapshot, plan.Store.Saved, plan.SiteHash = snapshot, &answer, saved.expected
	state.Result.PreviousProduct = &snapshot
	state.Result.URL, state.Result.Status = permalinkOf(wp.Item{Link: answer.Permalink}), answer.Status
	state.wrote(unitStore)
	return state.wait("wrote the store's fields of " + sc.Page.Path)
}

func finishProduct(ctx context.Context, deps Deps, sc *run.StepContext, state *publishState) (run.Result, error) {
	plan := state.Plan
	stored := plan.Store.Saved.product()
	if err := recordProduct(ctx, deps, sc, stored, state.Result.ContentHash); err != nil {
		return state.failed(err)
	}
	if err := recordBody(ctx, deps, sc.Page.ID, plan.Final); err != nil {
		return state.failed(err)
	}
	recorded := map[field.Key]string{field.ProductDescription: plan.Final}
	if !plan.kept()[field.ProductShortDescription] {
		recorded[field.ProductShortDescription] = stored.ShortDescription
	}
	for key, value := range recorded {
		if err := recordField(ctx, deps, sc, sc.Page.ID, key, value); err != nil {
			return state.failed(err)
		}
	}
	return state.finish("updated the product " + sc.Page.Path + " as " + strconv.FormatInt(plan.WPID, 10))
}

func descriptionOf(body []byte) (string, error) {
	doc, err := content.Parse(string(body))
	if err != nil {
		return "", err
	}
	return doc.RenderWithoutHeadingOne()
}

func findingsOfProduct(page pagemap.Page, held wp.Product, typed *content.Finding) []content.Finding {
	findings := driftFindings(page)
	if typed != nil {
		findings = append(findings, *typed)
	}
	if differs := nameDiffers(page, held); differs != nil {
		findings = append(findings, *differs)
	}
	return findings
}

type productWrite struct {
	deps   Deps
	sc     *run.StepContext
	client *wp.Client
	draft  *content.ProductDraft
	held   wp.Product
}

type productSaved struct {
	written  wp.Product
	expected string
}

func (w productWrite) saveFields(ctx context.Context, update wp.UpdateProduct, snapshot *ProductSnapshot,
	expected string) (productSaved, *run.Result, error) {
	if !hasFields(update) {
		return productSaved{written: w.held, expected: expected}, nil, nil
	}

	written, err := w.client.UpdateProduct(ctx, w.held.ID, update)
	if err != nil {
		return w.refusedBy(ctx, err)
	}
	if mismatches := productKept(*snapshot, w.draft, written); len(mismatches) > 0 {
		if written, err = w.client.UpdateProduct(ctx, w.held.ID, update); err != nil {
			return w.refusedBy(ctx, err)
		}
		if mismatches = productKept(*snapshot, w.draft, written); len(mismatches) > 0 {
			refused := refuseMismatch(w.sc, mismatches)
			return productSaved{}, &refused, nil
		}
	}
	snapshot.wrote(written)

	resaved, err := w.client.GetRaw(ctx, wp.TypeProduct, w.held.ID)
	if err != nil {
		return productSaved{}, nil, err
	}
	return productSaved{written: written, expected: resaved.ContentHash}, nil, nil
}

func (w productWrite) refusedBy(ctx context.Context, cause error) (productSaved, *run.Result, error) {
	refused, err := refusedByStore(ctx, w.deps, w.sc, cause)
	if err != nil {
		return productSaved{}, nil, err
	}
	return productSaved{}, &refused, nil
}

func productUpdate(held wp.Product, draft *content.ProductDraft, featured int64) (wp.UpdateProduct, ProductSnapshot, *content.Finding) {
	snapshot := ProductSnapshot{
		ShortDescription: held.ShortDescription, Attributes: snapshotOf(held.Attributes),
		Written: make([]SnapshotAttribute, 0), Images: imageIDs(held.Images), Added: make([]string, 0),
	}
	var (
		update wp.UpdateProduct
		typed  *content.Finding
	)
	if draft != nil && draft.ShortDescription != "" {
		short := draft.ShortDescription
		update.ShortDescription = &short
		snapshot.ShortWritten = true
	}
	if draft != nil && len(draft.Specifications) > 0 {
		if held.Type != "" && held.Type != simpleProduct {
			typed = &content.Finding{
				Severity: content.SeverityWarn, Code: CodeProductTypeKept,
				Message: "the product is a " + held.Type + " product, so its attributes stay as the store keeps them",
				Details: map[string]any{"type": held.Type},
			}
		} else if merged, added := mergedAttributes(held.Attributes, draft.Specifications); len(added) > 0 {
			update.Attributes = &merged
			snapshot.Added = added
			snapshot.AttributesSent = true
		}
	}
	if featured != 0 && len(held.Images) == 0 {
		images := []int64{featured}
		update.Images = &images
		snapshot.ImageID = featured
	}
	return update, snapshot, typed
}

func hasFields(update wp.UpdateProduct) bool {
	return update.ShortDescription != nil || update.Attributes != nil || update.Images != nil
}

func productKept(snapshot ProductSnapshot, draft *content.ProductDraft, written wp.Product) []pagemap.Mismatch {
	found := make([]pagemap.Mismatch, 0)
	if snapshot.ShortWritten && strings.TrimSpace(written.ShortDescription) == "" {
		found = append(found, pagemap.Mismatch{Field: FieldShortDescription, Planned: "a short description", Actual: "none"})
	}
	if draft == nil {
		return found
	}
	for _, name := range snapshot.Added {
		want := ""
		for _, specification := range draft.Specifications {
			if specification.Name == name {
				want = specification.Value
			}
		}
		at := attributeNamed(written.Attributes, name)
		if at < 0 || !slices.Contains(written.Attributes[at].Options, want) {
			found = append(found, pagemap.Mismatch{Field: FieldAttributes, Planned: name + ": " + want, Actual: "missing"})
		}
	}
	return found
}

func nameDiffers(page pagemap.Page, held wp.Product) *content.Finding {
	planned := strings.TrimSpace(page.H1)
	name := content.StoreName(pagemap.Page{Observed: pagemap.Observed{Title: held.Name}})
	if planned == "" || name == "" || strings.EqualFold(planned, name) {
		return nil
	}
	return &content.Finding{
		Severity: content.SeverityWarn, Code: CodeProductNameDiffers,
		Message: "the file names " + page.Path + " " + planned + " and the store calls it " + name +
			"; the name stays as the store has it, so rename the product in WooCommerce if the file is right",
		Details: map[string]any{"pageId": page.ID, "path": page.Path, "planned": planned, "store": name},
	}
}

const notInStore = "is a product the store does not hold yet; create it in WooCommerce and sync the site"

func refusedByStore(ctx context.Context, deps Deps, sc *run.StepContext, err error) (run.Result, error) {
	commerce := site.CommerceUnknown
	message := ""
	switch {
	case wp.StoreAbsent(err):
		commerce = site.CommerceAbsent
		message = "the site answers no WooCommerce store, so the product " + sc.Page.Path + " cannot be written"
	case wp.Forbidden(err):
		commerce = site.CommerceForbidden
		message = "the WordPress user may not edit products, so the product " + sc.Page.Path + " cannot be written"
	case errors.IsCode(err, errors.NotFound):
		message = "the product " + sc.Page.Path + " is no longer in the store; sync the site"
	default:
		return run.Result{}, err
	}

	if commerce != site.CommerceUnknown {
		if keepErr := keepCommerce(ctx, deps, sc.Run.SiteID, commerce); keepErr != nil {
			return run.Result{}, keepErr
		}
	}
	return needsHuman(message), nil
}

func keepCommerce(ctx context.Context, deps Deps, siteID string, commerce site.Commerce) error {
	if deps.SiteWriter == nil {
		return nil
	}
	owner, err := deps.Sites.Get(ctx, siteID)
	if err != nil {
		return err
	}
	if owner.Commerce == commerce {
		return nil
	}
	owner.Commerce = commerce
	owner.UpdatedAt = deps.now()
	return deps.SiteWriter.Update(ctx, owner)
}

func recordProduct(ctx context.Context, deps Deps, sc *run.StepContext, written wp.Product, hash string) error {
	now := deps.now()
	next := sc.Page
	next.WPID = &written.ID
	next.Status = pagemap.StatusFromWordPress(written.Status)
	next.ContentHash = hash
	next.Observed = pagemap.Observed{
		Link: permalinkOf(wp.Item{Link: written.Permalink}), Slug: written.Slug, Status: written.Status, Title: written.Name,
	}
	next.Drift = false
	next.LastSyncedAt = &now
	next.UpdatedAt = now
	if !written.Modified.IsZero() {
		modified := written.Modified.UTC()
		next.WPModifiedAt = &modified
	}
	return deps.Pages.Update(ctx, next)
}
