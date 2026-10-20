package field

import (
	"slices"
	"strings"
)

const metaKeyLimit = 255

type MetaDeny struct {
	Keys     []string
	Prefixes []string
}

var deniedMeta = MetaDeny{
	Keys: []string{
		"_price", "_regular_price", "_sale_price", "_sku", "_stock_status", "_stock", "_manage_stock",
		"_product_attributes", "total_sales",
	},
	Prefixes: []string{"_", "attribute_", "rank_math_"},
}

func DeniedMeta() MetaDeny {
	return MetaDeny{Keys: slices.Clone(deniedMeta.Keys), Prefixes: slices.Clone(deniedMeta.Prefixes)}
}

func MetaKeyAllowed(name string) bool {
	return metaKeyWellFormed(name) && !metaKeyDenied(name)
}

func metaKeyWellFormed(name string) bool {
	if name == "" || len(name) > metaKeyLimit {
		return false
	}
	for i := range len(name) {
		switch c := name[i]; {
		case c == '_', c == '-', c >= '0' && c <= '9', c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z':
		default:
			return false
		}
	}
	return true
}

func metaKeyDenied(name string) bool {
	folded := strings.ToLower(name)
	return slices.Contains(deniedMeta.Keys, folded) ||
		slices.ContainsFunc(deniedMeta.Prefixes, func(prefix string) bool { return strings.HasPrefix(folded, prefix) })
}
