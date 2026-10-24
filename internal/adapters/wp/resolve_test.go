package wp_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const resolveAgent = "Agent/7"

type hop struct {
	status   int
	location string
	headless bool
}

type seenHop struct {
	method        string
	path          string
	authorization string
	userAgent     string
}

type redirectSite struct {
	server *httptest.Server
	mu     sync.Mutex
	seen   []seenHop
}

func newRedirectSite(t *testing.T, hops func(base string) map[string]hop) *redirectSite {
	t.Helper()

	site := &redirectSite{}
	var routes map[string]hop
	site.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		site.mu.Lock()
		site.seen = append(site.seen, seenHop{
			method: r.Method, path: r.URL.Path, authorization: r.Header.Get("Authorization"), userAgent: r.Header.Get("User-Agent"),
		})
		site.mu.Unlock()

		route, ok := routes[r.URL.Path]
		switch {
		case !ok:
			w.WriteHeader(http.StatusNotFound)
		case route.headless && r.Method == http.MethodHead:
			w.WriteHeader(route.status)
		case route.headless:
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte("<html><body>a page read in full</body></html>")); err != nil {
				t.Errorf("write the page: %v", err)
			}
		default:
			if route.location != "" {
				w.Header().Set("Location", route.location)
			}
			w.WriteHeader(route.status)
		}
	}))
	t.Cleanup(site.server.Close)
	routes = hops(site.server.URL)
	return site
}

func (s *redirectSite) hops() []seenHop {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.seen)
}

func (s *redirectSite) client(t *testing.T, opts ...wp.Option) *wp.Client {
	t.Helper()

	options := append([]wp.Option{
		wp.WithRateLimit(0), wp.WithBackoff(func(int) time.Duration { return 0 }), wp.WithUserAgent(resolveAgent),
	}, opts...)
	client, err := wp.New(wp.Config{BaseURL: s.server.URL, Username: wptest.DefaultUser, AppPassword: wptest.DefaultPassword, AllowInsecure: true}, options...)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return client
}

