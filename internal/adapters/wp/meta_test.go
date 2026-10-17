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

func TestGetMetaReadsOnlyTheKeysAsked(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	page := server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Holder", Meta: map[string]string{"color": "red", "size": "10"}})[0]

	meta, err := newClient(t, server).GetMeta(t.Context(), wp.TypePage, page.ID, []string{"color", " missing ", "color"})
	if err != nil {
		t.Fatalf("GetMeta: %v", err)
	}
	if len(meta) != 1 || string(meta["color"]) != `"red"` {
		t.Errorf("meta = %v, want color alone, the missing key left out", meta)
	}
	recorded, _ := server.LastRequest()
	if got := recorded.Query.Get("keys"); got != "color,missing" {
		t.Errorf("keys = %q, want color,missing", got)
	}
}

func TestGetMetaRefusesWhatItMayNotAsk(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	page := server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Holder", Meta: map[string]string{"_secret": "x"}})[0]
	client := newClient(t, server)

	many := make([]string, 51)
	for index := range many {
		many[index] = "key" + strconv.Itoa(index)
	}
	for name, keys := range map[string][]string{"no key": {" "}, "too many keys": many, "a key with a comma": {"a,b"}} {
		server.ResetRequests()
		if _, err := client.GetMeta(t.Context(), wp.TypePage, page.ID, keys); !errors.IsCode(err, errors.Invalid) {
			t.Errorf("%s = %v, want Invalid", name, err)
		}
		for _, recorded := range server.Requests() {
			if strings.Contains(recorded.Path, "/meta/") {
				t.Errorf("%s reached the site", name)
			}
		}
	}

	_, err := client.GetMeta(t.Context(), wp.TypePage, page.ID, []string{"_secret"})
	if !errors.IsCode(err, errors.Invalid) || detailOf(t, err, "code") != "meta_key_denied" {
		t.Errorf("a protected key = %v, want the site's meta_key_denied", err)
	}

	withoutMeta := slices.DeleteFunc(slices.Clone(currentCapabilities), func(name string) bool { return name == wp.CapabilityMeta })
	old := wptest.New(t, wptest.WithCapabilities(withoutMeta...))
	held := old.Seed(wptest.Item{Type: wptest.TypePage, Title: "Holder"})[0]
	if _, err := newClient(t, old).GetMeta(t.Context(), wp.TypePage, held.ID, []string{"color"}); !wp.IsPluginOutdated(err) {
		t.Errorf("a plugin without meta = %v, want plugin_outdated", err)
	}
}
