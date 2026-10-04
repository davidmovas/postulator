package steps_test

import (
	"bytes"
	"context"
	"net/http"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/site"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/runtime/steps"
)

const (
	productDescription = `<p>Espresso is a way to make <a href="/coffee/">coffee</a>.</p>`
	productShort       = `<p>A compact machine for the counter.</p>`
	productDraft       = `{"title":"Espresso guide","h1":"Espresso Machine","sections":[` +
		`{"heading":"About","html":"<p>Espresso is a way to make coffee, part of our drinks range.</p>"}` +
		`],"summary":"A short guide to espresso.","product":{"shortDescription":"<p>A compact machine for the counter.</p>",` +
		`"specifications":[{"name":"Form","value":"Countertop"},{"name":"Origin","value":"Spain"},{"name":"Size","value":"Large"}]}}`
)

type siteKeeper struct {
	record  site.Site
	updates []site.Site
}

func (s *siteKeeper) Get(context.Context, string) (site.Site, error) {
	return s.record, nil
}

func (s *siteKeeper) Update(_ context.Context, record site.Site) error {
	s.record = record
	s.updates = append(s.updates, record)
	return nil
}

func storeSite() site.Site {
	return site.Site{
		ID: "site", Name: "Shop", BaseURL: "https://shop.example.com", Username: "editor", Status: site.StatusActive,
		Commerce: site.CommerceReady,
		Plugin:   site.PluginState{Installed: true, Capabilities: []string{wp.CapabilityRaw, steps.CapabilitySEOMeta}},
	}
}

func storeProduct() wptest.Item {
	return wptest.Item{
		Type: wptest.TypeProduct, Title: "Espresso Machine", Slug: "espresso-machine", Status: "publish",
		Content: "<p>old description</p>", Excerpt: "<p>old short</p>", RegularPrice: "120", SKU: "EM-1",
		Attributes: []wptest.Attribute{
			{Name: "Origin", Options: []string{"Italy"}, Visible: true},
			{Name: "form", Options: []string{}, Position: 1},
			{ID: 3, Name: "Size", Options: []string{}, Position: 2, Visible: true},
		},
	}
}

type productHarness struct {
	deps     steps.Deps
	server   *wptest.Server
	sites    *siteKeeper
	recorded *pagemap.Page
	held     wptest.Item
}

func newProductHarness(t *testing.T, item wptest.Item, opts ...wptest.Option) productHarness {
	t.Helper()

	deps, server := imageDepsWith(t, opts...)
	sites := &siteKeeper{record: storeSite()}
	recorded := &pagemap.Page{}
	deps.Sites = sites
	deps.SiteWriter = sites
	deps.Pages = pageList{recorded: recorded}
	return productHarness{deps: deps, server: server, sites: sites, recorded: recorded, held: server.Seed(item)[0]}
}

func storeContext(t *testing.T, wpID int64) *run.StepContext {
	t.Helper()

	sc := unitContext(t, map[run.ArtifactKind][]byte{
		run.ArtifactBodyHTML: []byte(`<h1>Espresso Machine</h1>` + productDescription),
		run.ArtifactDraft:    []byte(productDraft),
		run.ArtifactMeta:     []byte(`{"title":"Espresso Machine | Shop","description":"Pull a shot.","canonical":"https://shop.example.com/product/espresso-machine/"}`),
	})
	sc.Run.PublishMode = run.PublishLive
	sc.Page.Path = "/product/espresso-machine/"
	sc.Page.Slug = "espresso-machine"
	sc.Page.WPType = pagemap.WPProduct
	sc.Page.WPID = &wpID
	sc.Page.Status = pagemap.StatusPublished
	sc.Page.Observed = pagemap.Observed{Title: "Espresso Machine", Slug: "espresso-machine", Status: "publish"}
	sc.Spec.Product = &template.Product{
		ShortDescription: template.ProductShortDescription{Enabled: true, Intent: "Say what it is for", TargetWords: 30},
		Specifications:   []template.ProductSpecification{{Name: "Form"}, {Name: "Origin"}, {Name: "Size"}},
	}
	return sc
}

