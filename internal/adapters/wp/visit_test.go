package wp_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/wp"
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

func TestVisitNamesWhereItLanded(t *testing.T) {
	t.Parallel()

	server := wptest.New(t, wptest.WithPublicRedirect("/checkout/", "/cart/"))
	server.Seed(
		wptest.Item{Type: wptest.TypePage, Title: "Cart", Slug: "cart", Status: "publish", Content: "<p>Your cart is empty.</p>"},
		wptest.Item{Type: wptest.TypePage, Title: "Checkout", Slug: "checkout", Status: "publish", Content: "<p>Pay here.</p>"},
	)
	client := newClient(t, server)

	cases := []struct {
		name     string
		path     string
		wantPath string
		wantBody string
	}{
		{name: "a visit that was redirected", path: "/checkout/", wantPath: "/cart/", wantBody: "Your cart is empty."},
		{name: "a visit that stayed", path: "/cart/", wantPath: "/cart/", wantBody: "Your cart is empty."},
	}
	for _, tc := range cases {
		visit, err := client.Visit(t.Context(), server.URL()+tc.path)
		if err != nil {
			t.Fatalf("%s: Visit: %v", tc.name, err)
		}
		landed, err := url.Parse(visit.URL)
		if err != nil {
			t.Fatalf("%s: the landing URL %q: %v", tc.name, visit.URL, err)
		}
		if landed.Host != strings.TrimPrefix(server.URL(), "http://") || landed.Path != tc.wantPath {
			t.Errorf("%s: landed on %q, want %s on the site", tc.name, visit.URL, tc.wantPath)
		}
		if visit.Status != http.StatusOK || !strings.Contains(visit.Body, tc.wantBody) {
			t.Errorf("%s: visit = %d %q", tc.name, visit.Status, visit.Body)
		}
	}
}

func TestVisitAsksForAFreshRender(t *testing.T) {
	t.Parallel()

	type seen struct {
		path, query, cacheControl string
	}
	var (
		mu       sync.Mutex
		recorded []seen
	)
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		recorded = append(recorded, seen{path: r.URL.EscapedPath(), query: r.URL.RawQuery, cacheControl: r.Header.Get("Cache-Control")})
		mu.Unlock()
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if _, err := w.Write([]byte("<!doctype html><html><body><main>page</main></body></html>")); err != nil {
			t.Errorf("write the page: %v", err)
		}
	}))
	t.Cleanup(site.Close)

	client, err := wp.New(wp.Config{BaseURL: site.URL, Username: wptest.DefaultUser, AppPassword: wptest.DefaultPassword, AllowInsecure: true},
		wp.WithRateLimit(0), wp.WithBackoff(func(int) time.Duration { return 0 }))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	cases := []struct {
		name   string
		link   string
		path   string
		prefix string
	}{
		{name: "a bare address", link: site.URL + "/guide/setup/", path: "/guide/setup/"},
		{name: "an address with its own query", link: site.URL + "/?page_id=5&preview=true&preview_token=a%2Fb", path: "/", prefix: "page_id=5&preview=true&preview_token=a%2Fb&"},
		{name: "an escaped path", link: site.URL + "/caf%C3%A9/?b=2&a=1", path: "/caf%C3%A9/", prefix: "b=2&a=1&"},
	}
	for _, tc := range cases {
		mu.Lock()
		before := len(recorded)
		mu.Unlock()

		visit, err := client.Visit(t.Context(), tc.link)
		if err != nil {
			t.Fatalf("%s: Visit: %v", tc.name, err)
		}
		if visit.Status != http.StatusOK {
			t.Fatalf("%s: status = %d", tc.name, visit.Status)
		}

		mu.Lock()
		got := recorded[before:]
		mu.Unlock()
		if len(got) != 1 {
			t.Fatalf("%s: %d requests reached the site, want 1", tc.name, len(got))
		}
		if got[0].path != tc.path {
			t.Errorf("%s: path = %q, want %q", tc.name, got[0].path, tc.path)
		}
		nonce := regexp.MustCompile(`^` + regexp.QuoteMeta(tc.prefix) + `_=[1-9][0-9]*$`)
		if !nonce.MatchString(got[0].query) {
			t.Errorf("%s: query = %q, want %q kept and a _=<unix nanos> nonce after it", tc.name, got[0].query, tc.prefix)
		}
		if got[0].cacheControl != "no-cache" {
			t.Errorf("%s: Cache-Control = %q, want no-cache", tc.name, got[0].cacheControl)
		}
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
