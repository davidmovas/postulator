package wp_test

import (
	"slices"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
)

func TestPluginSettingsNameTheTechnicalPages(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	server.SetReading("page", 12, 14)
	server.SetPrivacyPage(3)
	server.SetStorePages(wptest.StorePages{Shop: 5, Cart: 6, Checkout: 7, Account: 8, Terms: 9})

	settings, err := newClient(t, server).PluginSettings(t.Context())
	if err != nil {
		t.Fatalf("PluginSettings: %v", err)
	}
	want := wp.PluginSettings{
		ShowOnFront: "page",
		Pages:       wp.TechnicalPages{Front: 12, Posts: 14, Privacy: 3, Shop: 5, Cart: 6, Checkout: 7, Account: 8, Terms: 9},
	}
	if settings != want {
		t.Errorf("settings = %+v, want %+v", settings, want)
	}

	withoutSettings := slices.DeleteFunc(slices.Clone(currentCapabilities), func(name string) bool { return name == wp.CapabilitySettings })
	old := wptest.New(t, wptest.WithCapabilities(withoutSettings...))
	if _, err := newClient(t, old).PluginSettings(t.Context()); !wp.IsPluginOutdated(err) {
		t.Errorf("a plugin without settings = %v, want plugin_outdated", err)
	}
}