func TestPublishEditsAProductAndLeavesTheStoreItsOwn(t *testing.T) {
	t.Parallel()

	h := newProductHarness(t, storeProduct())
	published := runPublish(t, h.deps, storeContext(t, h.held.ID))

	stored, ok := h.server.Lookup(h.held.ID)
	if !ok {
		t.Fatal("the product left the store")
	}
	if stored.Content != productDescription {
		t.Errorf("description = %q, want the body without its h1", stored.Content)
	}
	if stored.Excerpt != productShort {
		t.Errorf("short description = %q, want the draft's", stored.Excerpt)
	}
	if stored.Title != "Espresso Machine" || stored.Status != "publish" || stored.Slug != "espresso-machine" ||
		stored.RegularPrice != "120" || stored.SKU != "EM-1" {
		t.Errorf("the store's own fields moved: %+v", stored)
	}
	want := []wptest.Attribute{
		{Name: "Origin", Options: []string{"Italy"}, Visible: true},
		{Name: "form", Options: []string{"Countertop"}, Position: 1},
		{ID: 3, Name: "Size", Options: []string{}, Position: 2, Visible: true},
	}
	if !reflect.DeepEqual(stored.Attributes, want) {
		t.Errorf("attributes = %+v, want %+v", stored.Attributes, want)
	}
	if stored.Meta["_yoast_wpseo_title"] != "Espresso Machine | Shop" {
		t.Errorf("the run did not write its meta: %v", stored.Meta)
	}

	if published.Created || published.WPID != h.held.ID || published.Status != "publish" {
		t.Errorf("publish = %+v, want an update of the live product", published)
	}
	if published.ContentHash != wp.ContentHash(productDescription) {
		t.Errorf("contentHash = %q, want the hash of what was written", published.ContentHash)
	}
	if published.PreviousContent != "<p>old description</p>" ||
		published.PreviousContentHash != wp.ContentHash("<p>old description</p>") {
		t.Errorf("previous content = %q / %q", published.PreviousContent, published.PreviousContentHash)
	}
	snapshot := published.PreviousProduct
	if snapshot == nil {
		t.Fatal("the publish kept no snapshot of the product")
	}
	if snapshot.ShortDescription != "<p>old short</p>" || !snapshot.ShortWritten || !snapshot.AttributesSent || snapshot.ImageID != 0 {
		t.Errorf("snapshot = %+v", snapshot)
	}
	if snapshot.WrittenShort != productShort || len(snapshot.Written) != 1 || snapshot.Written[0].Name != "form" ||
		!slices.Equal(snapshot.Written[0].Options, []string{"Countertop"}) {
		t.Errorf("the snapshot records it wrote %q and %+v", snapshot.WrittenShort, snapshot.Written)
	}
	if !slices.Equal(snapshot.Added, []string{"Form"}) {
		t.Errorf("added = %v, want only the empty attribute the template filled", snapshot.Added)
	}
	if len(snapshot.StoreAttributes()) != 3 || len(snapshot.StoreAttributes()[1].Options) != 0 {
		t.Errorf("the snapshot holds %+v, want the attributes as they were", snapshot.StoreAttributes())
	}
	if published.PreviousMeta == nil {
		t.Error("the publish kept no previous meta")
	}

	if h.recorded.ContentHash != published.ContentHash || h.recorded.Status != pagemap.StatusPublished ||
		h.recorded.Observed.Title != "Espresso Machine" || h.recorded.Drift {
		t.Errorf("the page records %+v", h.recorded)
	}
	if len(h.sites.updates) != 0 {
		t.Errorf("a ready store was written back: %+v", h.sites.updates)
	}
}

