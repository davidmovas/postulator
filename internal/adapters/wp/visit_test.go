package wp_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func TestVisitReadsAPageAsAVisitorSeesIt(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	server.Seed(wptest.Item{Type: wptest.TypeProduct, Title: "Powder", Slug: "powder", Status: "publish", Content: "<p>Made by hand.</p>"})

	visit, err := newClient(t, server).Visit(t.Context(), server.URL()+"/product/powder/")
	if err != nil {
		t.Fatalf("Visit: %v", err)
	}
	if visit.Status != http.StatusOK || !strings.Contains(visit.Body, "<p>Made by hand.</p>") {
		t.Errorf("visit = %d %q", visit.Status, visit.Body)
	}
	recorded, _ := server.LastRequest()
	if recorded.Header.Get("Authorization") != "" {
		t.Error("the visit carried the application password; a visitor has none")
	}
	if !strings.Contains(recorded.Header.Get("Accept"), "text/html") {
		t.Errorf("accept = %q, want a page", recorded.Header.Get("Accept"))
	}
}

func TestVisitRefusesWhatIsNotAPageOfTheSite(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	client := newClient(t, server)

	cases := []struct {
		name string
		link string
		want errors.Code
	}{
		{name: "another host", link: "https://elsewhere.example.com/product/powder/", want: errors.Invalid},
		{name: "a relative address", link: "/product/powder/", want: errors.Invalid},
		{name: "a page the site does not hold", link: server.URL() + "/product/missing/", want: errors.NotFound},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if _, err := client.Visit(t.Context(), tc.link); !errors.IsCode(err, tc.want) {
				t.Fatalf("Visit = %v, want %s", err, tc.want)
			}
		})
	}
}
