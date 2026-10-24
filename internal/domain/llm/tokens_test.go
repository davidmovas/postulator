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
		{name: "prose by its words", text: "one two three", want: 6},
		{name: "a url by its length", text: "https://shop.example.com/peptides/bpc-157/", want: 12},
		{name: "json by its length", text: `{"price":"12.50","sku":"BPC-5MG"}`, want: 9},
		{name: "cyrillic by its length", text: "пептиды для лаборатории", want: 7},
		{name: "a long paragraph", text: strings.Repeat("word ", 100), want: 170},
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

func TestReadTokensPricesWordsAModelReadsAtThePacksRate(t *testing.T) {
	t.Parallel()

	for words, want := range map[int]int{0: 0, 1: 2, 800: 1360} {
		if got := llm.ReadTokens(words); got != want {
			t.Errorf("ReadTokens(%d) = %d, want %d", words, got, want)
		}
	}
	if llm.ReadTokens(800) >= llm.WordTokens(800) {
		t.Error("a body read is priced at the answer's rate; the input side reads at the measured one")
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
