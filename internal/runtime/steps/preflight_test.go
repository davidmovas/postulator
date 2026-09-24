package steps_test

import (
	"context"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/images"
	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/site"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/runtime/steps"
)

type stubSource struct{}

func (stubSource) Pick(context.Context, images.Query) ([]images.Image, error) {
	return nil, nil
}

type stubProvider struct{}

func (stubProvider) Generate(context.Context, images.Prompt) (images.Image, error) {
	return images.Image{}, nil
}

func preflightRun(recipe []template.StepSpec, targets ...pagemap.Page) (record run.Run, planned map[string]run.Target) {
	record = run.Run{ID: "run", SiteID: "site", Kind: run.KindGenerate, Recipe: recipe}
	planned = make(map[string]run.Target, len(targets))
	for i := range targets {
		record.Targets = append(record.Targets, targets[i].ID)
		planned[targets[i].ID] = run.Target{Page: targets[i], Spec: spec()}
	}
	return record, planned
}

func codesOf(findings []run.EstimateFinding) []string {
	out := make([]string, 0, len(findings))
	for i := range findings {
		out = append(out, findings[i].Code)
	}
	return out
}

func TestResolveContextPreflightNamesWhatTheGraphLacks(t *testing.T) {
	t.Parallel()

	mapped := pagemap.Page{ID: "page-child", SiteID: "site", Path: "/coffee/espresso/", WPType: pagemap.WPPage, EntityID: pointer("child")}
	unmapped := pagemap.Page{ID: "page-orphan", SiteID: "site", Path: "/orphan/", WPType: pagemap.WPPage}
	homeless := pagemap.Page{ID: "page-latte", SiteID: "site", Path: "/coffee/latte/", WPType: pagemap.WPPage, EntityID: pointer("latte")}

	orphaned := unitEntities()
	orphaned[0].CanonicalPageID = nil
	orphaned = append(orphaned, graph.Entity{
		ID: "latte", SiteID: "site", Name: "Latte", PrimaryKeyword: "latte",
		Anchors: []graph.Anchor{{Text: "latte", Source: graph.AnchorUser, Weight: 1}},
		Kind:    graph.KindTopic, Source: graph.SourceUser, CanonicalPageID: pointer("page-latte"),
	})
	orphanedEdges := append(unitEdges(), graph.Edge{
		ID: "e2", SiteID: "site", FromEntityID: "latte", ToEntityID: "parent",
		Kind: graph.EdgeParent, Weight: 1, Source: graph.SourceUser, Status: graph.StatusApproved,
	})

	cases := []struct {
		name     string
		deps     func(steps.Deps) steps.Deps
		recipe   []template.StepSpec
		targets  []pagemap.Page
		codes    []string
		severity content.Severity
	}{
		{
			name:    "a mapped page whose parent has a page",
			targets: []pagemap.Page{mapped},
			codes:   []string{},
		},
		{
			name:     "a page mapped to nothing",
			targets:  []pagemap.Page{unmapped},
			codes:    []string{steps.CodeEntityMissing},
			severity: content.SeverityError,
		},
		{
			name: "a required parent without a page",
			deps: func(d steps.Deps) steps.Deps {
				d.Entities = entityList{items: orphaned}
				d.Edges = edgeList{items: orphanedEdges}
				return d
			},
			targets:  []pagemap.Page{homeless},
			codes:    []string{steps.CodeRequiredTargetUnplaced},
			severity: content.SeverityError,
		},
		{
			name: "a required parent without a page when validate allows errors",
			deps: func(d steps.Deps) steps.Deps {
				d.Entities = entityList{items: orphaned}
				d.Edges = edgeList{items: orphanedEdges}
				return d
			},
			recipe: []template.StepSpec{
				{Name: steps.NameResolveContext, Enabled: true},
				{Name: steps.NameValidate, Enabled: true, Params: map[string]any{steps.ParamAllowErrors: true}},
			},
			targets:  []pagemap.Page{homeless},
			codes:    []string{steps.CodeRequiredTargetUnplaced},
			severity: content.SeverityWarn,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			deps := unitDeps()
			if tc.deps != nil {
				deps = tc.deps(deps)
			}
			recipe := tc.recipe
			if recipe == nil {
				recipe = []template.StepSpec{{Name: steps.NameResolveContext, Enabled: true}}
			}
			record, targets := preflightRun(recipe, tc.targets...)

			findings, err := steps.ResolveContext(deps).Preflight(t.Context(), record, targets)
			if err != nil {
				t.Fatalf("Preflight: %v", err)
			}
			if got := codesOf(findings); len(got) != len(tc.codes) || (len(got) > 0 && got[0] != tc.codes[0]) {
				t.Fatalf("findings = %+v, want %v", findings, tc.codes)
			}
			for _, finding := range findings {
				if finding.Severity != tc.severity || finding.PageID != tc.targets[0].ID || finding.Path != tc.targets[0].Path {
					t.Fatalf("finding = %+v, want it graded %s and naming the page", finding, tc.severity)
				}
			}
		})
	}
}

