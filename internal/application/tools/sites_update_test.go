package tools_test

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/davidmovas/postulator/internal/application/sites"
	"github.com/davidmovas/postulator/internal/application/tools"
	domainagent "github.com/davidmovas/postulator/internal/domain/agent"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func updateSite(t *testing.T, call func(json.RawMessage) (any, error), args map[string]any) sites.Site {
	t.Helper()

	out, err := call(encoded(t, args))
	if err != nil {
		t.Fatalf("sites_manage update %v: %v", args, err)
	}
	answered, ok := out.(sites.UpdateResponse)
	if !ok {
		t.Fatalf("sites_manage update answered %T", out)
	}
	return answered.Site
}

func TestSitesUpdateCarriesTheSettings(t *testing.T) {
	t.Parallel()

	registry, binding, seeded := wired(t)
	binding.Mode = domainagent.ModeAutonomous
	call := func(args json.RawMessage) (any, error) {
		return registry.Call(t.Context(), binding, "sites_manage", args)
	}

	steps := []struct {
		name        string
		args        map[string]any
		agent       string
		concurrency int
		cadence     int
		form        string
	}{
		{
			name:  "the agent and the concurrency",
			args:  map[string]any{"userAgent": "Agent/1", "renderConcurrency": 2},
			agent: "Agent/1", concurrency: 2, cadence: 10, form: "auto",
		},
		{
			name:  "the cadence and the form keep the rest",
			args:  map[string]any{"fullWalkEvery": 3, "linkUrlForm": "relative"},
			agent: "Agent/1", concurrency: 2, cadence: 3, form: "relative",
		},
		{
			name:  "a default keeps the settings",
			args:  map[string]any{"defaultTemplateId": seeded.template, "name": "Shop Two"},
			agent: "Agent/1", concurrency: 2, cadence: 3, form: "relative",
		},
		{
			name:  "an empty agent goes back to the default",
			args:  map[string]any{"userAgent": ""},
			agent: "", concurrency: 2, cadence: 3, form: "relative",
		},
	}

	for _, step := range steps {
		step.args["op"], step.args["siteId"] = "update", seeded.site
		got := updateSite(t, call, step.args)
		settings := got.Settings
		if settings.UserAgent != step.agent || settings.Observe.RenderConcurrency != step.concurrency ||
			settings.Observe.FullWalkEvery != step.cadence || string(settings.LinkURLForm) != step.form {
			t.Fatalf("%s: settings = %+v", step.name, settings)
		}
		if len(settings.Builders) == 0 || len(settings.TechnicalShortcodes) == 0 {
			t.Fatalf("%s: the merge lost the builder and shortcode defaults: %+v", step.name, settings)
		}
	}

	read, err := registry.Call(t.Context(), binding, "sites", encoded(t, map[string]any{"view": "get"}))
	if err != nil {
		t.Fatalf("sites get: %v", err)
	}
	held, ok := read.(tools.SitesResponse)
	if !ok || held.Site == nil {
		t.Fatalf("sites get answered %+v", read)
	}
	if held.Site.Defaults.TemplateID == nil || *held.Site.Defaults.TemplateID != seeded.template || held.Site.Name != "Shop Two" {
		t.Fatalf("the settings merge lost the defaults or the name: %+v", held.Site)
	}

	detected := updateSite(t, call, map[string]any{
		"op": "update", "technicalShortcodes": []string{"vc_row", "member_area"}, "technicalBlocks": []string{"acme/form"},
		"builders": []map[string]any{{"name": "Acme Builder", "classPrefixes": []string{"acme-"}}},
	}).Settings
	if !slices.Equal(detected.TechnicalShortcodes, []string{"vc_row", "member_area"}) || !slices.Equal(detected.TechnicalBlocks, []string{"acme/form"}) ||
		len(detected.Builders) != 1 || detected.Builders[0].Name != "Acme Builder" || !slices.Equal(detected.Builders[0].ClassPrefixes, []string{"acme-"}) {
		t.Fatalf("the detection settings = %+v, want the shortcodes, blocks and builder the agent named", detected)
	}
	if detected.Observe.FullWalkEvery != 3 || string(detected.LinkURLForm) != "relative" {
		t.Fatalf("setting the detection lost the rest: %+v", detected)
	}
	reset := updateSite(t, call, map[string]any{"op": "update", "technicalShortcodes": []string{}}).Settings
	if len(reset.TechnicalShortcodes) == 0 || slices.Contains(reset.TechnicalShortcodes, "member_area") {
		t.Fatalf("an empty shortcode list = %v, want the defaults back", reset.TechnicalShortcodes)
	}

	refusals := []struct {
		name string
		args map[string]any
	}{
		{name: "a builder without a name", args: map[string]any{"builders": []map[string]any{{"classPrefixes": []string{"x-"}}}}},
		{name: "a concurrency over the ceiling", args: map[string]any{"renderConcurrency": 17}},
		{name: "a negative concurrency", args: map[string]any{"renderConcurrency": -1}},
		{name: "a negative cadence", args: map[string]any{"fullWalkEvery": -1}},
		{name: "a link form it does not know", args: map[string]any{"linkUrlForm": "sometimes"}},
	}

	for _, tc := range refusals {
		t.Run(tc.name, func(t *testing.T) {
			tc.args["op"] = "update"
			if _, refusedErr := call(encoded(t, tc.args)); !errors.IsCode(refusedErr, errors.Invalid) {
				t.Fatalf("code = %q, want INVALID (err %v)", errors.CodeOf(refusedErr), refusedErr)
			}
		})
	}
}
