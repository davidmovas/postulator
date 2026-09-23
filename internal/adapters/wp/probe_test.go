package wp_test

import (
	"slices"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func TestProbeReportsAHealthySite(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	result, err := newClient(t, server).Probe(t.Context())
	if err != nil {
		t.Fatalf("Probe: %v", err)
	}

	if result.Status != wp.ProbeOK {
		t.Errorf("status = %q, want %q", result.Status, wp.ProbeOK)
	}
	if !slices.Contains(result.Namespaces, "wp/v2") {
		t.Errorf("namespaces = %v", result.Namespaces)
	}
	if !result.HasPlugin {
		t.Error("the fake advertises postulator/v1")
	}
	if !result.HasWoo {
		t.Error("the fake advertises wc/v3")
	}
	if result.SiteName == "" || result.HomeURL == "" {
		t.Errorf("site = %q, home = %q", result.SiteName, result.HomeURL)
	}
	if len(result.Warnings) != 0 {
		t.Errorf("warnings = %v, want none", result.Warnings)
	}
}

func TestProbeReportsAMissingPlugin(t *testing.T) {
	t.Parallel()

	server := wptest.New(t, wptest.WithoutPlugin())
	result, err := newClient(t, server).Probe(t.Context())
	if err != nil {
		t.Fatalf("Probe: %v", err)
	}
	if result.HasPlugin {
		t.Error("the plugin namespace is not advertised")
	}
}

func TestProbeClassifiesAFailingSite(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		options []wptest.Option
		want    errors.Code
	}{
		{name: "no namespaces at all", options: []wptest.Option{wptest.WithoutNamespaces()}, want: errors.External},
		{name: "redirected to the login page", options: []wptest.Option{wptest.WithRedirect(wptest.RedirectLogin)}, want: errors.Unauthorized},
		{name: "redirected to the admin area", options: []wptest.Option{wptest.WithRedirect(wptest.RedirectAdmin)}, want: errors.Unauthorized},
		{name: "credentials rejected", options: []wptest.Option{wptest.WithCredentials("other", "pass word")}, want: errors.Unauthorized},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server := wptest.New(t, tc.options...)
			_, err := newClient(t, server).Probe(t.Context())
			if !errors.IsCode(err, tc.want) {
				t.Errorf("code = %q, want %q", errors.CodeOf(err), tc.want)
			}
		})
	}
}

func TestProbeReportsAnHTTPSUpgradeAsAWarningRatherThanAFailure(t *testing.T) {
	t.Parallel()

	server := wptest.New(t, wptest.WithRedirect(wptest.RedirectHTTPS))
	result, err := newClient(t, server).Probe(t.Context())
	if err != nil {
		t.Fatalf("Probe: %v", err)
	}

	if result.Status != wp.ProbeUpgradeRequired {
		t.Errorf("status = %q, want %q", result.Status, wp.ProbeUpgradeRequired)
	}
	if len(result.Warnings) == 0 {
		t.Error("an upgrade must come with a warning the user can act on")
	}
	if result.SuggestedBaseURL == "" {
		t.Error("the warning must carry the base URL the site wants")
	}
	if got := len(server.Requests()); got != 1 {
		t.Errorf("the site saw %d requests; Probe must not follow the redirect", got)
	}
}

func TestProbeRejectsSomethingThatIsNotWordPress(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	client, err := wp.New(wp.Config{
		BaseURL:       server.URL() + "/shop",
		Username:      wptest.DefaultUser,
		AppPassword:   wptest.DefaultPassword,
		AllowInsecure: true,
	}, wp.WithRateLimit(0), wp.WithBackoff(func(int) time.Duration { return 0 }))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if _, err = client.Probe(t.Context()); !errors.IsCode(err, errors.External) {
		t.Errorf("code = %q, want %q", errors.CodeOf(err), errors.External)
	}
}
