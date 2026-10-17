package wptest_test

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
)

func TestTheRenderedRouteAnswersTheFilteredContent(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	seeded := server.Seed(
		wptest.Item{Type: wptest.TypePage, Title: "Hub", Content: "", Filtered: `<nav aria-label="Breadcrumb">Home</nav><section><h2>Products</h2></section>`},
		wptest.Item{Type: wptest.TypePage, Title: "Plain", Content: "<p>Plain.</p>"},
	)

	cases := []struct {
		name string
		want string
		id   int64
	}{
		{name: "a page the site adds to", id: seeded[0].ID, want: `<nav aria-label="Breadcrumb">Home</nav><section><h2>Products</h2></section>`},
		{name: "a page nothing adds to", id: seeded[1].ID, want: "<p>Plain.</p>"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			response, payload := call(t, server, http.MethodGet, "/wp-json/postulator/v1/content/"+itoa(tc.id)+"/rendered", nil, true)
			if response.StatusCode != http.StatusOK {
				t.Fatalf("status = %d: %s", response.StatusCode, payload)
			}
			var got struct {
				Type        string `json:"type"`
				Content     string `json:"content"`
				ContentHash string `json:"contentHash"`
				ID          int64  `json:"id"`
			}
			decode(t, payload, &got)
			sum := sha256.Sum256([]byte(tc.want))
			if got.ID != tc.id || got.Type != wptest.TypePage || got.Content != tc.want || got.ContentHash != hex.EncodeToString(sum[:]) {
				t.Errorf("rendered = %+v, want %q", got, tc.want)
			}
		})
	}

	response, _ := call(t, server, http.MethodGet, "/wp-json/postulator/v1/content/9999/rendered", nil, true)
	if response.StatusCode != http.StatusNotFound {
		t.Errorf("an unknown id = %d, want 404", response.StatusCode)
	}
}
