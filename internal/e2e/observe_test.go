//go:build e2e

package e2e_test

import (
	"slices"
	"testing"

	"github.com/davidmovas/postulator/internal/application/observe"
	"github.com/davidmovas/postulator/internal/application/sites"
	appsync "github.com/davidmovas/postulator/internal/application/sync"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/render"
)

type expectedParent struct {
	path   string
	reason pagemap.ParentReason
}

type standModel struct {
	parents         map[string]expectedParent
	roots           map[string]render.RootState
	listings        []string
	technical       map[string]pagemap.TechnicalReason
	technicalCounts []pagemap.TechnicalReason
	unresolvedCrumb string
	hubs            []string
	trailsResolved  int
}

func modelOf(t *testing.T, env environment) standModel {
	t.Helper()

	switch env.stand {
	case "vanilla":
		return standModel{
			parents: map[string]expectedParent{
				"/guide/setup/":    {path: "/guide/", reason: pagemap.ParentWP},
				"/product/powder/": {path: "/product-category/koffein/", reason: pagemap.ParentJSONLD},
			},
			roots: map[string]render.RootState{
				"/guide/":          render.RootFound,
				"/guide/setup/":    render.RootFound,
				"/product/powder/": render.RootFound,
			},
			technicalCounts: []pagemap.TechnicalReason{
				pagemap.TechnicalShop, pagemap.TechnicalCart, pagemap.TechnicalCheckout, pagemap.TechnicalAccount,
			},
			trailsResolved: 1,
		}
	case "replica":
		return standModel{
			parents: map[string]expectedParent{
				"/bpc-157/":                 {path: "/", reason: pagemap.ParentURL},
				"/tb-500/":                  {path: "/", reason: pagemap.ParentURL},
				"/peptides/":                {path: "/", reason: pagemap.ParentURL},
				"/product/bpc-157-peptide/": {path: "/product-category/peptides/healing-peptides/", reason: pagemap.ParentJSONLD},
				"/product-category/peptides/healing-peptides/": {path: "/product-category/peptides/", reason: pagemap.ParentWP},
			},
			roots: map[string]render.RootState{
				"/peptides/": render.RootEmptyRaw,
				"/bpc-157/":  render.RootDynamicOnly,
				"/tb-500/":   render.RootFound,
			},
			listings:  []string{"/peptides/", "/bpc-157/", "/tb-500/"},
			technical: map[string]pagemap.TechnicalReason{"/": pagemap.TechnicalFront},
			technicalCounts: []pagemap.TechnicalReason{
				pagemap.TechnicalFront, pagemap.TechnicalShop, pagemap.TechnicalCart, pagemap.TechnicalCheckout, pagemap.TechnicalAccount,
			},
			unresolvedCrumb: "/healing-peptides/",
			hubs:            []string{"/bpc-157/", "/tb-500/"},
			trailsResolved:  1,
		}
	}
	t.Fatalf("E2E_STAND is %q, and the observation test knows the vanilla and the replica stands", env.stand)
	return standModel{}
}

