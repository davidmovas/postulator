package tools_test

import (
	"encoding/json"
	"testing"

	"github.com/davidmovas/postulator/internal/application/sites"
	domainagent "github.com/davidmovas/postulator/internal/domain/agent"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func updateSite(t *testing.T, call func(json.RawMessage) (any, error), args map[string]any) sites.Site {
	t.Helper()

	out, err := call(encoded(t, args))
	if err != nil {
		t.Fatalf("sites_update %v: %v", args, err)
	}
	answered, ok := out.(sites.UpdateResponse)
	if !ok {
		t.Fatalf("sites_update answered %T", out)
	}
	return answered.Site
}

func TestSitesUpdateCarriesTheSettings(t *testing.T) {
	t.Parallel()

	registry, binding, seeded := wired(t)
	binding.Mode = domainagent.ModeAutonomous
	call := func(args json.RawMessage) (any, error) {
		return registry.Call(t.Context(), binding, "sites_update", args)
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
		step.args["id"] = seeded.site
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

	read, err := registry.Call(t.Context(), binding, "sites_get", encoded(t, map[string]any{"id": seeded.site}))
	if err != nil {
		t.Fatalf("sites_get: %v", err)
	}
	held, ok := read.(sites.GetResponse)
	if !ok {
		t.Fatalf("sites_get answered %T", read)
	}
	if held.Site.Defaults.TemplateID == nil || *held.Site.Defaults.TemplateID != seeded.template || held.Site.Name != "Shop Two" {
		t.Fatalf("the settings merge lost the defaults or the name: %+v", held.Site)
	}

	refusals := []struct {
		name string
		args map[string]any
	}{
		{name: "a concurrency over the ceiling", args: map[string]any{"renderConcurrency": 17}},
		{name: "a negative concurrency", args: map[string]any{"renderConcurrency": -1}},
		{name: "a negative cadence", args: map[string]any{"fullWalkEvery": -1}},
		{name: "a link form it does not know", args: map[string]any{"linkUrlForm": "sometimes"}},
	}

	for _, tc := range refusals {
		t.Run(tc.name, func(t *testing.T) {
			tc.args["id"] = seeded.site
			if _, refusedErr := call(encoded(t, tc.args)); !errors.IsCode(refusedErr, errors.Invalid) {
				t.Fatalf("code = %q, want INVALID (err %v)", errors.CodeOf(refusedErr), refusedErr)
			}
		})
	}
}
