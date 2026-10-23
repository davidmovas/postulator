package main

import (
	"slices"
	"testing"

	"github.com/tiktoken-go/tokenizer"

	"github.com/davidmovas/postulator/internal/domain/llm"
)

func TestTheCounterAgreesWithTheEncodingsOwnVectors(t *testing.T) {
	t.Parallel()

	cases := []struct {
		text string
		ids  []uint
	}{
		{text: "", ids: nil},
		{text: "'RE", ids: []uint{6, 1099}},
		{text: "hello world", ids: []uint{24912, 2375}},
		{text: "hello  world", ids: []uint{24912, 220, 2375}},
		{text: "hello   world", ids: []uint{24912, 256, 2375}},
		{text: "supercalifragilistic", ids: []uint{17789, 5842, 366, 17764, 311, 6207}},
		{text: "We know what we are, but know not what we may be.", ids: []uint{2167, 1761, 1412, 581, 553, 11, 889, 1761, 625, 1412, 581, 1340, 413, 13}},
	}

	counter, err := newCounter(tokenizer.O200kBase)
	if err != nil {
		t.Fatalf("newCounter: %v", err)
	}
	for _, tc := range cases {
		t.Run(tc.text, func(t *testing.T) {
			t.Parallel()

			ids, _, err := counter.codec.Encode(tc.text)
			if err != nil {
				t.Fatalf("encode %q: %v", tc.text, err)
			}
			if !slices.Equal(ids, tc.ids) {
				t.Errorf("o200k_base encodes %q as %v, want %v", tc.text, ids, tc.ids)
			}
			if got, err := counter.count(tc.text); err != nil || got != len(tc.ids) {
				t.Errorf("count(%q) = %d, %v; want %d", tc.text, got, err, len(tc.ids))
			}
		})
	}
}

func TestEveryModelHasAnEncodingAndAnUnknownOneSaysSo(t *testing.T) {
	t.Parallel()

	cases := []struct {
		model    string
		fallback bool
	}{
		{model: "gpt-5.6-terra"},
		{model: "gpt-5.6-luna"},
		{model: "gpt-5.6-sol"},
		{model: "gpt-5"},
		{model: "gpt-4.1-mini"},
		{model: "gpt-4o"},
		{model: "chatgpt-4o-latest"},
		{model: "o3"},
		{model: "o4-mini"},
		{model: "gpt-image-2", fallback: true},
		{model: "davinci", fallback: true},
	}
	for _, tc := range cases {
		t.Run(tc.model, func(t *testing.T) {
			t.Parallel()

			got := encodingFor(llm.ModelRef{Provider: "openai", Model: tc.model})
			if got.encoding != tokenizer.O200kBase || got.fallback != tc.fallback {
				t.Errorf("encodingFor(%s) = %+v, want o200k_base with fallback %v", tc.model, got, tc.fallback)
			}
		})
	}
}

func TestTheCatalogsModelsAreEachMapped(t *testing.T) {
	t.Parallel()

	mapped, err := catalogEncodings(t.Context())
	if err != nil {
		t.Fatalf("catalogEncodings: %v", err)
	}
	fallbacks := map[string]bool{}
	for _, row := range mapped {
		fallbacks[row.model] = row.fallback
	}
	want := map[string]bool{"gpt-5.6-luna": false, "gpt-5.6-sol": false, "gpt-5.6-terra": false, "gpt-image-2": true}
	if len(fallbacks) != len(want) {
		t.Fatalf("the catalog maps %v, want %v", fallbacks, want)
	}
	for model, fallback := range want {
		if got, ok := fallbacks[model]; !ok || got != fallback {
			t.Errorf("%s maps with fallback %v (present %v), want %v", model, got, ok, fallback)
		}
	}
}