func TestTheSiteIsObservedIntoTheModel(t *testing.T) {
	live := newSite(t)
	requirePlugin(t, live.env, true)
	expected := modelOf(t, live.env)

	core := openCore(t)
	owner, err := core.Sites.Create(t.Context(), sites.CreateRequest{
		Name: "Observed Shop", BaseURL: live.env.baseURL, Username: live.env.user, Password: live.env.pass, AllowInsecure: true,
	})
	if err != nil {
		t.Fatalf("create the site: %v", err)
	}
	siteID := owner.Site.ID

	synced, err := core.Sync.SyncSite(t.Context(), appsync.SyncSiteRequest{SiteID: siteID})
	if err != nil {
		t.Fatalf("sync the site: %v", err)
	}
	awaitRun(t, core.Runs, synced.RunID)

	summary, err := core.Observe.Summary(t.Context(), siteID)
	if err != nil {
		t.Fatalf("summarize the model: %v", err)
	}
	t.Logf("observed %d of %d pages on %s: %d rendered, roots %v, %d trails resolved, %d listings, links %v, technical %v, findings %v",
		summary.Observed, summary.Pages, live.env.stand, summary.Rendered, summary.Roots, summary.TrailsResolved, summary.Listings,
		summary.LinksByType, summary.Technical, summary.Findings)
	if summary.Pages == 0 || summary.Observed != summary.Pages {
		t.Fatalf("summary = %+v, want every page observed", summary)
	}
	if summary.Rendered == 0 || summary.LinksByType[string(pagemap.LinkBreadcrumb)] == 0 || summary.LinksByType[string(pagemap.LinkNavigation)] == 0 {
		t.Errorf("summary = %+v, want renders with breadcrumb and navigation links", summary)
	}
	if summary.TrailsResolved < expected.trailsResolved || summary.Listings < len(expected.listings) {
		t.Errorf("trails resolved %d listings %d, want at least %d and %d", summary.TrailsResolved, summary.Listings, expected.trailsResolved, len(expected.listings))
	}
	for _, reason := range expected.technicalCounts {
		if summary.Technical[string(reason)] == 0 {
			t.Errorf("no page is technical for %s: %v", reason, summary.Technical)
		}
	}

	stored := pagesByPath(t, core.Pages, siteID)
	byID := make(map[string]string, len(stored))
	for path, page := range stored {
		byID[page.ID] = path
	}
	models := make(map[string]observe.PageModel, len(stored))
	for path, page := range stored {
		model, modelErr := core.Observe.Page(t.Context(), page.ID)
		if modelErr != nil {
			t.Fatalf("read the model of %s: %v", path, modelErr)
		}
		models[path] = model
		if page.WPID == nil {
			continue
		}
		if model.Identity.PageID != page.ID {
			t.Errorf("%s has no identity row", path)
		}
		if !slices.ContainsFunc(model.Fields, func(value field.Value) bool { return value.Key == field.Title }) {
			t.Errorf("%s has no title field", path)
		}
		if page.Status == string(pagemap.StatusPublished) && model.Render == nil {
			t.Errorf("%s is published and has no render row", path)
		}
		if !model.Page.WPType.Term() && model.Body == nil {
			t.Errorf("%s has no body row", path)
		}
	}

	for path, want := range expected.parents {
		model, ok := models[path]
		if !ok {
			t.Errorf("the model holds no %s", path)
			continue
		}
		if model.Page.ParentReason != want.reason || model.Page.ParentPageID == nil || byID[*model.Page.ParentPageID] != want.path {
			parent := "<none>"
			if model.Page.ParentPageID != nil {
				parent = byID[*model.Page.ParentPageID]
			}
			t.Errorf("%s sits under %s by %s, want %s by %s", path, parent, model.Page.ParentReason, want.path, want.reason)
		}
	}
	for path, want := range expected.roots {
		model, ok := models[path]
		if !ok || model.Render == nil {
			t.Errorf("%s has no render to read the root from", path)
			continue
		}
		if model.Render.Root != want {
			t.Errorf("%s has the content root %s, want %s", path, model.Render.Root, want)
		}
	}
	for _, path := range expected.listings {
		model, ok := models[path]
		if !ok || model.Render == nil || !slices.ContainsFunc(model.Render.Regions, func(region render.Region) bool { return region.Kind == render.RegionListing }) {
			t.Errorf("%s carries no listing region", path)
		}
	}
	for path, want := range expected.technical {
		if model, ok := models[path]; !ok || model.Page.TechnicalReason != want {
			t.Errorf("%s is technical for %q, want %s", path, model.Page.TechnicalReason, want)
		}
	}
	for _, hub := range expected.hubs {
		model, ok := models[hub]
		if !ok {
			t.Errorf("the model holds no hub %s", hub)
			continue
		}
		subject := finding.SubjectCrumb(model.Page.ID, expected.unresolvedCrumb)
		if !slices.ContainsFunc(model.Findings, func(f finding.Finding) bool {
			return f.Code == finding.ParentUnresolved && f.SubjectKey == subject && f.Status == finding.Open
		}) {
			t.Errorf("%s raises no parent_unresolved for %s: %+v", hub, expected.unresolvedCrumb, model.Findings)
		}
	}

	again, err := core.Sync.SyncSite(t.Context(), appsync.SyncSiteRequest{SiteID: siteID})
	if err != nil {
		t.Fatalf("sync the site again: %v", err)
	}
	awaitRun(t, core.Runs, again.RunID)
	after, err := core.Observe.Summary(t.Context(), siteID)
	if err != nil {
		t.Fatalf("summarize the model again: %v", err)
	}
	if after.Pages != summary.Pages || after.Observed != summary.Observed || after.Rendered != summary.Rendered ||
		after.TrailsResolved != summary.TrailsResolved || after.Listings != summary.Listings {
		t.Errorf("the second observation changed the model: before %+v after %+v", summary, after)
	}
	for code, count := range summary.Findings {
		if after.Findings[code] != count {
			t.Errorf("the second observation changed the open %s findings from %d to %d", code, count, after.Findings[code])
		}
	}
	for path, before := range models {
		model, modelErr := core.Observe.Page(t.Context(), before.Page.ID)
		if modelErr != nil {
			t.Fatalf("read the model of %s again: %v", path, modelErr)
		}
		if before.Render != nil && (model.Render == nil || model.Render.Hash != before.Render.Hash) {
			t.Errorf("the second observation changed the render of %s", path)
		}
		if before.Body != nil && (model.Body == nil || model.Body.Hash != before.Body.Hash || model.Base.Hash != before.Base.Hash) {
			t.Errorf("the second observation changed the body of %s", path)
		}
		if model.Page.ParentReason != before.Page.ParentReason {
			t.Errorf("the second observation changed the parent of %s from %s to %s", path, before.Page.ParentReason, model.Page.ParentReason)
		}
	}
}
