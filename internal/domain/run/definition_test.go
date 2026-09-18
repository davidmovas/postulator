package run_test

import (
	"context"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func noop(context.Context, *run.StepContext) (run.Result, error) {
	return run.Result{}, nil
}

func testRegistry(t *testing.T) *run.Registry {
	t.Helper()

	registry := run.NewRegistry()
	defs := []run.StepDef{
		{Name: "resolve_context", Produces: []run.ArtifactKind{run.ArtifactLinkContext}, Run: noop},
		{
			Name:     "generate_body",
			Requires: []run.ArtifactKind{run.ArtifactLinkContext},
			Produces: []run.ArtifactKind{run.ArtifactDraft, run.ArtifactBodyHTML},
			Run:      noop,
		},
		{
			Name:     "validate",
			Requires: []run.ArtifactKind{run.ArtifactBodyHTML},
			Produces: []run.ArtifactKind{run.ArtifactValidationReport},
			Run:      noop,
		},
	}
	for _, def := range defs {
		if err := registry.Register(def); err != nil {
			t.Fatalf("Register(%s): %v", def.Name, err)
		}
	}
	return registry
}

func TestRegistryRejectsBadSteps(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		def  run.StepDef
		want errors.Code
	}{
		{name: "no name", def: run.StepDef{Run: noop}, want: errors.Invalid},
		{name: "nothing to run", def: run.StepDef{Name: "ghost"}, want: errors.Invalid},
		{name: "negative retry", def: run.StepDef{Name: "x", Run: noop, Retry: run.RetryPolicy{Max: -1}}, want: errors.Invalid},
		{
			name: "retry above the ceiling",
			def:  run.StepDef{Name: "x", Run: noop, Retry: run.RetryPolicy{Max: run.MaxRetryAttempts + 1}},
			want: errors.Invalid,
		},
		{name: "negative timeout", def: run.StepDef{Name: "x", Run: noop, Timeout: -time.Second}, want: errors.Invalid},
		{
			name: "unknown artifact kind",
			def:  run.StepDef{Name: "x", Run: noop, Produces: []run.ArtifactKind{"poem"}},
			want: errors.Invalid,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if err := run.NewRegistry().Register(tc.def); !errors.IsCode(err, tc.want) {
				t.Fatalf("Register = %v, want %s", err, tc.want)
			}
		})
	}
}

func TestRegistryRejectsADuplicate(t *testing.T) {
	t.Parallel()

	registry := testRegistry(t)
	err := registry.Register(run.StepDef{Name: "validate", Run: noop})
	if !errors.IsCode(err, errors.Conflict) {
		t.Fatalf("Register of a duplicate = %v, want a conflict", err)
	}
	if names := registry.Names(); len(names) != 3 || names[0] != "resolve_context" {
		t.Fatalf("Names = %v", names)
	}
	if _, ok := registry.Lookup("absent"); ok {
		t.Error("Lookup of an unregistered step must report false")
	}
}

func TestValidateRecipe(t *testing.T) {
	t.Parallel()

	registry := testRegistry(t)
	cases := []struct {
		name   string
		recipe []template.StepSpec
		want   errors.Code
	}{
		{
			name: "a complete recipe",
			recipe: []template.StepSpec{
				{Name: "resolve_context", Enabled: true},
				{Name: "generate_body", Enabled: true},
				{Name: "validate", Enabled: true},
			},
		},
		{name: "nothing enabled", recipe: []template.StepSpec{{Name: "validate"}}, want: errors.Invalid},
		{name: "an unknown step", recipe: []template.StepSpec{{Name: "summon", Enabled: true}}, want: errors.NotFound},
		{
			name: "a missing dependency",
			recipe: []template.StepSpec{
				{Name: "resolve_context", Enabled: true},
				{Name: "validate", Enabled: true},
			},
			want: errors.Invalid,
		},
		{
			name: "a step out of order",
			recipe: []template.StepSpec{
				{Name: "generate_body", Enabled: true},
				{Name: "resolve_context", Enabled: true},
			},
			want: errors.Invalid,
		},
		{
			name: "a step twice",
			recipe: []template.StepSpec{
				{Name: "resolve_context", Enabled: true},
				{Name: "resolve_context", Enabled: true},
			},
			want: errors.Conflict,
		},
		{
			name: "a disabled step does not satisfy a dependency",
			recipe: []template.StepSpec{
				{Name: "resolve_context"},
				{Name: "generate_body", Enabled: true},
			},
			want: errors.Invalid,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := run.ValidateRecipe(registry, tc.recipe)
			if tc.want == "" {
				if err != nil {
					t.Fatalf("ValidateRecipe: %v", err)
				}
				return
			}
			if !errors.IsCode(err, tc.want) {
				t.Fatalf("ValidateRecipe = %v, want %s", err, tc.want)
			}
		})
	}
}

