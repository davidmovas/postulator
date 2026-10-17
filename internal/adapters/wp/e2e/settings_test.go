//go:build e2e

package e2e_test

import (
	"fmt"
	"net/http"
	"testing"
)

const settingsPath = "/wp-json/postulator/v1/settings"

type technicalPages struct {
	ShowOnFront string `json:"showOnFront"`
	Pages       struct {
		Front    int `json:"front"`
		Posts    int `json:"posts"`
		Privacy  int `json:"privacy"`
		Shop     int `json:"shop"`
		Cart     int `json:"cart"`
		Checkout int `json:"checkout"`
		Account  int `json:"account"`
		Terms    int `json:"terms"`
	} `json:"pages"`
}

func pageSlug(t *testing.T, c *client, id int) string {
	t.Helper()

	var page struct {
		Slug string `json:"slug"`
	}
	c.expect(t, http.MethodGet, fmt.Sprintf("/wp-json/wp/v2/pages/%d?context=edit", id), nil, http.StatusOK, &page)
	return page.Slug
}

func TestSettingsNameTheTechnicalPages(t *testing.T) {
	c, env := newClient(t)
	stand := standOf(t, env)

	var got technicalPages
	c.expect(t, http.MethodGet, settingsPath, nil, http.StatusOK, &got)
	if got.ShowOnFront != stand.frontPage {
		t.Errorf("showOnFront = %q, want %q", got.ShowOnFront, stand.frontPage)
	}
	switch stand.frontPage {
	case "page":
		if got.Pages.Front == 0 || pageSlug(t, c, got.Pages.Front) != stand.frontPageSlug {
			t.Errorf("the front page = %d, want the page %s", got.Pages.Front, stand.frontPageSlug)
		}
	default:
		if got.Pages.Front != 0 || got.Pages.Posts != 0 {
			t.Errorf("a front page of posts names the pages %d and %d, want none", got.Pages.Front, got.Pages.Posts)
		}
	}
	if got.Pages.Privacy == 0 || pageSlug(t, c, got.Pages.Privacy) == "" {
		t.Errorf("the privacy page = %d, want the page WordPress created", got.Pages.Privacy)
	}

	if env.woo {
		for name, id := range map[string]int{"shop": got.Pages.Shop, "cart": got.Pages.Cart, "checkout": got.Pages.Checkout, "my-account": got.Pages.Account} {
			if id == 0 || pageSlug(t, c, id) != name {
				t.Errorf("the %s page = %d, want the store's page", name, id)
			}
		}
	}

	if status, _, _ := c.anonymous(t, env.baseURL+settingsPath); status != http.StatusUnauthorized {
		t.Errorf("an anonymous read = %d, want 401", status)
	}
}