func TestResolveFollowsTheSiteWithoutABody(t *testing.T) {
	t.Parallel()

	const elsewhere = "https://elsewhere.example/landing/"
	cases := []struct {
		name        string
		start       string
		hops        func(base string) map[string]hop
		want        func(base string) wp.Resolution
		wantMethods []string
	}{
		{
			name:  "a chain of redirects on the site to a page",
			start: "/category/healing/",
			hops: func(base string) map[string]hop {
				return map[string]hop{
					"/category/healing/":         {status: http.StatusMovedPermanently, location: "/healing/"},
					"/healing/":                  {status: http.StatusFound, location: base + "/product-category/healing/"},
					"/product-category/healing/": {status: http.StatusOK},
				}
			},
			want: func(base string) wp.Resolution {
				return wp.Resolution{URL: base + "/product-category/healing/", Status: http.StatusOK}
			},
			wantMethods: []string{http.MethodHead, http.MethodHead, http.MethodHead},
		},
		{
			name:        "a page that answers at once",
			start:       "/peptides/",
			hops:        func(string) map[string]hop { return map[string]hop{"/peptides/": {status: http.StatusOK}} },
			want:        func(base string) wp.Resolution { return wp.Resolution{URL: base + "/peptides/", Status: http.StatusOK} },
			wantMethods: []string{http.MethodHead},
		},
		{
			name:  "a dead link is a resolution, not an error",
			start: "/old/",
			hops: func(string) map[string]hop {
				return map[string]hop{"/old/": {status: http.StatusMovedPermanently, location: "/gone/"}, "/gone/": {status: http.StatusNotFound}}
			},
			want: func(base string) wp.Resolution {
				return wp.Resolution{URL: base + "/gone/", Status: http.StatusNotFound}
			},
			wantMethods: []string{http.MethodHead, http.MethodHead},
		},
		{
			name:  "the first hop off the site stops the walk",
			start: "/partner/",
			hops: func(string) map[string]hop {
				return map[string]hop{"/partner/": {status: http.StatusMovedPermanently, location: elsewhere}}
			},
			want:        func(string) wp.Resolution { return wp.Resolution{URL: elsewhere, Status: http.StatusMovedPermanently} },
			wantMethods: []string{http.MethodHead},
		},
		{
			name:  "another port of the same host is off the site",
			start: "/moved/",
			hops: func(base string) map[string]hop {
				return map[string]hop{"/moved/": {status: http.StatusTemporaryRedirect, location: otherPort(base) + "/moved/"}}
			},
			want: func(base string) wp.Resolution {
				return wp.Resolution{URL: otherPort(base) + "/moved/", Status: http.StatusTemporaryRedirect}
			},
			wantMethods: []string{http.MethodHead},
		},
		{
			name:  "a site that refuses a head is read with one get",
			start: "/refusing/",
			hops: func(string) map[string]hop {
				return map[string]hop{"/refusing/": {status: http.StatusMethodNotAllowed, headless: true}}
			},
			want:        func(base string) wp.Resolution { return wp.Resolution{URL: base + "/refusing/", Status: http.StatusOK} },
			wantMethods: []string{http.MethodHead, http.MethodGet},
		},
		{
			name:  "a site that never implemented a head is read with one get",
			start: "/old-server/",
			hops: func(string) map[string]hop {
				return map[string]hop{"/old-server/": {status: http.StatusNotImplemented, headless: true}}
			},
			want: func(base string) wp.Resolution {
				return wp.Resolution{URL: base + "/old-server/", Status: http.StatusOK}
			},
			wantMethods: []string{http.MethodHead, http.MethodGet},
		},
		{
			name:  "a redirect without a location ends the walk where it stands",
			start: "/odd/",
			hops: func(string) map[string]hop {
				return map[string]hop{"/odd/": {status: http.StatusFound}}
			},
			want:        func(base string) wp.Resolution { return wp.Resolution{URL: base + "/odd/", Status: http.StatusFound} },
			wantMethods: []string{http.MethodHead},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			site := newRedirectSite(t, tc.hops)
			base := site.server.URL
			got, err := site.client(t).Resolve(t.Context(), base+tc.start)
			if err != nil {
				t.Fatalf("Resolve: %v", err)
			}
			if want := tc.want(base); got != want {
				t.Errorf("resolution %+v, want %+v", got, want)
			}

			seen := site.hops()
			methods := make([]string, 0, len(seen))
			for _, hop := range seen {
				methods = append(methods, hop.method)
				if hop.authorization != "" {
					t.Errorf("%s %s carried an Authorization header; a redirect is resolved as a visitor", hop.method, hop.path)
				}
				if hop.userAgent != resolveAgent {
					t.Errorf("%s %s User-Agent = %q, want %q", hop.method, hop.path, hop.userAgent, resolveAgent)
				}
			}
			if !slices.Equal(methods, tc.wantMethods) {
				t.Errorf("methods %v, want %v", methods, tc.wantMethods)
			}
		})
	}
}

func otherPort(base string) string {
	if strings.HasSuffix(base, ":1") {
		return strings.TrimSuffix(base, ":1") + ":2"
	}
	return base[:strings.LastIndex(base, ":")] + ":1"
}

