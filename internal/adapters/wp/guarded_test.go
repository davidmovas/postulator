package wp_test

import (
	"encoding/json"
	stderrors "errors"
	"maps"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func hashOf(value string) *string {
	return new(wp.ContentHash(value))
}

func movedOf(t *testing.T, err error) map[string]string {
	t.Helper()

	var kernel *errors.Error
	if !stderrors.As(err, &kernel) || kernel.Code != errors.Conflict {
		t.Fatalf("err = %v, want CONFLICT", err)
	}
	moved, ok := kernel.Details["moved"].(map[string]string)
	if !ok {
		t.Fatalf("the conflict names no moved values: %+v", kernel.Details)
	}
	return moved
}

func writesTo(server *wptest.Server, route string) int {
	count := 0
	for _, request := range server.Requests() {
		if request.Method != http.MethodGet && strings.Contains(request.Path, route) {
			count++
		}
	}
	return count
}

func TestPutMetaWritesUnderTheHashRead(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	page := server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Holder", Meta: map[string]string{"subtitle": "Old"}})[0]
	client := newClient(t, server)

	stored, err := client.PutMeta(t.Context(), wp.TypePage, page.ID, map[string]wp.MetaWrite{
		"subtitle": {Value: new("New"), Expected: hashOf("Old")},
		"tagline":  {Value: new("Fresh")},
	})
	if err != nil {
		t.Fatalf("PutMeta: %v", err)
	}
	if got := stored["subtitle"]; got.Value == nil || *got.Value != "New" || got.Hash == nil || *got.Hash != wp.ContentHash("New") {
		t.Errorf("the stored subtitle is %+v", got)
	}
	if got := stored["tagline"]; got.Value == nil || *got.Value != "Fresh" {
		t.Errorf("the stored tagline is %+v", got)
	}
	held, _ := server.Lookup(page.ID)
	if held.Meta["subtitle"] != "New" || held.Meta["tagline"] != "Fresh" {
		t.Errorf("the site holds %v", held.Meta)
	}

	request, _ := server.LastRequest()
	var wire struct {
		Meta     map[string]*string `json:"meta"`
		Expected map[string]*string `json:"expected"`
	}
	if err = json.Unmarshal(request.Body, &wire); err != nil {
		t.Fatalf("decode the request: %v", err)
	}
	if want := []string{"subtitle", "tagline"}; !slices.Equal(slices.Sorted(maps.Keys(wire.Expected)), want) || wire.Expected["tagline"] != nil {
		t.Errorf("the request expected %v, want every key it writes, the new one as absent", wire.Expected)
	}

	stored, err = client.PutMeta(t.Context(), wp.TypePage, page.ID, map[string]wp.MetaWrite{
		"subtitle": {Expected: hashOf("New")},
	})
	if err != nil {
		t.Fatalf("PutMeta to delete: %v", err)
	}
	if got := stored["subtitle"]; got.Value != nil || got.Hash != nil {
		t.Errorf("a deleted key is stored as %+v", got)
	}
	if held, _ = server.Lookup(page.ID); slices.Contains(slices.Collect(maps.Keys(held.Meta)), "subtitle") {
		t.Errorf("the site still holds the subtitle: %v", held.Meta)
	}
}

func TestPutMetaWritesNothingWhenAValueMoved(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	page := server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Holder", Meta: map[string]string{"subtitle": "Old", "tagline": "Kept"}})[0]
	client := newClient(t, server)

	server.EditMeta(page.ID, "subtitle", "The client's own")
	server.EditMeta(page.ID, "note", "Added since")
	_, err := client.PutMeta(t.Context(), wp.TypePage, page.ID, map[string]wp.MetaWrite{
		"subtitle": {Value: new("New"), Expected: hashOf("Old")},
		"tagline":  {Value: new("Changed"), Expected: hashOf("Kept")},
		"note":     {Value: new("Ours")},
	})
	moved := movedOf(t, err)
	if want := map[string]string{"subtitle": wp.ContentHash("The client's own"), "note": wp.ContentHash("Added since")}; !maps.Equal(moved, want) {
		t.Errorf("moved = %v, want %v", moved, want)
	}
	held, _ := server.Lookup(page.ID)
	if held.Meta["subtitle"] != "The client's own" || held.Meta["tagline"] != "Kept" || held.Meta["note"] != "Added since" {
		t.Errorf("a refused write changed the site: %v", held.Meta)
	}
}

func TestADeniedMetaKeyIsRefusedByTheClient(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	page := server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Holder"})[0]
	client := newClient(t, server)

	for _, key := range []string{"_price", "TOTAL_SALES", "attribute_pa_color", "rank_math_title", "café_note", "total_sales ", ""} {
		_, err := client.PutMeta(t.Context(), wp.TypePage, page.ID, map[string]wp.MetaWrite{
			"subtitle": {Value: new("fine")},
			key:        {Value: new("9.99")},
		})
		if !errors.IsCode(err, errors.Invalid) || detailOf(t, err, "reason") != field.ReasonDenied || detailOf(t, err, "key") != key {
			t.Errorf("PutMeta(%q) = %v, want INVALID %s naming the key", key, err, field.ReasonDenied)
		}
	}
	if got := writesTo(server, "/meta/"); got != 0 {
		t.Errorf("%d meta writes left the client, want none", got)
	}
}

func TestEveryPluginWriteNeedsTheGuardedWrite(t *testing.T) {
	t.Parallel()

	older := slices.DeleteFunc(slices.Clone(currentCapabilities), func(name string) bool {
		return name == wp.CapabilityGuardedWrite || name == wp.CapabilityMetaWrite
	})
	server := wptest.New(t, wptest.WithPluginVersion("1.4.1"), wptest.WithCapabilities(older...))
	page := server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Holder", Content: koffeinBody})[0]
	client := newClient(t, server)

	writes := map[string]func() error{
		"the raw body": func() error {
			_, err := client.PutRaw(t.Context(), wp.TypePage, page.ID, "<p>x</p>", koffeinHash)
			return err
		},
		"the SEO meta": func() error {
			_, err := client.SetSEOMeta(t.Context(), wp.TypePage, page.ID, wp.SEOMeta{Title: "x"}, nil)
			return err
		},
		"the SEO meta put back": func() error {
			_, err := client.ReplaceSEOMeta(t.Context(), wp.TypePage, page.ID, wp.SEOMeta{}, []string{"title"}, nil)
			return err
		},
		"a meta key": func() error {
			_, err := client.PutMeta(t.Context(), wp.TypePage, page.ID, map[string]wp.MetaWrite{"subtitle": {Value: new("x")}})
			return err
		},
	}
	for name, write := range writes {
		if err := write(); !wp.IsPluginOutdated(err) {
			t.Errorf("%s through plugin 1.4.1: %v, want plugin_outdated", name, err)
		}
	}
	for _, route := range []string{"/raw", "/seo-meta/", "/meta/"} {
		if got := writesTo(server, route); got != 0 {
			t.Errorf("%d writes reached %s, want none", got, route)
		}
	}
}

func TestThePutRawAnswersWhatTheSiteStored(t *testing.T) {
	t.Parallel()

	server := wptest.New(t, wptest.WithFilteredHTML())
	page := server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Holder", Content: koffeinBody})[0]
	client := newClient(t, server)

	written, err := client.PutRaw(t.Context(), wp.TypePage, page.ID, "<p>Powder</p><script>alert(1)</script>", koffeinHash)
	if err != nil {
		t.Fatalf("PutRaw: %v", err)
	}
	held, _ := server.Lookup(page.ID)
	if written.Content != held.Content || written.Content != "<p>Powder</p>" || written.Hash != wp.ContentHash(held.Content) {
		t.Errorf("the write answered %+v, and the site holds %q", written, held.Content)
	}
}

func TestGetRawSaysWhetherASaveKeepsTheContent(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		options []wptest.Option
		content string
		stable  bool
	}{
		{name: "a user without unfiltered HTML over a script", options: []wptest.Option{wptest.WithFilteredHTML()}, content: "<p>a</p><script>x</script>"},
		{name: "a user without unfiltered HTML over plain prose", options: []wptest.Option{wptest.WithFilteredHTML()}, content: "<p>a</p>", stable: true},
		{name: "a user with unfiltered HTML over a script", content: "<p>a</p><script>x</script>", stable: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server := wptest.New(t, tc.options...)
			page := server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Holder", Content: tc.content})[0]
			raw, err := newClient(t, server).GetRaw(t.Context(), wp.TypePage, page.ID)
			if err != nil {
				t.Fatalf("GetRaw: %v", err)
			}
			if raw.SaveStable != tc.stable {
				t.Errorf("SaveStable = %t, want %t", raw.SaveStable, tc.stable)
			}
		})
	}
}