func TestGenerateImagesPreflightWarnsWhenTheSourceIsNotConfigured(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		images  template.Images
		sources map[template.ImageSource]steps.ImageSource
		drawn   bool
		codes   []string
	}{
		{name: "no images asked for", images: template.Images{}, codes: []string{}},
		{name: "drawn images without a model", images: template.Images{Featured: true, Source: template.ImagesAI},
			codes: []string{steps.CodeImageSourceUnavailable}},
		{name: "drawn images with a model", images: template.Images{Featured: true, Source: template.ImagesAI},
			drawn: true, codes: []string{}},
		{name: "a folder that is not configured", images: template.Images{Inline: 1, Source: template.ImagesLocal},
			codes: []string{steps.CodeImageSourceUnavailable}},
		{name: "a library that is configured", images: template.Images{Inline: 1, Source: template.ImagesWPMedia},
			sources: map[template.ImageSource]steps.ImageSource{template.ImagesWPMedia: stubSource{}}, codes: []string{}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			deps := unitDeps()
			deps.ImageSources = tc.sources
			if tc.drawn {
				deps.ImageProvider = stubProvider{}
			}
			page := pagemap.Page{ID: "page-child", SiteID: "site", Path: "/coffee/espresso/", WPType: pagemap.WPPage, EntityID: pointer("child")}
			record, targets := preflightRun([]template.StepSpec{{Name: steps.NameGenerateImages, Enabled: true}}, page)
			target := targets[page.ID]
			target.Spec.Images = tc.images
			targets[page.ID] = target

			findings, err := steps.GenerateImages(deps).Preflight(t.Context(), record, targets)
			if err != nil {
				t.Fatalf("Preflight: %v", err)
			}
			if got := codesOf(findings); len(got) != len(tc.codes) || (len(got) > 0 && got[0] != tc.codes[0]) {
				t.Fatalf("findings = %+v, want %v", findings, tc.codes)
			}
			for _, finding := range findings {
				if finding.Severity != content.SeverityWarn || finding.Path != page.Path {
					t.Fatalf("finding = %+v, want a warning naming the page", finding)
				}
			}
		})
	}
}

func TestTheStepsThatTalkToThePluginWarnWhenTheSiteHasNone(t *testing.T) {
	t.Parallel()

	for _, installed := range []bool{true, false} {
		t.Run(map[bool]string{true: "with the plugin", false: "without the plugin"}[installed], func(t *testing.T) {
			t.Parallel()

			deps := unitDeps()
			deps.Sites = siteStub{record: site.Site{
				ID: "site", Name: "Shop", BaseURL: "https://shop.example.com", Username: "editor",
				Status: site.StatusActive, Plugin: site.PluginState{Installed: installed},
			}}
			page := pagemap.Page{ID: "page-child", SiteID: "site", Path: "/coffee/espresso/", WPType: pagemap.WPPage, EntityID: pointer("child")}
			record, targets := preflightRun(run.GenerateRecipe(), page)

			for _, def := range []run.StepDef{
				steps.Publish(deps), steps.RelinkNeighbors(deps), steps.RelinkPage(deps), steps.SyncBack(deps),
			} {
				findings, err := def.Preflight(t.Context(), record, targets)
				if err != nil {
					t.Fatalf("%s.Preflight: %v", def.Name, err)
				}
				if installed && len(findings) != 0 {
					t.Fatalf("%s warns %+v although the plugin is installed", def.Name, findings)
				}
				if !installed && (len(findings) != 1 || findings[0].Code != steps.CodePluginMissing || findings[0].Severity != content.SeverityWarn) {
					t.Fatalf("%s reports %+v, want one plugin_missing warning", def.Name, findings)
				}
			}
		})
	}
}