func TestResolveRefusesWhatItCannotFinish(t *testing.T) {
	t.Parallel()

	loop := func(string) map[string]hop {
		return map[string]hop{"/loop/": {status: http.StatusMovedPermanently, location: "/loop/"}}
	}
	closed := newRedirectSite(t, loop)
	unreachable := closed.client(t)
	closed.server.Close()

	cancelled, cancel := context.WithCancel(t.Context())
	cancel()

	cases := []struct {
		name      string
		ctx       context.Context
		link      func(base string) string
		want      errors.Code
		wantHops  int
		reachable bool
	}{
		{
			name: "a loop beyond ten redirects", ctx: t.Context(), reachable: true,
			link: func(base string) string { return base + "/loop/" }, want: errors.External, wantHops: 11,
		},
		{
			name: "a cancelled context", ctx: cancelled, reachable: true,
			link: func(base string) string { return base + "/loop/" }, want: errors.Cancelled, wantHops: 0,
		},
		{
			name: "a relative address", ctx: t.Context(), reachable: true,
			link: func(string) string { return "/loop/" }, want: errors.Invalid, wantHops: 0,
		},
		{
			name: "an address of another site", ctx: t.Context(), reachable: true,
			link: func(string) string { return "https://elsewhere.example/loop/" }, want: errors.Invalid, wantHops: 0,
		},
		{
			name: "a site that cannot be reached", ctx: t.Context(),
			link: func(string) string { return closed.server.URL + "/loop/" }, want: errors.External,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if !tc.reachable {
				if _, err := unreachable.Resolve(tc.ctx, tc.link("")); !errors.IsCode(err, tc.want) {
					t.Fatalf("Resolve = %v, want %s", err, tc.want)
				}
				return
			}
			site := newRedirectSite(t, loop)
			if _, err := site.client(t).Resolve(tc.ctx, tc.link(site.server.URL)); !errors.IsCode(err, tc.want) {
				t.Fatalf("Resolve = %v, want %s", err, tc.want)
			}
			if got := len(site.hops()); got != tc.wantHops {
				t.Errorf("%d requests reached the site, want %d", got, tc.wantHops)
			}
		})
	}
}

func TestResolveWaitsOnTheRateLimitBeforeEveryHop(t *testing.T) {
	t.Parallel()

	hops := func(string) map[string]hop {
		return map[string]hop{
			"/a/": {status: http.StatusMovedPermanently, location: "/b/"},
			"/b/": {status: http.StatusOK},
		}
	}

	t.Run("a hop after the first waits for the limit", func(t *testing.T) {
		t.Parallel()

		site := newRedirectSite(t, hops)
		ctx, cancel := context.WithTimeout(t.Context(), 300*time.Millisecond)
		defer cancel()
		if _, err := site.client(t, wp.WithRateLimit(1)).Resolve(ctx, site.server.URL+"/a/"); !errors.IsCode(err, errors.Cancelled) {
			t.Fatalf("Resolve = %v, want the second hop refused by the rate limit before the deadline", err)
		}
		if got := len(site.hops()); got != 1 {
			t.Errorf("%d requests reached the site, want only the first hop", got)
		}
	})

	t.Run("the first hop waits for the limit too", func(t *testing.T) {
		t.Parallel()

		site := newRedirectSite(t, hops)
		client := site.client(t, wp.WithRateLimit(1))
		if _, err := client.Resolve(t.Context(), site.server.URL+"/b/"); err != nil {
			t.Fatalf("the first Resolve: %v", err)
		}
		ctx, cancel := context.WithTimeout(t.Context(), 300*time.Millisecond)
		defer cancel()
		if _, err := client.Resolve(ctx, site.server.URL+"/b/"); !errors.IsCode(err, errors.Cancelled) {
			t.Fatalf("Resolve = %v, want the first hop refused by the rate limit before the deadline", err)
		}
		if got := len(site.hops()); got != 1 {
			t.Errorf("%d requests reached the site, want only the first call's", got)
		}
	})

	t.Run("without a limit every hop goes", func(t *testing.T) {
		t.Parallel()

		site := newRedirectSite(t, hops)
		got, err := site.client(t).Resolve(t.Context(), site.server.URL+"/a/")
		if err != nil || got.Status != http.StatusOK || len(site.hops()) != 2 {
			t.Fatalf("Resolve = %+v, %v after %d hops, want 200 after 2", got, err, len(site.hops()))
		}
	})
}