func TestPublishWritesTheDescriptionAfterTheStoreResavesTheProduct(t *testing.T) {
	t.Parallel()

	const lines = `<p>Pull a shot.<br/>Steam the milk.</p>`
	item := storeProduct()
	item.Content = "<p>old<br>description</p>"
	h := newProductHarness(t, item, wptest.WithFilteredHTML())
	sc := storeContext(t, h.held.ID)
	sc.Artifacts[run.ArtifactBodyHTML] = run.Artifact{Kind: run.ArtifactBodyHTML, Blob: []byte(`<h1>Espresso Machine</h1>` + lines)}
	published := runPublish(t, h.deps, sc)

	stored, _ := h.server.Lookup(h.held.ID)
	if stored.Content != lines || published.ContentHash != wp.ContentHash(lines) {
		t.Errorf("description = %q under %q, want %q byte for byte after the store saved the short description",
			stored.Content, published.ContentHash, lines)
	}
	if stored.Excerpt != productShort {
		t.Errorf("short description = %q", stored.Excerpt)
	}
	if published.PreviousContent != "<p>old<br>description</p>" {
		t.Errorf("previous content = %q, want the description the client had", published.PreviousContent)
	}
}

func TestPublishPausesAProductItCannotWrite(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		options  []wptest.Option
		prepare  func(*testing.T, productHarness, *run.StepContext)
		commerce site.Commerce
		message  string
	}{
		{
			name: "a row with no product",
			prepare: func(_ *testing.T, _ productHarness, sc *run.StepContext) {
				sc.Page.WPID = nil
			},
			message: "create it in WooCommerce",
		},
		{
			name: "a product the store no longer holds",
			prepare: func(t *testing.T, h productHarness, _ *run.StepContext) {
				if !h.server.Delete(h.held.ID) {
					t.Fatal("the seeded product could not be removed")
				}
			},
			message: "no longer in the store",
		},
		{name: "a site with no store", options: []wptest.Option{wptest.WithoutCommerce()}, commerce: site.CommerceAbsent, message: "no WooCommerce store"},
		{name: "a user who may not edit products", options: []wptest.Option{wptest.WithoutProductEdit()}, commerce: site.CommerceForbidden, message: "may not edit products"},
		{
			name: "a description a human changed while it was written",
			prepare: func(_ *testing.T, h productHarness, _ *run.StepContext) {
				h.server.EditBeforeNextRawWrite(h.held.ID, "<p>a human was here</p>")
			},
			message: "changed in the store",
		},
		{
			name: "a drifted product the recipe refuses",
			prepare: func(_ *testing.T, _ productHarness, sc *run.StepContext) {
				sc.Page.Drift = true
				sc.Params[steps.ParamRefuseDrift] = true
			},
			message: "edited",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := newProductHarness(t, storeProduct(), tc.options...)
			sc := storeContext(t, h.held.ID)
			if tc.prepare != nil {
				tc.prepare(t, h, sc)
			}

			result, err := steps.Publish(h.deps).Run(t.Context(), sc)
			if err != nil {
				t.Fatalf("Publish: %v", err)
			}
			if result.Next != run.TransitionPause || result.Reason != run.PauseNeedsHuman || len(result.Artifacts) != 0 {
				t.Fatalf("Publish = %+v, want a pause for a human with nothing kept", result)
			}
			if !strings.Contains(result.Message, tc.message) {
				t.Errorf("message = %q, want it to say %q", result.Message, tc.message)
			}
			if tc.commerce == "" && len(h.sites.updates) != 0 {
				t.Errorf("the site was rewritten: %+v", h.sites.updates)
			}
			if tc.commerce != "" && (len(h.sites.updates) != 1 || h.sites.record.Commerce != tc.commerce) {
				t.Errorf("the site keeps %+v after %d updates, want the store %q", h.sites.record.Commerce, len(h.sites.updates), tc.commerce)
			}
			if stored, held := h.server.Lookup(h.held.ID); held && stored.Content == productDescription {
				t.Errorf("the refused product holds the new description")
			}
		})
	}
}

