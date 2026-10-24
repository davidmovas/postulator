package wp_test

import (
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func TestGetRenderedReadsTheFilteredContent(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	filtered := `<nav aria-label="Breadcrumb">Home</nav><p>Raw.</p>`
	page := server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Hub", Content: "Raw.", Filtered: filtered})[0]

	rendered, err := newClient(t, server).GetRendered(t.Context(), wp.TypePage, page.ID)
	if err != nil {
		t.Fatalf("GetRendered: %v", err)
	}
	if rendered.ID != page.ID || rendered.Type != wp.TypePage || rendered.Content != filtered || rendered.ContentHash != wp.ContentHash(filtered) {
		t.Errorf("rendered = %+v", rendered)
	}
	recorded, _ := server.LastRequest()
	if !strings.HasSuffix(recorded.Path, "/content/"+strconv.FormatInt(page.ID, 10)+"/rendered") {
		t.Errorf("path = %q", recorded.Path)
	}
}

func TestGetRenderedRefusesWhatItCannotRead(t *testing.T) {
	t.Parallel()

	withoutRendered := slices.DeleteFunc(slices.Clone(currentCapabilities), func(name string) bool { return name == wp.CapabilityRendered })
	old := wptest.New(t, wptest.WithCapabilities(withoutRendered...))
	page := old.Seed(wptest.Item{Type: wptest.TypePage, Title: "Hub"})[0]
	if _, err := newClient(t, old).GetRendered(t.Context(), wp.TypePage, page.ID); !wp.IsPluginOutdated(err) {
		t.Errorf("a plugin without rendered = %v, want plugin_outdated", err)
	}
	for _, recorded := range old.Requests() {
		if strings.HasSuffix(recorded.Path, "/rendered") {
			t.Errorf("an outdated plugin was asked for %s", recorded.Path)
		}
	}

	server := wptest.New(t)
	client := newClient(t, server)
	if _, err := client.GetRendered(t.Context(), wp.TypeProductCategory, 3); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("a product category = %v, want Invalid", err)
	}
	if _, err := client.GetRendered(t.Context(), wp.TypePage, 9999); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("an unknown page = %v, want NotFound", err)
	}
	product := server.Seed(wptest.Item{Type: wptest.TypeProduct, Title: "Powder"})[0]
	if _, err := client.GetRendered(t.Context(), wp.TypePage, product.ID); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("a product read as a page = %v, want NotFound", err)
	}
}
