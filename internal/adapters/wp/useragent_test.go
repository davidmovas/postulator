package wp_test

import (
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
)

func TestTheDefaultUserAgentNamesNeitherTheProgramNorThePlugin(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"postulator", "companion"} {
		if strings.Contains(strings.ToLower(wp.DefaultUserAgent), name) {
			t.Errorf("the default User-Agent %q names %q", wp.DefaultUserAgent, name)
		}
	}

	cases := []struct {
		name string
		opts []wp.Option
	}{
		{name: "no override"},
		{name: "an empty override", opts: []wp.Option{wp.WithUserAgent("")}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server := wptest.New(t)
			if _, err := newClient(t, server, tc.opts...).Probe(t.Context()); err != nil {
				t.Fatalf("Probe: %v", err)
			}
			recorded, ok := server.LastRequest()
			if !ok {
				t.Fatal("no request reached the site")
			}
			if got := recorded.Header.Get("User-Agent"); got != wp.DefaultUserAgent {
				t.Errorf("User-Agent = %q, want %q", got, wp.DefaultUserAgent)
			}
		})
	}
}

func TestAUserAgentOverrideReachesEveryRequest(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	server.Seed(wptest.Item{Type: wptest.TypeProduct, Title: "Powder", Slug: "powder", Status: "publish", Content: "<p>Made by hand.</p>"})
	client := newClient(t, server, wp.WithUserAgent("Agent/1"))

	if _, err := client.Capabilities(t.Context()); err != nil {
		t.Fatalf("Capabilities: %v", err)
	}
	if _, err := client.ListItems(t.Context(), wp.TypePage, wp.ListQuery{}); err != nil {
		t.Fatalf("ListItems: %v", err)
	}
	if _, err := client.Visit(t.Context(), server.URL()+"/product/powder/"); err != nil {
		t.Fatalf("Visit: %v", err)
	}

	requests := server.Requests()
	if len(requests) != 3 {
		t.Fatalf("requests = %d, want 3", len(requests))
	}
	for _, recorded := range requests {
		if got := recorded.Header.Get("User-Agent"); got != "Agent/1" {
			t.Errorf("%s %s User-Agent = %q, want Agent/1", recorded.Method, recorded.Path, got)
		}
	}
}
