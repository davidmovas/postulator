package run_test

import (
	"encoding/json"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func TestAnEditNamesOneChangeItCanMake(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		edit run.Edit
		ok   bool
	}{
		{name: "a meta description", edit: run.Edit{Op: run.EditSetField, Field: "seo_description", Value: "New", Hash: "h"}, ok: true},
		{name: "a variation's description", edit: run.Edit{Op: run.EditSetField, Field: "variation_description:4", Value: "New", Hash: "h"}, ok: true},
		{name: "a custom field", edit: run.Edit{Op: run.EditSetField, Field: "meta:dosage", Value: "5 mg", Hash: "h"}, ok: true},
		{name: "a block replaced", edit: run.Edit{Op: run.EditReplace, Index: 2, HTML: "<p>New</p>", Hash: "b"}, ok: true},
		{name: "a block at the start", edit: run.Edit{Op: run.EditInsertAfter, Index: -1, HTML: "<p>New</p>", Hash: "b"}, ok: true},
		{name: "a block deleted", edit: run.Edit{Op: run.EditDelete, Index: 0, Hash: "b"}, ok: true},
		{name: "a link placed", edit: run.Edit{Op: run.EditPlaceLink, Index: 1, TargetID: "p2", Anchor: "peptides", Hash: "b"}, ok: true},
		{name: "an unknown op", edit: run.Edit{Op: "rewrite", Hash: "b"}},
		{name: "a field read empty", edit: run.Edit{Op: run.EditSetField, Field: "excerpt", Value: "x"}, ok: true},
		{name: "a denied field", edit: run.Edit{Op: run.EditSetField, Field: "sku", Value: "X-1", Hash: "h"}},
		{name: "a read-only field", edit: run.Edit{Op: run.EditSetField, Field: "h1", Value: "x", Hash: "h"}},
		{name: "a person's switch", edit: run.Edit{Op: run.EditSetField, Field: "seo_canonical", Value: "x", Hash: "h"}},
		{name: "the body whole", edit: run.Edit{Op: run.EditSetField, Field: "body", Value: "<p>x</p>", Hash: "h"}},
		{name: "a product's description whole", edit: run.Edit{Op: run.EditSetField, Field: "product_description", Value: "x", Hash: "h"}},
		{name: "an empty value", edit: run.Edit{Op: run.EditSetField, Field: "excerpt", Hash: "h"}},
		{name: "a block op without the body's hash", edit: run.Edit{Op: run.EditDelete, Index: 0}},
		{name: "a replacement with no html", edit: run.Edit{Op: run.EditReplace, Index: 0, Hash: "b"}},
		{name: "a block before the first", edit: run.Edit{Op: run.EditReplace, Index: -1, HTML: "<p>x</p>", Hash: "b"}},
		{name: "a link with no anchor", edit: run.Edit{Op: run.EditPlaceLink, Index: 0, TargetID: "p2", Hash: "b"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.edit.Validate()
			if tc.ok && err != nil {
				t.Fatalf("Validate(%+v) = %v, want it accepted", tc.edit, err)
			}
			if !tc.ok && !errors.IsCode(err, errors.Invalid) {
				t.Fatalf("Validate(%+v) = %v, want invalid", tc.edit, err)
			}
		})
	}
}

func TestAnEditTravelsInTheParamsOfItsStep(t *testing.T) {
	t.Parallel()

	sent := run.Edit{Op: run.EditInsertAfter, Index: -1, HTML: "<p>New</p>", Hash: "b"}
	recipe := run.EditRecipe(sent)
	if len(recipe) != 1 || recipe[0].Name != "edit_page" || !recipe[0].Enabled {
		t.Fatalf("EditRecipe = %+v, want the one edit step", recipe)
	}
	stored, err := json.Marshal(recipe[0].Params)
	if err != nil {
		t.Fatalf("encode the params: %v", err)
	}
	var read map[string]any
	if err = json.Unmarshal(stored, &read); err != nil {
		t.Fatalf("decode the params: %v", err)
	}
	got, err := run.EditOf(read)
	if err != nil || got != sent {
		t.Fatalf("EditOf after a round trip = %+v, %v, want %+v", got, err, sent)
	}
	if _, err = run.EditOf(map[string]any{}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("EditOf of nothing = %v, want invalid", err)
	}
}
