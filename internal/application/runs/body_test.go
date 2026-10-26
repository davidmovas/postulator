package runs_test

import (
	stderrors "errors"
	"slices"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/application/runs"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func infoAndGuide(fixture *fixture) (info, guide string) {
	info, guide = "type-info", "type-guide"
	types := []pagetype.Type{
		{ID: info, SiteID: fixture.siteID, Name: "Info", Preset: pagetype.Info},
		{ID: guide, SiteID: fixture.siteID, Name: "Guide", Preset: pagetype.Guide},
	}
	scopes := append(pagetype.DefaultScopes(pagetype.Info, info), pagetype.DefaultScopes(pagetype.Guide, guide)...)
	fixture.types.set = pagetype.NewSet(types, scopes)
	return info, guide
}

func typedAs(fixture *fixture, pageID, typeID string, onSite bool) {
	if fixture.mapping.known == nil {
		fixture.mapping.known = map[string]pagemap.Page{}
	}
	entityID := "entity-" + pageID
	page := pagemap.Page{ID: pageID, SiteID: fixture.siteID, Path: "/" + pageID + "/", EntityID: &entityID, PageTypeID: &typeID}
	if onSite {
		wpID := int64(100 + len(fixture.mapping.known))
		page.WPID = &wpID
	}
	fixture.mapping.known[pageID] = page
}

func bodyRecipe() []template.StepSpec {
	return []template.StepSpec{
		{Name: "resolve_context", Enabled: true}, {Name: "generate_body", Enabled: true}, {Name: "generate_meta", Enabled: true},
		{Name: "generate_fields", Enabled: true}, {Name: "publish", Enabled: true},
	}
}

func TestAPageTypeWithItsBodyOffScopesTheRunToTheFieldsItWrites(t *testing.T) {
	t.Parallel()

	fixture := newFixture(t)
	fixture.specs.siteID = fixture.siteID
	fixture.specs.spec = template.TemplateSpec{Recipe: bodyRecipe()}
	info, _ := infoAndGuide(fixture)
	typedAs(fixture, fixture.pages[0], info, true)
	typedAs(fixture, fixture.pages[1], info, true)
	estimated, err := fixture.service.Estimate(t.Context(), runs.StartRequest{SiteID: fixture.siteID, PageIDs: fixture.pages})
	if err != nil || !slices.Equal(estimated.Fields, []string{"seo_description", "seo_title"}) {
		t.Fatalf("the estimate over two Info pages = %+v %v, want it to say the run writes the SEO pair alone", estimated, err)
	}
	started, err := fixture.service.Start(t.Context(), runs.StartRequest{SiteID: fixture.siteID, PageIDs: fixture.pages})
	if err != nil || !slices.Equal(started.Fields, estimated.Fields) {
		t.Fatalf("Start over two Info pages = %+v %v", started, err)
	}
	queued := fixture.engine.queued
	if got := enabledNames(queued.Recipe); !slices.Equal(got, []string{"generate_meta", "publish"}) {
		t.Errorf("the run over Info pages keeps %v, want the meta and publish: Info writes no body", got)
	}
	if listed, held := run.ParamsFor(queued.Recipe, "publish")[run.ParamFields].([]string); !held || !slices.Equal(listed, []string{"seo_description", "seo_title"}) {
		t.Errorf("publish's params = %v, want the SEO pair Info writes", run.ParamsFor(queued.Recipe, "publish"))
	}
	if !slices.Equal(queued.Targets, fixture.pages) {
		t.Errorf("the run targets %v, want the two pages and no ancestor", queued.Targets)
	}

	whole := newFixture(t)
	whole.specs.siteID = whole.siteID
	whole.specs.spec = template.TemplateSpec{Recipe: bodyRecipe()}
	_, guide := infoAndGuide(whole)
	typedAs(whole, whole.pages[0], guide, true)
	if started, err = whole.service.Start(t.Context(), runs.StartRequest{SiteID: whole.siteID, PageIDs: whole.pages[:1]}); err != nil || len(started.Fields) != 0 {
		t.Fatalf("Start over a Guide page = %+v %v, want a whole run", started, err)
	}
	if _, scoped := run.ParamsFor(whole.engine.queued.Recipe, "publish")[run.ParamFields]; scoped ||
		!slices.Contains(enabledNames(whole.engine.queued.Recipe), "generate_body") {
		t.Errorf("the run over a Guide page = %+v, want the whole recipe", whole.engine.queued.Recipe)
	}
}

func TestARunRefusesPagesWhoseTypesDisagreeOnTheBody(t *testing.T) {
	t.Parallel()

	withoutMeta := bodyRecipe()
	withoutMeta[2].Enabled = false
	cases := []struct {
		name    string
		recipe  []template.StepSpec
		guide   bool
		planned bool
		says    string
	}{
		{name: "an Info page beside a Guide page", guide: true, says: "start a run for each"},
		{name: "an Info page not on the site", planned: true, says: "cannot create"},
		{name: "a recipe that writes nothing Info writes", recipe: withoutMeta, says: "nothing to write"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			fixture := newFixture(t)
			fixture.specs.siteID = fixture.siteID
			fixture.specs.spec = template.TemplateSpec{Recipe: bodyRecipe()}
			if tc.recipe != nil {
				fixture.specs.spec.Recipe = tc.recipe
			}
			info, guide := infoAndGuide(fixture)
			typedAs(fixture, fixture.pages[0], info, !tc.planned)
			second := info
			if tc.guide {
				second = guide
			}
			typedAs(fixture, fixture.pages[1], second, true)
			_, err := fixture.service.Start(t.Context(), runs.StartRequest{SiteID: fixture.siteID, PageIDs: fixture.pages})
			kernel, typed := stderrors.AsType[*errors.Error](err)
			if !typed || kernel.Code != errors.Invalid || kernel.Details["field"] != "pageIds" || !strings.Contains(err.Error(), tc.says) {
				t.Fatalf("Start = %v, want INVALID on pageIds saying %q", err, tc.says)
			}
			if fixture.engine.queued.ID != "" || len(fixture.engine.queued.Targets) > 0 {
				t.Errorf("a refused run was queued: %+v", fixture.engine.queued)
			}
		})
	}
}
