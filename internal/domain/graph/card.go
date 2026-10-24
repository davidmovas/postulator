package graph

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
)

func (e Entity) CardHash() string {
	var card strings.Builder
	definition := strings.TrimSpace(e.Definition)
	card.WriteString(strconv.Itoa(len(definition)))
	card.WriteString(":")
	card.WriteString(definition)
	for _, fact := range CleanFacts(e.Facts) {
		for _, part := range []string{fact.Label, fact.Value} {
			card.WriteString(strconv.Itoa(len(part)))
			card.WriteString(":")
			card.WriteString(part)
		}
	}
	sum := sha256.Sum256([]byte(card.String()))
	return hex.EncodeToString(sum[:])
}
