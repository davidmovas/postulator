package pagemap_test

import (
	"testing"

	"github.com/davidmovas/postulator/internal/domain/pagemap"
)

func TestATypeReasonSaysWhoPlacedThePage(t *testing.T) {
	t.Parallel()

	cases := []struct {
		reason      pagemap.TypeReason
		manual      bool
		rulePlaced  bool
		beforeRules bool
	}{
		{reason: pagemap.TypeByNone},
		{reason: pagemap.TypeByProduct, beforeRules: true},
		{reason: pagemap.TypeByStoreCategory, beforeRules: true},
		{reason: pagemap.TypeByShop, beforeRules: true},
		{reason: pagemap.TypeByTechnical, beforeRules: true},
		{reason: pagemap.TypeByListing, beforeRules: true},
		{reason: pagemap.TypeByDefault, beforeRules: true},
		{reason: pagemap.TypeByKind, beforeRules: true},
		{reason: pagemap.TypeByUser, manual: true},
		{reason: pagemap.TypeByAI, manual: true},
		{reason: pagemap.TypeByImport, manual: true},
		{reason: pagemap.TypeByRule, rulePlaced: true},
		{reason: pagemap.TypeByUnmatched, rulePlaced: true},
	}
	for _, tc := range cases {
		if got := tc.reason.Manual(); got != tc.manual {
			t.Errorf("%q Manual = %t, want %t", tc.reason, got, tc.manual)
		}
		if got := tc.reason.RulePlaced(); got != tc.rulePlaced {
			t.Errorf("%q RulePlaced = %t, want %t", tc.reason, got, tc.rulePlaced)
		}
		if got := tc.reason.BeforeRules(); got != tc.beforeRules {
			t.Errorf("%q BeforeRules = %t, want %t", tc.reason, got, tc.beforeRules)
		}
	}
}