func TestPublishWritesOverDriftAndSaysSoForAPageAndAProductAlike(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		setup func(*testing.T) (steps.Deps, *run.StepContext)
		codes []string
	}{
		{
			name: "a page",
			setup: func(t *testing.T) (steps.Deps, *run.StepContext) {
				t.Helper()
				deps, _ := imageDeps(t)
				return deps, publishContext(t)
			},
			codes: []string{steps.CodePublishOverDrift},
		},
		{
			name: "a product the file names otherwise",
			setup: func(t *testing.T) (steps.Deps, *run.StepContext) {
				t.Helper()
				h := newProductHarness(t, storeProduct())
				sc := storeContext(t, h.held.ID)
				sc.Page.H1 = "Espresso Maker"
				return h.deps, sc
			},
			codes: []string{steps.CodePublishOverDrift, steps.CodeProductNameDiffers},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			deps, sc := tc.setup(t)
			sc.Page.Drift = true

			published := runPublish(t, deps, sc)
			if published.WPID == 0 {
				t.Fatalf("publish over drift = %+v, want the item written", published)
			}
			codes := make([]string, 0, len(published.Findings))
			for i := range published.Findings {
				codes = append(codes, published.Findings[i].Code)
			}
			if !slices.Equal(codes, tc.codes) {
				t.Fatalf("findings = %v, want %v in that order", codes, tc.codes)
			}
		})
	}
}

func TestPublishFillsOnlyTheAttributesAProductLacks(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		kind     string
		held     []wptest.Attribute
		want     []wptest.Attribute
		added    []string
		findings []string
	}{
		{
			name: "a product with no attributes",
			held: []wptest.Attribute{},
			want: []wptest.Attribute{
				{Name: "Form", Options: []string{"Countertop"}, Visible: true},
				{Name: "Origin", Options: []string{"Spain"}, Position: 1, Visible: true},
				{Name: "Size", Options: []string{"Large"}, Position: 2, Visible: true},
			},
			added: []string{"Form", "Origin", "Size"},
		},
		{
			name: "a product that carries every attribute",
			held: []wptest.Attribute{
				{Name: "FORM", Options: []string{"Wall"}},
				{Name: " origin ", Options: []string{"Italy"}, Position: 1},
				{Name: "Size", Options: []string{"Small"}, Position: 2},
			},
			want: []wptest.Attribute{
				{Name: "FORM", Options: []string{"Wall"}},
				{Name: " origin ", Options: []string{"Italy"}, Position: 1},
				{Name: "Size", Options: []string{"Small"}, Position: 2},
			},
			added: []string{},
		},
		{
			name:  "a variation attribute with no options",
			held:  []wptest.Attribute{{Name: "Form", Options: []string{}, Variation: true}, {Name: "Origin", Options: []string{"Italy"}, Position: 1}, {Name: "Size", Options: []string{"Small"}, Position: 2}},
			want:  []wptest.Attribute{{Name: "Form", Options: []string{}, Variation: true}, {Name: "Origin", Options: []string{"Italy"}, Position: 1}, {Name: "Size", Options: []string{"Small"}, Position: 2}},
			added: []string{},
		},
		{
			name:     "a variable product",
			kind:     "variable",
			held:     []wptest.Attribute{{Name: "Form", Options: []string{}, Variation: true}},
			want:     []wptest.Attribute{{Name: "Form", Options: []string{}, Variation: true}},
			added:    []string{},
			findings: []string{steps.CodeProductTypeKept},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			item := storeProduct()
			item.ProductType = tc.kind
			item.Attributes = tc.held
			h := newProductHarness(t, item)
			published := runPublish(t, h.deps, storeContext(t, h.held.ID))

			stored, _ := h.server.Lookup(h.held.ID)
			if !reflect.DeepEqual(stored.Attributes, tc.want) {
				t.Errorf("attributes = %+v, want %+v", stored.Attributes, tc.want)
			}
			if !slices.Equal(published.PreviousProduct.Added, tc.added) {
				t.Errorf("added = %v, want %v", published.PreviousProduct.Added, tc.added)
			}
			if published.PreviousProduct.AttributesSent != (len(tc.added) > 0) {
				t.Errorf("attributesSent = %t with %v added", published.PreviousProduct.AttributesSent, tc.added)
			}
			if got := findingCodes(published.Findings); !slices.Equal(got, orNone(tc.findings)) {
				t.Errorf("findings = %v, want %v", got, tc.findings)
			}
			if stored.Excerpt != productShort {
				t.Errorf("short description = %q, want it written whatever the attributes", stored.Excerpt)
			}
		})
	}
}