func TestPlan(t *testing.T) {
	t.Parallel()

	registry := testRegistry(t)
	recipe := []template.StepSpec{
		{Name: "resolve_context", Enabled: true},
		{Name: "generate_body", Enabled: true, Params: map[string]any{"tone": "plain"}},
		{Name: "validate"},
	}

	definition, err := run.Plan(registry, run.KindGenerate, recipe, time.Hour)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(definition.Steps) != 2 || definition.Steps[1].Name != "generate_body" {
		t.Fatalf("Plan = %+v", definition.Steps)
	}

	index, step, found := definition.StepByName("generate_body")
	if !found || index != 1 || step.Name != "generate_body" {
		t.Fatalf("StepByName = %d, %+v, %v", index, step, found)
	}
	if _, _, found = definition.StepByName("validate"); found {
		t.Error("a disabled step must not be planned")
	}

	if params := run.ParamsFor(recipe, "generate_body"); params["tone"] != "plain" {
		t.Errorf("ParamsFor = %v", params)
	}
	if params := run.ParamsFor(recipe, "absent"); params != nil {
		t.Errorf("ParamsFor of an absent step = %v", params)
	}

	if _, err = run.Plan(registry, run.KindGenerate, recipe, 0); !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("Plan without a deadline = %v", err)
	}
	if _, err = run.Plan(registry, run.KindGenerate, nil, time.Hour); !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("Plan of an empty recipe = %v", err)
	}
}

func TestInputHashIsStableAndSensitive(t *testing.T) {
	t.Parallel()

	params := map[string]any{"allowErrors": true}
	required := []run.Artifact{{Kind: run.ArtifactLinkContext, Hash: "aaa"}, {Kind: run.ArtifactDraft, Hash: "bbb"}}

	first, err := run.InputHash("validate", params, required, 3)
	if err != nil {
		t.Fatalf("InputHash: %v", err)
	}

	reordered, err := run.InputHash("validate", params, []run.Artifact{required[1], required[0]}, 3)
	if err != nil {
		t.Fatalf("InputHash: %v", err)
	}
	if first != reordered {
		t.Error("the input hash must not depend on the order of the required artifacts")
	}

	cases := []struct {
		name     string
		step     string
		params   map[string]any
		required []run.Artifact
		version  int
	}{
		{name: "another step", step: "generate_body", params: params, required: required, version: 3},
		{name: "another parameter", step: "validate", params: map[string]any{"allowErrors": false}, required: required, version: 3},
		{
			name: "another artifact", step: "validate", params: params,
			required: []run.Artifact{{Kind: run.ArtifactLinkContext, Hash: "ccc"}}, version: 3,
		},
		{name: "another template version", step: "validate", params: params, required: required, version: 4},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			other, hashErr := run.InputHash(tc.step, tc.params, tc.required, tc.version)
			if hashErr != nil {
				t.Fatalf("InputHash: %v", hashErr)
			}
			if other == first {
				t.Error("the input hash must change")
			}
		})
	}

	if _, err = run.InputHash("validate", map[string]any{"chan": make(chan int)}, nil, 1); !errors.IsCode(err, errors.Internal) {
		t.Fatalf("InputHash of unencodable parameters = %v", err)
	}
}

func TestStepContextAccessors(t *testing.T) {
	t.Parallel()

	sc := &run.StepContext{
		Params: map[string]any{"allowErrors": true, "iterations": 2},
		Artifacts: map[run.ArtifactKind]run.Artifact{
			run.ArtifactBodyHTML: {Kind: run.ArtifactBodyHTML, Blob: []byte("<p>x</p>")},
			run.ArtifactDraft:    {Kind: run.ArtifactDraft, Purged: true},
		},
	}

	if _, err := sc.Artifact(run.ArtifactBodyHTML); err != nil {
		t.Fatalf("Artifact: %v", err)
	}
	if _, err := sc.Artifact(run.ArtifactMeta); !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("Artifact of a missing kind = %v", err)
	}
	if _, err := sc.Artifact(run.ArtifactDraft); !errors.IsCode(err, errors.NotFound) {
		t.Fatalf("Artifact of a purged kind = %v", err)
	}
	if !sc.BoolParam("allowErrors") || sc.BoolParam("iterations") || sc.BoolParam("absent") {
		t.Error("BoolParam must answer only for a boolean parameter that is set")
	}
	if value, ok := sc.Param("iterations"); !ok || value != 2 {
		t.Errorf("Param = %v, %v", value, ok)
	}
	if _, ok := sc.Param("absent"); ok {
		t.Error("Param of an absent key must report false")
	}
}
