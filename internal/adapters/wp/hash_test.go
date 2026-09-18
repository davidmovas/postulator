package wp_test

import (
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/wp"
)

func TestContentHashIsPlainSHA256OfTheRawBytes(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		raw  string
		want string
	}{
		{name: "empty", raw: "", want: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"},
		{name: "paragraph", raw: "<p>Koffein ist ein Alkaloid.</p>", want: "119b7cff7356b21d2b00e64d2d3c0589b50f3270b302a94360fac92f1adc332b"},
		{name: "powder", raw: "<p>Powder</p>", want: "79db24b7a931978a1db05ccdb1ae1ed16b07aafa5277af75a7dc1b3b7ec6a509"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := wp.ContentHash(tc.raw); got != tc.want {
				t.Errorf("ContentHash(%q) = %q, want %q", tc.raw, got, tc.want)
			}
		})
	}
}

func TestContentHashDoesNotNormalise(t *testing.T) {
	t.Parallel()

	if wp.ContentHash("<p>a</p>\n") == wp.ContentHash("<p>a</p>") {
		t.Error("a trailing newline must change the hash; the hash is over the raw bytes")
	}
	if wp.ContentHash(" <p>a</p>") == wp.ContentHash("<p>a</p>") {
		t.Error("leading whitespace must change the hash; the hash is over the raw bytes")
	}
}
