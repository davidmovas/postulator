package pagemap_test

import (
	"testing"

	"github.com/davidmovas/postulator/internal/domain/pagemap"
)

func TestStatusFromWordPressCollapsesTheEditableStatuses(t *testing.T) {
	t.Parallel()

	cases := map[string]pagemap.Status{
		"publish": pagemap.StatusPublished,
		"draft":   pagemap.StatusExists,
		"pending": pagemap.StatusExists,
		"private": pagemap.StatusExists,
		"future":  pagemap.StatusExists,
	}

	for wordpress, want := range cases {
		if got := pagemap.StatusFromWordPress(wordpress); got != want {
			t.Errorf("StatusFromWordPress(%q) = %q, want %q", wordpress, got, want)
		}
	}
}
