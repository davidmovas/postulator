package site_test

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/site"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func TestDefaultSettingsAreFilledIn(t *testing.T) {
	t.Parallel()

	got := site.Settings{}.Normalized()
	if got.LinkURLForm != site.URLFormAuto {
		t.Errorf("link url form = %q, want auto", got.LinkURLForm)
	}
	if got.Observe.RenderConcurrency != 4 || got.Observe.FullWalkEvery != 10 {
		t.Errorf("observe = %+v, want 4 renders at a time and a full walk every 10", got.Observe)
	}
	names := make([]string, 0, len(got.Builders))
	for _, rule := range got.Builders {
		names = append(names, rule.Name)
	}
	for _, want := range []string{"elementor", "wpbakery", "divi", "beaver-builder", "oxygen", "bricks", "breakdance"} {
		if !slices.Contains(names, want) {
			t.Errorf("the builders lack %s: %v", want, names)
		}
	}
	for _, want := range []string{"woocommerce_cart", "woocommerce_checkout", "woocommerce_my_account"} {
		if !slices.Contains(got.TechnicalShortcodes, want) {
			t.Errorf("the technical shortcodes lack %s", want)
		}
	}
	for _, want := range []string{"woocommerce/cart", "woocommerce/checkout"} {
		if !slices.Contains(got.TechnicalBlocks, want) {
			t.Errorf("the technical blocks lack %s", want)
		}
	}

	kept := site.Settings{UserAgent: "Agent/1", Observe: site.ObserveSettings{RenderConcurrency: 2}}.Normalized()
	if kept.UserAgent != "Agent/1" || kept.Observe.RenderConcurrency != 2 || kept.Observe.FullWalkEvery != 10 {
		t.Errorf("a set value was replaced: %+v", kept)
	}
	if err := got.Validate(); err != nil {
		t.Errorf("the defaults do not validate: %v", err)
	}
}

func TestSettingsRefuseAnUnknownURLFormOrConcurrency(t *testing.T) {
	t.Parallel()

	if err := (site.Settings{LinkURLForm: "sometimes"}).Normalized().Validate(); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("an unknown url form is accepted: %v", err)
	}
	if err := (site.Settings{Observe: site.ObserveSettings{RenderConcurrency: 17}}).Normalized().Validate(); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("17 renders at a time is accepted: %v", err)
	}
	if err := (site.Settings{Observe: site.ObserveSettings{FullWalkEvery: -1}}).Normalized().Validate(); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("a negative cadence is accepted: %v", err)
	}
	for _, form := range []site.URLForm{site.URLFormAuto, site.URLFormAbsolute, site.URLFormRelative} {
		if err := (site.Settings{LinkURLForm: form}).Normalized().Validate(); err != nil {
			t.Errorf("%s is refused: %v", form, err)
		}
	}
}

func TestSettingsMarshalInCamelCase(t *testing.T) {
	t.Parallel()

	raw, err := json.Marshal(site.DefaultSettings())
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	for _, key := range []string{`"linkUrlForm"`, `"userAgent"`, `"observe"`, `"renderConcurrency"`, `"fullWalkEvery"`, `"builders"`, `"dataAttributes"`, `"classPrefixes"`, `"shortcodePrefixes"`, `"technicalShortcodes"`, `"technicalBlocks"`, `"technical"`, `"observation"`, `"lastStartedAt"`} {
		if !strings.Contains(string(raw), key) {
			t.Errorf("the settings JSON lacks %s: %s", key, raw)
		}
	}
	var back site.Settings
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if back.Normalized().Observe.RenderConcurrency != 4 {
		t.Error("the settings do not round trip")
	}
}