func TestSetSEOMetaWritesUnderTheHashesRead(t *testing.T) {
	t.Parallel()

	server := wptest.New(t, wptest.WithSEOPlugin("yoast"))
	page := server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Holder", Meta: map[string]string{"_yoast_wpseo_title": "Old"}})[0]
	client := newClient(t, server)

	result, err := client.SetSEOMeta(t.Context(), wp.TypePage, page.ID, wp.SEOMeta{Title: "New"},
		map[string]string{"title": wp.ContentHash("Old")})
	if err != nil {
		t.Fatalf("SetSEOMeta: %v", err)
	}
	if !slices.Equal(result.Applied, []string{"title"}) || result.Hashes["title"] != wp.ContentHash("New") {
		t.Errorf("result = %+v", result)
	}

	server.EditMeta(page.ID, "_yoast_wpseo_title", "The client's own")
	_, err = client.SetSEOMeta(t.Context(), wp.TypePage, page.ID, wp.SEOMeta{Title: "Again", Description: "About"},
		map[string]string{"title": wp.ContentHash("New"), "description": wp.ContentHash("")})
	if moved := movedOf(t, err); !maps.Equal(moved, map[string]string{"title": wp.ContentHash("The client's own")}) {
		t.Errorf("moved = %v", moved)
	}
	_, err = client.ReplaceSEOMeta(t.Context(), wp.TypePage, page.ID, wp.SEOMeta{}, []string{"title"},
		map[string]string{"title": wp.ContentHash("New")})
	movedOf(t, err)
	held, _ := server.Lookup(page.ID)
	if held.Meta["_yoast_wpseo_title"] != "The client's own" || held.Meta["_yoast_wpseo_metadesc"] != "" {
		t.Errorf("a refused write changed the site: %v", held.Meta)
	}
}

func TestTheManifestSaysWhetherTheUserMaySaveUnfilteredHTML(t *testing.T) {
	t.Parallel()

	for _, filtered := range []bool{false, true} {
		var options []wptest.Option
		if filtered {
			options = append(options, wptest.WithFilteredHTML())
		}
		capabilities, err := newClient(t, wptest.New(t, options...)).Capabilities(t.Context())
		if err != nil {
			t.Fatalf("Capabilities: %v", err)
		}
		if capabilities.UnfilteredHTML == filtered {
			t.Errorf("UnfilteredHTML = %t for a user whose HTML is filtered %t", capabilities.UnfilteredHTML, filtered)
		}
		if !capabilities.Has(wp.CapabilityGuardedWrite) || !capabilities.Has(wp.CapabilityMetaWrite) {
			t.Errorf("names = %v, want the guarded write and the meta write", capabilities.Names)
		}
	}
}
