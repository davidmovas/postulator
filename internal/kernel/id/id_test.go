package id_test

import (
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/kernel/id"
)

func TestNewIsValid(t *testing.T) {
	t.Parallel()

	seen := make(map[string]struct{}, 1000)
	for range 1000 {
		value := id.New()
		if !id.Valid(value) {
			t.Fatalf("New() produced %q, which Valid rejects", value)
		}
		if value != strings.ToLower(value) {
			t.Fatalf("New() produced %q, which is not lowercase", value)
		}
		if _, clash := seen[value]; clash {
			t.Fatalf("New() produced %q twice", value)
		}
		seen[value] = struct{}{}
	}
}

func TestValid(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   string
		want bool
	}{
		{name: "canonical v4", in: "7c9e6679-7425-40de-944b-e07fc1f90ae7", want: true},
		{name: "uppercase v4", in: "7C9E6679-7425-40DE-944B-E07FC1F90AE7", want: true},
		{name: "variant 8", in: "7c9e6679-7425-40de-844b-e07fc1f90ae7", want: true},
		{name: "variant a", in: "7c9e6679-7425-40de-a44b-e07fc1f90ae7", want: true},
		{name: "variant b", in: "7c9e6679-7425-40de-b44b-e07fc1f90ae7", want: true},
		{name: "empty", in: "", want: false},
		{name: "nonsense", in: "x", want: false},
		{name: "version 7", in: "0199a5b1-0b3f-7c3a-9b1e-2f6a1c0d4e5f", want: false},
		{name: "version 1", in: "7c9e6679-7425-10de-944b-e07fc1f90ae7", want: false},
		{name: "nil uuid", in: "00000000-0000-0000-0000-000000000000", want: false},
		{name: "reserved variant", in: "7c9e6679-7425-40de-c44b-e07fc1f90ae7", want: false},
		{name: "unhyphenated", in: "7c9e667974254 0de944be07fc1f90ae7", want: false},
		{name: "compact", in: "7c9e6679742540de944be07fc1f90ae7", want: false},
		{name: "braced", in: "{7c9e6679-7425-40de-944b-e07fc1f90ae7}", want: false},
		{name: "urn", in: "urn:uuid:7c9e6679-7425-40de-944b-e07fc1f90ae7", want: false},
		{name: "trailing space", in: "7c9e6679-7425-40de-944b-e07fc1f90ae7 ", want: false},
		{name: "non hex digit", in: "7c9e6679-7425-40de-944b-e07fc1f90azz", want: false},
		{name: "misplaced hyphen", in: "7c9e66797-425-40de-944b-e07fc1f90ae7", want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := id.Valid(tc.in); got != tc.want {
				t.Fatalf("Valid(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}
