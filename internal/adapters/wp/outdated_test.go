package wp_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
)

var currentCapabilities = []string{
	"bulk", "seo_meta", "seo_meta_read", "content_hash", "raw", "preview", "rendered", "meta", "terms", "settings",
	"meta_write", "guarded_write",
}

func TestAnOutdatedPluginIsNamedByVersionAndCapabilities(t *testing.T) {
	t.Parallel()

	withoutRaw := slices.DeleteFunc(slices.Clone(currentCapabilities), func(name string) bool { return name == "raw" })
	cases := []struct {
		name         string
		version      string
		capabilities []string
		missing      []string
		outdated     bool
	}{
		{name: "the version the client needs with every capability", version: wp.PluginVersion, capabilities: currentCapabilities},
		{name: "the contract this client needs", version: "1.5.1", capabilities: currentCapabilities},
		{name: "the contract before the save probe covered the title and the excerpt", version: "1.5.0", capabilities: currentCapabilities, outdated: true},
		{name: "the contract before writes were guarded", version: "1.4.1", capabilities: currentCapabilities, outdated: true},
		{
			name: "the guarded write missing", version: wp.PluginVersion, missing: []string{"guarded_write"}, outdated: true,
			capabilities: slices.DeleteFunc(slices.Clone(currentCapabilities), func(name string) bool { return name == "guarded_write" }),
		},
		{name: "a newer version", version: "99.0.0", capabilities: currentCapabilities},
		{name: "a minor number past ten", version: "1.10.0", capabilities: currentCapabilities},
		{name: "an older version", version: "0.9.0", capabilities: currentCapabilities, outdated: true},
		{name: "a capability missing", version: wp.PluginVersion, capabilities: withoutRaw, missing: []string{"raw"}, outdated: true},
		{name: "an unreadable version", version: "x", capabilities: currentCapabilities, outdated: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server := wptest.New(t, wptest.WithPluginVersion(tc.version), wptest.WithCapabilities(tc.capabilities...))
			capabilities, err := newClient(t, server).Capabilities(t.Context())
			if err != nil {
				t.Fatalf("Capabilities: %v", err)
			}
			if got := capabilities.Outdated(); got != tc.outdated {
				t.Errorf("Outdated = %t, want %t", got, tc.outdated)
			}
			if got := capabilities.Missing(); !slices.Equal(got, tc.missing) {
				t.Errorf("Missing = %v, want %v", got, tc.missing)
			}
		})
	}
}

func TestAnOutdatedPluginSaysHowToUpdateIt(t *testing.T) {
	t.Parallel()

	withoutRead := slices.DeleteFunc(slices.Clone(currentCapabilities), func(name string) bool { return name == wp.CapabilitySEOMetaRead })
	server := wptest.New(t, wptest.WithPluginVersion("1.1.0"), wptest.WithCapabilities(withoutRead...))
	page := server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Coffee"})[0]

	_, err := newClient(t, server).GetSEOMeta(t.Context(), wp.TypePage, page.ID)
	if !wp.IsPluginOutdated(err) {
		t.Fatalf("GetSEOMeta = %v, want plugin_outdated", err)
	}
	for key, want := range map[string]string{"capability": wp.CapabilitySEOMetaRead, "version": "1.1.0", "required": wp.PluginVersion} {
		if got := detailOf(t, err, key); got != want {
			t.Errorf("detail %s = %q, want %q", key, got, want)
		}
	}
	if !strings.Contains(err.Error(), "upload the new zip over the old one") {
		t.Errorf("the error does not say how to update: %v", err)
	}
}
