package dto_test

import (
	"encoding/json"
	"testing"

	"github.com/davidmovas/postulator/internal/kernel/dto"
)

func TestCategoryJSON(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   dto.Category
		want string
	}{
		{
			name: "a category the site has",
			in:   dto.Category{ID: "1a1a1a1a-1a1a-4a1a-8a1a-1a1a1a1a1a1a", Name: "Healing", TermID: new(int64(14))},
			want: `{"id":"1a1a1a1a-1a1a-4a1a-8a1a-1a1a1a1a1a1a","name":"Healing","termId":14}`,
		},
		{
			name: "a category the site does not have yet",
			in:   dto.Category{ID: "2b2b2b2b-2b2b-4b2b-8b2b-2b2b2b2b2b2b", Name: "BPC-157"},
			want: `{"id":"2b2b2b2b-2b2b-4b2b-8b2b-2b2b2b2b2b2b","name":"BPC-157"}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			encoded, err := json.Marshal(tc.in)
			if err != nil {
				t.Fatalf("encode: %v", err)
			}
			if string(encoded) != tc.want {
				t.Fatalf("Category = %s, want %s", encoded, tc.want)
			}

			var back dto.Category
			if err = json.Unmarshal(encoded, &back); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if back.ID != tc.in.ID || back.Name != tc.in.Name || (back.TermID == nil) != (tc.in.TermID == nil) ||
				(back.TermID != nil && *back.TermID != *tc.in.TermID) {
				t.Fatalf("round trip = %+v, want %+v", back, tc.in)
			}
		})
	}
}
