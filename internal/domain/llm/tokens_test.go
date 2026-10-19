package llm_test

import (
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/llm"
)

func TestTokensCountsWordsAndLongRuns(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		text string
		want int
	}{
		{name: "nothing", text: "", want: 0},
		{name: "blank", text: " \n\t ", want: 2},
		{name: "prose by its words", text: "one two three", want: 7},
		{name: "a url by its length", text: "https://shop.example.com/peptides/bpc-157/", want: 14},
		{name: "json by its length", text: `{"price":"12.50","sku":"BPC-5MG"}`, want: 11},
		{name: "cyrillic by its length", text: "пептиды для лаборатории", want: 8},
		{name: "a long paragraph", text: strings.Repeat("word ", 100), want: 220},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := llm.Tokens(tc.text); got != tc.want {
				t.Errorf("Tokens(%q) = %d, want %d", tc.text, got, tc.want)
			}
		})
	}
}

func TestWordTokensIsTheEstimatesRate(t *testing.T) {
	t.Parallel()

	cases := []struct {
		words int
		want  int
	}{
		{words: 0, want: 0},
		{words: 1, want: 2},
		{words: 800, want: 1760},
	}
	for _, tc := range cases {
		if got := llm.WordTokens(tc.words); got != tc.want {
			t.Errorf("WordTokens(%d) = %d, want %d", tc.words, got, tc.want)
		}
	}
}