func TestPublishSetsAProductImageOnlyWhereThereIsNone(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		images []int64
		want   []int64
		set    bool
	}{
		{name: "a product with no image", images: []int64{}, want: []int64{77}, set: true},
		{name: "a product with an image", images: []int64{5}, want: []int64{5}, set: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			item := storeProduct()
			item.Images = tc.images
			h := newProductHarness(t, item)
			sc := storeContext(t, h.held.ID)
			sc.Artifacts[run.ArtifactImages] = run.Artifact{Kind: run.ArtifactImages, Blob: []byte(`{"featuredId":77}`)}
			published := runPublish(t, h.deps, sc)

			stored, _ := h.server.Lookup(h.held.ID)
			if !slices.Equal(stored.Images, tc.want) {
				t.Errorf("images = %v, want %v", stored.Images, tc.want)
			}
			if (published.PreviousProduct.ImageID == 77) != tc.set || !slices.Equal(published.PreviousProduct.Images, tc.images) {
				t.Errorf("snapshot = %+v, want the image set %t over %v", published.PreviousProduct, tc.set, tc.images)
			}
		})
	}
}

func TestPublishSaysWhenTheFileNamesAProductOtherwise(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		h1   string
		want []string
	}{
		{name: "no h1 in the file", h1: "", want: []string{}},
		{name: "the store's name in another case", h1: "espresso machine", want: []string{}},
		{name: "another name", h1: "Espresso Maker", want: []string{steps.CodeProductNameDiffers}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := newProductHarness(t, storeProduct())
			sc := storeContext(t, h.held.ID)
			sc.Page.H1 = tc.h1
			published := runPublish(t, h.deps, sc)

			if got := findingCodes(published.Findings); !slices.Equal(got, tc.want) {
				t.Fatalf("findings = %v, want %v", got, tc.want)
			}
			if stored, _ := h.server.Lookup(h.held.ID); stored.Title != "Espresso Machine" {
				t.Errorf("name = %q, want the store's", stored.Title)
			}
		})
	}
}

