package llm

import (
	"strings"
	"unicode/utf8"
)

const (
	tokensPerTenWords = 22
	runesPerToken     = 3
)

func Tokens(text string) int {
	byWords := (len(strings.Fields(text))*tokensPerTenWords + 9) / 10
	byRunes := (utf8.RuneCountInString(text) + runesPerToken - 1) / runesPerToken
	return max(byWords, byRunes)
}

func WordTokens(words int) int {
	return words * tokensPerTenWords / 10
}
