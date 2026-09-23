package imports_test

import (
	"encoding/json"
	"testing"

	"github.com/davidmovas/postulator/internal/application/imports"
	"github.com/davidmovas/postulator/internal/kernel/settings"
)

func TestMaxRowsCarriesItsDefaultAndItsSetting(t *testing.T) {
	t.Parallel()

	values := settings.Default().NewValues()
	if got := imports.MaxRows(values); got != imports.DefaultMaxRows {
		t.Fatalf("MaxRows = %d, want %d", got, imports.DefaultMaxRows)
	}

	unknown, err := settings.Default().Apply(values, map[string]json.RawMessage{"import.maxRows": json.RawMessage(`500`)})
	if err != nil || len(unknown) != 0 {
		t.Fatalf("Apply = %v, unknown = %v", err, unknown)
	}
	if got := imports.MaxRows(values); got != 500 {
		t.Fatalf("MaxRows = %d, want 500", got)
	}
}
