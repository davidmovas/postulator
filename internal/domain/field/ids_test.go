package field_test

import (
	"slices"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/field"
)

func TestAnIDListIsWrittenInItsOrderAndReadBack(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		ids    []int64
		text   string
		parsed []int64
		valid  bool
	}{
		{name: "an empty list", ids: []int64{}, text: "", parsed: []int64{}, valid: true},
		{name: "the store's order", ids: []int64{15, 12}, text: "15,12", parsed: []int64{15, 12}, valid: true},
		{name: "a repeat is dropped", ids: []int64{12, 12, 15}, text: "12,15", parsed: []int64{12, 15}, valid: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := field.IDList(tc.ids); got != tc.text {
				t.Errorf("IDList(%v) = %q, want %q", tc.ids, got, tc.text)
			}
			parsed, ok := field.ParseIDList(tc.text)
			if ok != tc.valid || !slices.Equal(parsed, tc.parsed) {
				t.Errorf("ParseIDList(%q) = %v %t, want %v", tc.text, parsed, ok, tc.parsed)
			}
		})
	}
	for _, text := range []string{"12, 15", "12,x", "-3", "0", ",12"} {
		if _, ok := field.ParseIDList(text); ok {
			t.Errorf("ParseIDList(%q) read a list it should refuse", text)
		}
	}
}
