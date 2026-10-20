package contextpack

import (
	"cmp"
	"encoding/json"
	"slices"
	"strconv"
	"strings"

	"github.com/davidmovas/postulator/internal/domain/field"
)

type Variation struct {
	Key         field.Key `json:"key"`
	Options     []string  `json:"options"`
	Description string    `json:"description"`
	Ours        bool      `json:"ours"`
}

type variationFacts struct {
	Attributes []struct {
		Name   string `json:"name"`
		Option string `json:"option"`
	} `json:"attributes"`
}

func Variations(values []field.Value) []Variation {
	descriptions := make(map[string]field.Value)
	for i := range values {
		if values[i].Key.Kind() == field.VariationDescription {
			descriptions[values[i].Key.Instance()] = values[i]
		}
	}
	out := make([]Variation, 0)
	for i := range values {
		if values[i].Key.Kind() != field.Variation {
			continue
		}
		instance := values[i].Key.Instance()
		described := descriptions[instance]
		out = append(out, Variation{
			Key: field.Instance(field.VariationDescription, instance), Options: optionsOf(values[i].Current),
			Description: described.Current,
			Ours:        described.WrittenRunID != nil && described.CurrentHash == described.BaseHash,
		})
	}
	slices.SortFunc(out, func(a, b Variation) int {
		return cmp.Or(cmp.Compare(instanceNumber(a.Key), instanceNumber(b.Key)), strings.Compare(string(a.Key), string(b.Key)))
	})
	return out
}

func instanceNumber(key field.Key) int64 {
	number, err := strconv.ParseInt(key.Instance(), 10, 64)
	if err != nil {
		return 0
	}
	return number
}

func optionsOf(facts string) []string {
	options := make([]string, 0)
	var held variationFacts
	if json.Unmarshal([]byte(facts), &held) != nil {
		return options
	}
	for _, attribute := range held.Attributes {
		name, option := strings.TrimSpace(attribute.Name), strings.TrimSpace(attribute.Option)
		if name != "" && option != "" {
			options = append(options, name+": "+option)
		}
	}
	return options
}