func TestPublishPreflightRefusesAProductBeforeItSpends(t *testing.T) {
	t.Parallel()

	wpID := int64(12)
	product := pagemap.Page{
		ID: "page-product", SiteID: "site", Path: "/product/espresso-machine/", WPType: pagemap.WPProduct,
		WPID: &wpID, EntityID: pointer("child"),
	}
	outputs := &template.Product{ShortDescription: template.ProductShortDescription{Enabled: true, TargetWords: 30}}

	cases := []struct {
		name    string
		page    pagemap.Page
		owner   func(*site.Site)
		mode    run.PublishMode
		outputs *template.Product
		errors  []string
		warns   []string
	}{
		{name: "a product the store can take", page: product, outputs: outputs},
		{name: "a store never checked", page: product, outputs: outputs, owner: func(s *site.Site) { s.Commerce = site.CommerceUnknown }, errors: []string{steps.CodeCommerceUnknown}},
		{name: "a site with no store", page: product, outputs: outputs, owner: func(s *site.Site) { s.Commerce = site.CommerceAbsent }, errors: []string{steps.CodeCommerceAbsent}},
		{name: "a user who may not edit products", page: product, outputs: outputs, owner: func(s *site.Site) { s.Commerce = site.CommerceForbidden }, errors: []string{steps.CodeCommerceForbidden}},
		{
			name: "a site without the plugin", page: product, outputs: outputs,
			owner:  func(s *site.Site) { s.Plugin = site.PluginState{} },
			errors: []string{steps.CodeProductNeedsPlugin}, warns: []string{steps.CodePluginMissing},
		},
		{
			name: "a plugin that cannot write raw", page: product, outputs: outputs,
			owner:  func(s *site.Site) { s.Plugin.Capabilities = []string{steps.CapabilitySEOMeta} },
			errors: []string{steps.CodeProductNeedsPlugin},
		},
		{
			name: "a row the store does not hold", outputs: outputs,
			page:   pagemap.Page{ID: "page-product", SiteID: "site", Path: "/espresso-machine/", WPType: pagemap.WPProduct},
			errors: []string{steps.CodeProductNotInStore},
		},
		{name: "a draft run", page: product, outputs: outputs, mode: run.PublishDraft, errors: []string{steps.CodeProductEditedLive}},
		{name: "a template with no product outputs", page: product, warns: []string{steps.CodeProductOutputsMissing}},
		{
			name: "a product category", outputs: outputs,
			page:   pagemap.Page{ID: "page-shelf", SiteID: "site", Path: "/product-category/machines/", WPType: pagemap.WPProductCategory, WPID: &wpID},
			errors: []string{steps.CodeProductCategoryUnwritable},
		},
		{
			name: "a page whose template declares product outputs", outputs: outputs,
			page:  pagemap.Page{ID: "page-child", SiteID: "site", Path: "/coffee/espresso/", WPType: pagemap.WPPage},
			warns: []string{steps.CodeProductOutputsIgnored},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			owner := storeSite()
			if tc.owner != nil {
				tc.owner(&owner)
			}
			deps := unitDeps()
			deps.Sites = siteStub{record: owner}
			record, targets := preflightRun(run.GenerateRecipe(), tc.page)
			record.PublishMode = run.PublishLive
			if tc.mode != "" {
				record.PublishMode = tc.mode
			}
			target := targets[tc.page.ID]
			target.Spec.Product = tc.outputs
			targets[tc.page.ID] = target

			findings, err := steps.Publish(deps).Preflight(t.Context(), record, targets)
			if err != nil {
				t.Fatalf("Preflight: %v", err)
			}
			errs, warns := make([]string, 0), make([]string, 0)
			for _, finding := range findings {
				switch finding.Severity {
				case content.SeverityError:
					errs = append(errs, finding.Code)
				default:
					warns = append(warns, finding.Code)
				}
			}
			if !slices.Equal(errs, orNone(tc.errors)) || !slices.Equal(warns, orNone(tc.warns)) {
				t.Fatalf("findings = %+v, want errors %v and warnings %v", findings, tc.errors, tc.warns)
			}
		})
	}
}

func orNone(codes []string) []string {
	if codes == nil {
		return []string{}
	}
	return codes
}

func TestPublishNeverWritesAProductsCategories(t *testing.T) {
	t.Parallel()

	h := newProductHarness(t, storeProduct())

	published := runPublish(t, h.deps, storeContext(t, h.held.ID))
	if published.WPID != h.held.ID {
		t.Fatalf("publish = %+v, want an update of the product", published)
	}
	for _, request := range h.server.Requests() {
		if request.Method != http.MethodGet && strings.HasPrefix(request.Path, "/wp-json/wc/v3/products") &&
			bytes.Contains(request.Body, []byte(`"categories"`)) {
			t.Errorf("the publish wrote the product's categories: %s %s %s", request.Method, request.Path, request.Body)
		}
	}
	if stored, _ := h.server.Lookup(h.held.ID); len(stored.Categories) != 0 {
		t.Errorf("the product carries %v, want the store's categories untouched", stored.Categories)
	}
}
