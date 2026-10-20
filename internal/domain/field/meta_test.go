package field_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/field"
)

func TestAMetaKeyIsDeniedAlikeOnBothSides(t *testing.T) {
	t.Parallel()

	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "wp-plugin", "testdata", "meta_deny.json"))
	if err != nil {
		t.Fatalf("read the meta deny table: %v", err)
	}
	var table struct {
		Cases []struct {
			Key    string `json:"key"`
			Denied bool   `json:"denied"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(raw, &table); err != nil {
		t.Fatalf("decode the meta deny table: %v", err)
	}
	if len(table.Cases) < 20 {
		t.Fatalf("the table holds %d cases, too few to pin the rule", len(table.Cases))
	}
	for _, tc := range table.Cases {
		t.Run(tc.Key, func(t *testing.T) {
			t.Parallel()

			if allowed := field.MetaKeyAllowed(tc.Key); allowed == tc.Denied {
				t.Errorf("MetaKeyAllowed(%q) = %t, want denied %t", tc.Key, allowed, tc.Denied)
			}
		})
	}
}

func TestTheStoresMetaKeysAreDenied(t *testing.T) {
	t.Parallel()

	denied := field.DeniedMeta()
	for _, key := range []string{
		"_price", "_regular_price", "_sale_price", "_sku", "_stock_status", "_stock", "_manage_stock",
		"_product_attributes", "total_sales",
	} {
		if field.MetaKeyAllowed(key) {
			t.Errorf("%s is allowed, and it is the store's", key)
		}
		found := false
		for _, listed := range denied.Keys {
			found = found || listed == key
		}
		if !found {
			t.Errorf("%s is not listed among the denied keys %v", key, denied.Keys)
		}
	}
	for _, prefix := range []string{"_", "attribute_", "rank_math_"} {
		found := false
		for _, listed := range denied.Prefixes {
			found = found || listed == prefix
		}
		if !found {
			t.Errorf("%s is not listed among the denied prefixes %v", prefix, denied.Prefixes)
		}
	}
	denied.Keys[0] = "changed"
	if field.DeniedMeta().Keys[0] == "changed" {
		t.Error("DeniedMeta hands out the list itself, so a caller can change the deny set")
	}
}
