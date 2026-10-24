package content_test

import (
	"slices"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/content"
)

func TestWordsOfFoldsCaseAndDropsPunctuation(t *testing.T) {
	t.Parallel()

	got := content.WordsOf("BPC‑157 — “Healing” Peptides… ships cold, sealed; 2mg/5mg!")
	want := []string{"bpc", "157", "healing", "peptides", "ships", "cold", "sealed", "2mg", "5mg"}
	if !slices.Equal(got, want) {
		t.Errorf("WordsOf = %q, want %q", got, want)
	}
	if len(content.WordsOf("  …  ")) != 0 {
		t.Error("punctuation alone yields words")
	}
}
