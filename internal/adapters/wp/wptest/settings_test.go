package wptest_test

import (
	"net/http"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
)

func TestThePluginSettingsNameTheTechnicalPages(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		showOnFront string
		front       int64
		posts       int64
	}{
		{name: "a static front page", showOnFront: "page", front: 12, posts: 14},
		{name: "a front page of posts", showOnFront: "posts"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server := wptest.New(t)
			server.SetReading(tc.showOnFront, 12, 14)
			server.SetPrivacyPage(3)
			server.SetStorePages(wptest.StorePages{Shop: 5, Cart: 6, Checkout: 7, Account: 8})

			response, payload := call(t, server, http.MethodGet, "/wp-json/postulator/v1/settings", nil, true)
			if response.StatusCode != http.StatusOK {
				t.Fatalf("status = %d: %s", response.StatusCode, payload)
			}
			var got struct {
				ShowOnFront string           `json:"showOnFront"`
				Pages       map[string]int64 `json:"pages"`
			}
			decode(t, payload, &got)
			want := map[string]int64{"front": tc.front, "posts": tc.posts, "privacy": 3, "shop": 5, "cart": 6, "checkout": 7, "account": 8, "terms": 0}
			if got.ShowOnFront != tc.showOnFront || len(got.Pages) != len(want) {
				t.Fatalf("settings = %s", payload)
			}
			for name, id := range want {
				if got.Pages[name] != id {
					t.Errorf("%s = %d, want %d", name, got.Pages[name], id)
				}
			}
		})
	}

	bare := wptest.New(t, wptest.WithoutCommerce())
	response, payload := call(t, bare, http.MethodGet, "/wp-json/postulator/v1/settings", nil, true)
	var got struct {
		Pages map[string]int64 `json:"pages"`
	}
	decode(t, payload, &got)
	if response.StatusCode != http.StatusOK || got.Pages["shop"] != 0 || got.Pages["cart"] != 0 {
		t.Errorf("a site without a store answers %d %s, want no store page", response.StatusCode, payload)
	}
}
