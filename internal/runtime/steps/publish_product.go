package steps

import (
	"context"
	"slices"
	"strconv"
	"strings"

	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/domain/content"
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

func publishProduct(ctx context.Context, deps Deps, sc *run.StepContext, body []byte,
	draft content.ContentDraft) (run.Result, error) {
	if sc.Page.WPID == nil {
		return needsHuman(sc.Page.Path + " " + notInStore), nil
	}
	client, err := clientFor(ctx, deps, sc.Run.SiteID)
	if err != nil {
		return run.Result{}, err
	}
	held, err := client.GetProduct(ctx, *sc.Page.WPID)
	if err != nil {
		return refusedByStore(ctx, deps, sc, err)
	}
	if refusesDrift(sc) {
		return driftRefused(sc.Page, "in the store since it was last written"), nil
	}
	description, err := descriptionOf(body)
	if err != nil {
		return run.Result{}, err
	}
	raw, err := client.GetRaw(ctx, wp.TypeProduct, held.ID)
	switch {
	case wp.IsPluginMissing(err):
		return needsHuman(sc.Page.Path + " is a product and the site has no companion plugin to write its description"), nil
	case err != nil:
		return run.Result{}, err
	}
	featured, _, err := decodeArtifact[ImagesResult](sc, run.ArtifactImages)
	if err != nil {
		return run.Result{}, err
	}
	categories, err := ensureCategories(ctx, deps, client, sc, held.Categories, false)
	if err != nil {
		return run.Result{}, err
	}

	update, snapshot, typed := productUpdate(held, draft.Product, featured.FeaturedID, categories.send)
	store := productWrite{deps: deps, sc: sc, client: client, held: held, draft: draft.Product}
	saved, refused, err := store.saveFields(ctx, update, &snapshot, raw.ContentHash)
	if err != nil {
		return run.Result{}, err
	}
	if refused != nil {
		return *refused, nil
	}
	hash, err := client.PutRaw(ctx, wp.TypeProduct, held.ID, description, saved.expected)
	if err != nil {
		if errors.IsCode(err, errors.Conflict) {
			return needsHuman("the product " + sc.Page.Path + " changed in the store while its description was written: " +
				"it held " + saved.expected + " and now holds " + currentHashOf(err)), nil
		}
		return run.Result{}, err
	}

	written := saved.written
	categories.took(sc.Page, written.Categories)
	result := PublishResult{
		WPID: held.ID, URL: permalinkOf(wp.Item{Link: written.Permalink}), Status: written.Status, ContentHash: hash,
		PreviousContent: raw.Content, PreviousContentHash: raw.ContentHash, PreviousProduct: &snapshot,
		Categories: categories.write, SEOApplied: make([]string, 0), Skipped: make([]string, 0),
		Findings:   append(findingsOfProduct(sc.Page, held, typed), categories.findings...),
		Mismatches: make([]pagemap.Mismatch, 0),
	}
	seo, err := applySEO(ctx, client, sc, wp.TypeProduct, held.ID, true)
	if err != nil {
		return run.Result{}, err
	}
	result.took(seo)

	if recordErr := recordProduct(ctx, deps, sc, written, hash); recordErr != nil {
		return run.Result{}, recordErr
	}
	return publishedAs(result, "updated the product "+sc.Page.Path+" as "+strconv.FormatInt(held.ID, 10))
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

func productUpdate(held wp.Product, draft *content.ProductDraft, featured int64,
	categories []int64) (wp.UpdateProduct, ProductSnapshot, *content.Finding) {
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
	if categories != nil {
		filed := slices.Clone(categories)
		update.Categories = &filed
	}
	return update, snapshot, typed
}

func hasFields(update wp.UpdateProduct) bool {
	return update.ShortDescription != nil || update.Attributes != nil || update.Images != nil ||
		update.Categories != nil
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
	case wp.StoreForbidden(err):
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
