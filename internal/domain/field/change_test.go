package field_test

import (
	"testing"

	"github.com/davidmovas/postulator/internal/domain/field"
)

func TestAChangeCarriesTheHashesOfBothSides(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name          string
		before, after string
		moves         bool
	}{
		{name: "a new title", before: "Espresso", after: "Espresso guide", moves: true},
		{name: "the same title", before: "Espresso", after: "Espresso"},
		{name: "a field filled for the first time", before: "", after: "Pull a shot.", moves: true},
	}
	for _, tc := range cases {
		change := field.NewChange(field.Title, tc.before, tc.after)
		if change.BeforeHash != field.Hash(tc.before) || change.AfterHash != field.Hash(tc.after) {
			t.Errorf("%s: hashes %q %q, want the values' own", tc.name, change.BeforeHash, change.AfterHash)
		}
		if change.Moves() != tc.moves {
			t.Errorf("%s: Moves = %v, want %v", tc.name, change.Moves(), tc.moves)
		}
	}
}
