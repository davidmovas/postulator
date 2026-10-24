package llm

import (
	"strings"
	"unicode/utf8"
)

const (
	tokensPerTenWords      = 22
	readTokensPerTenWords  = 17
	readRunesPerTenTokens  = 37
	readRunesRoundingSlack = readRunesPerTenTokens - 1
)

func Tokens(text string) int {
	byWords := (len(strings.Fields(text))*readTokensPerTenWords + 9) / 10
	byRunes := (utf8.RuneCountInString(text)*10 + readRunesRoundingSlack) / readRunesPerTenTokens
	return max(byWords, byRunes)
}

func WordTokens(words int) int {
	return words * tokensPerTenWords / 10
}

func ReadTokens(words int) int {
	return (words*readTokensPerTenWords + 9) / 10
}
