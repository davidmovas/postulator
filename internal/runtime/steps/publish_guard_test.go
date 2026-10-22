package steps_test

import (
	"encoding/json"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/runtime/steps"
)

const (
	hisIntro  = "<p>His intro about coffee beans.</p>"
	hisOutro  = "<p>His outro about the grinder.</p>"
	hisGrid   = `[products limit="4"]`
	hisBanner = `<div class="banner">Sale today</div>`
)

func existingPage(t *testing.T, opts []wptest.Option, item wptest.Item) (steps.Deps, *wptest.Server, *run.StepContext, int64) {
	t.Helper()

	deps, server := imageDepsWith(t, opts...)
	if item.Type == "" {
		item.Type = wptest.TypePage
	}
	if item.Slug == "" {
		item.Slug = "espresso"
	}
	if item.Status == "" {
		item.Status = "publish"
	}
	seeded := server.Seed(item)[0]
	sc := rewritten(publishContext(t))
	sc.Page.WPID = &seeded.ID
	return deps, server, sc, seeded.ID
}

func itoa(id int64) string {
	return strconv.FormatInt(id, 10)
}

func writesSince(server *wptest.Server, from int) []wptest.Request {
	out := make([]wptest.Request, 0)
	for i, request := range server.Requests() {
		if i >= from && request.Method != http.MethodGet && request.Method != http.MethodHead {
			out = append(out, request)
		}
	}
	return out
}

func planOnly(t *testing.T, def run.StepDef, sc *run.StepContext) {
	t.Helper()

	result, err := def.Run(t.Context(), sc)
	if err != nil || result.Next != run.TransitionWait {
		t.Fatalf("the plan = %+v, %v; want it to wait for its first write", result, err)
	}
	settle(def, sc, result)
}

func TestRewritesKeepTheClientsNonTextBlocks(t *testing.T) {
	t.Parallel()

	deps, server, sc, id := existingPage(t, nil, wptest.Item{
		Title: "Espresso", Content: hisIntro + "\n\n" + hisGrid + "\n\n" + hisOutro + "\n\n" + hisBanner,
	})
	runPublish(t, deps, sc)

	held, _ := server.Lookup(id)
	for _, kept := range []string{hisGrid, hisBanner} {
		if !strings.Contains(held.Content, kept) {
			t.Errorf("the rewrite lost the client's %q: %q", kept, held.Content)
		}
	}
	if strings.Contains(held.Content, "His intro") || strings.Contains(held.Content, "His outro") || !strings.Contains(held.Content, "Espresso is a way") {
		t.Errorf("the rewrite did not put ours in place of his text: %q", held.Content)
	}
	if strings.Index(held.Content, "Espresso is a way") > strings.Index(held.Content, hisGrid) {
		t.Errorf("ours did not take his first text block's place: %q", held.Content)
	}
}

func TestTheFeaturedImageIsNeverReplaced(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		held int64
		want int64
	}{
		{name: "a page with the client's image keeps it", held: 77, want: 77},
		{name: "a page without one takes ours", held: 0, want: 99},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			deps, server, sc, id := existingPage(t, nil, wptest.Item{Title: "Espresso", Content: hisIntro, FeaturedMedia: tc.held})
			sc.Artifacts[run.ArtifactImages] = run.Artifact{Kind: run.ArtifactImages, Blob: []byte(`{"images":[],"findings":[],"featuredId":99,"wanted":1}`)}
			runPublish(t, deps, sc)
			if held, _ := server.Lookup(id); held.FeaturedMedia != tc.want {
				t.Errorf("the featured image is %d, want %d", held.FeaturedMedia, tc.want)
			}
		})
	}
}

func TestOurBlocksAreWhatTheSiteStored(t *testing.T) {
	t.Parallel()

	const (
		ours   = "<p>Pull a shot.<br>Steam the milk.</p>"
		stored = "<p>Pull a shot.<br />Steam the milk.</p>"
	)
	deps, server := imageDepsWith(t, wptest.WithFilteredHTML())
	log := &bodyLog{}
	deps.Bodies = log
	seeded := server.Seed(wptest.Item{Type: "page", Title: "Espresso", Slug: "espresso", Status: "publish", Content: clientText})[0]
	sc := publishContext(t)
	sc.Page.WPID = &seeded.ID
	sc.Artifacts[run.ArtifactBodyHTML] = run.Artifact{Kind: run.ArtifactBodyHTML, Blob: []byte("<h1>Espresso</h1>" + ours)}
	sc.Artifacts[run.ArtifactDraft] = run.Artifact{Kind: run.ArtifactDraft, Blob: keptDraft(t, &content.KeptBody{
		Raw: clientText, Hash: block.BodyHash(clientText),
	})}
	server.ResetRequests()

	runPublish(t, deps, sc)
	held, _ := server.Lookup(seeded.ID)
	if !strings.Contains(held.Content, stored) {
		t.Fatalf("the site holds %q, want the save filters' form of ours", held.Content)
	}
	if got := log.ours(sc.Page.ID); !slices.Equal(got, []string{block.Hash(stored)}) {
		t.Errorf("our blocks are %v, want the hash of what the site stored", got)
	}
	put := -1
	for i, request := range server.Requests() {
		switch {
		case request.Method == http.MethodPut && strings.HasSuffix(request.Path, "/raw"):
			put = i
		case put >= 0 && request.Method == http.MethodGet && strings.HasSuffix(request.Path, "/raw"):
			t.Errorf("the body was read again after its write, where an edit could race the record")
		}
	}
}

func TestAPageThatIsNotSaveStableIsNotResaved(t *testing.T) {
	t.Parallel()

	const unstable = "<p>Map</p><script>window.map = 1;</script>"
	deps, server, sc, id := existingPage(t, []wptest.Option{wptest.WithFilteredHTML(), wptest.WithSEOPlugin("yoast")},
		wptest.Item{Title: "Espresso", Content: unstable})
	server.ResetRequests()

	result, err := drive(t, steps.Publish(deps), sc)
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}
	if result.Next != run.TransitionPause || !strings.Contains(result.Message, "unfiltered_html") {
		t.Fatalf("result = %+v, want a pause naming the user's missing unfiltered_html", result)
	}
	held, _ := server.Lookup(id)
	if held.Content != unstable || held.Title != "Espresso" {
		t.Errorf("the page was saved again: %q %q", held.Title, held.Content)
	}
	for _, request := range writesSince(server, 0) {
		if !strings.Contains(request.Path, "/seo-meta/") {
			t.Errorf("%s %s re-saved a page whose content a save would change", request.Method, request.Path)
		}
	}
	if held.Meta["_yoast_wpseo_title"] != "espresso | Shop" {
		t.Errorf("the SEO meta, which never re-saves the post, was not written: %v", held.Meta)
	}
}

func TestAPageWhoseTitleOrExcerptASaveWouldAlterIsNotResaved(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		item wptest.Item
	}{
		{name: "a title kses would change", item: wptest.Item{Title: "Salt & Pepper", Content: "<p>Only prose.</p>"}},
		{name: "an excerpt kses would strip", item: wptest.Item{Title: "Espresso", Excerpt: "<p>Map</p><script>window.map = 1;</script>", Content: "<p>Only prose.</p>"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			deps, server, sc, id := existingPage(t, []wptest.Option{wptest.WithFilteredHTML()}, tc.item)
			server.ResetRequests()

			result, err := drive(t, steps.Publish(deps), sc)
			if err != nil {
				t.Fatalf("Publish: %v", err)
			}
			if result.Next != run.TransitionPause || !strings.Contains(result.Message, "unfiltered_html") {
				t.Fatalf("result = %+v, want a pause naming the user's missing unfiltered_html", result)
			}
			held, _ := server.Lookup(id)
			if held.Title != tc.item.Title || held.Excerpt != tc.item.Excerpt || held.Content != tc.item.Content {
				t.Errorf("the page was saved again: %q %q %q", held.Title, held.Excerpt, held.Content)
			}
			for _, request := range writesSince(server, 0) {
				if !strings.Contains(request.Path, "/seo-meta/") {
					t.Errorf("%s %s re-saved a page whose title or excerpt a save would change", request.Method, request.Path)
				}
			}
		})
	}
}

func TestTheExcerptBaseIsWhatTheSiteStored(t *testing.T) {
	t.Parallel()

	deps, server, sc, id := existingPage(t, []wptest.Option{wptest.WithFilteredHTML()}, wptest.Item{Title: "Espresso", Content: "<p>Only prose.</p>"})
	deps.Types = scopedTypes{keys: []field.Key{field.Excerpt}}
	log := &fieldLog{}
	deps.Fields = log
	withFields(t, sc, steps.FieldValue{Key: string(field.Excerpt), Value: "Salt & pepper notes."})

	if _, err := drive(t, steps.Publish(deps), sc); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	held, _ := server.Lookup(id)
	if held.Excerpt != "Salt &amp; pepper notes." {
		t.Fatalf("the site stored the excerpt as %q, want kses' form", held.Excerpt)
	}
	if got := log.recordedValue(field.Excerpt); got != held.Excerpt {
		t.Errorf("the excerpt's base is %q, want what the site stored %q", got, held.Excerpt)
	}
}

func TestAWriteAgainstAStaleHashPauses(t *testing.T) {
	t.Parallel()

	deps, server, sc, id := existingPage(t, nil, wptest.Item{Title: "Espresso", Content: hisIntro})
	def := steps.Publish(deps)
	planOnly(t, def, sc)
	server.Rewrite(id, "<p>Edited by the client after the read.</p>")
	from := len(server.Requests())

	result, err := drive(t, def, sc)
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}
	if result.Next != run.TransitionPause || !strings.Contains(result.Message, "body") {
		t.Fatalf("result = %+v, want a pause naming the body", result)
	}
	if writes := writesSince(server, from); len(writes) != 0 {
		t.Errorf("%d writes left after the body moved, want none: %+v", len(writes), writes[0].Path)
	}
	if held, _ := server.Lookup(id); held.Content != "<p>Edited by the client after the read.</p>" {
		t.Errorf("the client's edit was written over: %q", held.Content)
	}
}

func TestEveryGuardIsCheckedBeforeTheFirstWrite(t *testing.T) {
	t.Parallel()

	deps, server, sc, id := existingPage(t, []wptest.Option{wptest.WithSEOPlugin("yoast")}, wptest.Item{
		Title: "Espresso", Content: hisIntro, Meta: map[string]string{"_yoast_wpseo_title": "His SEO title"},
	})
	def := steps.Publish(deps)
	planOnly(t, def, sc)
	server.EditMeta(id, "_yoast_wpseo_title", "His newer SEO title")
	from := len(server.Requests())

	result, err := drive(t, def, sc)
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}
	if result.Next != run.TransitionPause || !strings.Contains(result.Message, "seo_title") {
		t.Fatalf("result = %+v, want a pause naming the SEO title", result)
	}
	if writes := writesSince(server, from); len(writes) != 0 {
		t.Errorf("%d writes left before every guard was checked, want none", len(writes))
	}
	if held, _ := server.Lookup(id); held.Title != "Espresso" || held.Content != hisIntro {
		t.Errorf("a field was written beside a moved one: %+v", held)
	}
}

func TestEveryWriteOverAnExistingPageIsGuarded(t *testing.T) {
	t.Parallel()

	deps, server, sc, id := existingPage(t, []wptest.Option{wptest.WithSEOPlugin("yoast")}, wptest.Item{
		Title: "Espresso", Content: hisIntro, Meta: map[string]string{"_yoast_wpseo_title": "His SEO title"},
	})
	server.ResetRequests()
	runPublish(t, deps, sc)

	read := false
	for _, request := range server.Requests() {
		var sent map[string]any
		if len(request.Body) > 0 {
			if err := json.Unmarshal(request.Body, &sent); err != nil {
				t.Fatalf("decode %s: %v", request.Path, err)
			}
		}
		switch {
		case request.Method == http.MethodGet && strings.HasSuffix(request.Path, "/pages/"+itoa(id)):
			read = true
		case request.Method == http.MethodPost && strings.HasSuffix(request.Path, "/pages/"+itoa(id)):
			if !read {
				t.Errorf("the core update was sent without reading the values it writes over")
			}
		case request.Method == http.MethodPut && strings.HasSuffix(request.Path, "/raw"):
			if sent["expectedHash"] == nil {
				t.Errorf("the body was written without the hash it was read with")
			}
		case request.Method == http.MethodPut && strings.Contains(request.Path, "/seo-meta/"):
			expected, named := sent["expected"].(map[string]any)
			if !named {
				t.Errorf("the SEO meta was written with no expected hashes at all")
			}
			for name := range sent {
				if _, guarded := expected[name]; name != "expected" && !guarded {
					t.Errorf("the SEO field %s was written without the hash it was read with", name)
				}
			}
		}
	}
}

func TestAPublishKilledAfterItsSecondFieldResumesAtTheThird(t *testing.T) {
	t.Parallel()

	deps, server, sc, id := existingPage(t, []wptest.Option{wptest.WithSEOPlugin("yoast")}, wptest.Item{
		Title: "Espresso, the old title", Content: hisIntro, Meta: map[string]string{"_yoast_wpseo_title": "His SEO title"},
	})
	server.ResetRequests()
	seoWrites := 0
	result, err := driveLosing(t, steps.Publish(deps), sc, func(int) bool {
		count := 0
		for _, request := range writesSince(server, 0) {
			if strings.Contains(request.Path, "/seo-meta/") {
				count++
			}
		}
		lost := count == 1 && seoWrites == 0
		seoWrites = count
		return lost
	})
	if err != nil || result.Next == run.TransitionPause {
		t.Fatalf("Publish = %+v, %v; want it to finish after the lost answer", result, err)
	}
	held, _ := server.Lookup(id)
	if held.Meta["_yoast_wpseo_title"] != "espresso | Shop" || !strings.Contains(held.Content, "Espresso is a way") {
		t.Errorf("the run did not finish its fields: %+v", held)
	}
	bodies := 0
	for _, request := range writesSince(server, 0) {
		if request.Method == http.MethodPut && strings.HasSuffix(request.Path, "/raw") {
			bodies++
		}
	}
	if bodies != 1 {
		t.Errorf("the body was sent %d times, want once after the SEO answer was lost", bodies)
	}
	if published, ok := publishedIn(t, result); !ok || !slices.Contains(published.SEOApplied, "title") {
		t.Errorf("the record lost the SEO write whose answer was lost: %+v", published)
	}
}

func TestTheOpenGraphFieldsFollowTheirGuard(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		og   string
		want string
	}{
		{name: "the client's own Open Graph title is kept", og: "His own share title", want: "His own share title"},
		{name: "an empty one takes ours", og: "", want: "espresso | Shop"},
		{name: "one that followed the SEO title follows ours", og: "His SEO title", want: "espresso | Shop"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			meta := map[string]string{"_yoast_wpseo_title": "His SEO title"}
			if tc.og != "" {
				meta["_yoast_wpseo_opengraph-title"] = tc.og
			}
			deps, server, sc, id := existingPage(t, []wptest.Option{wptest.WithSEOPlugin("yoast")}, wptest.Item{
				Title: "Espresso", Content: hisIntro, Meta: meta,
			})
			sc.Artifacts[run.ArtifactMeta] = run.Artifact{Kind: run.ArtifactMeta, Blob: []byte(
				`{"title":"espresso | Shop","description":"Pull a shot.","ogTitle":"espresso | Shop","ogDescription":"Pull a shot."}`)}
			runPublish(t, deps, sc)
			if held, _ := server.Lookup(id); held.Meta["_yoast_wpseo_opengraph-title"] != tc.want {
				t.Errorf("the Open Graph title is %q, want %q", held.Meta["_yoast_wpseo_opengraph-title"], tc.want)
			}
		})
	}
}

func TestAnOutdatedPluginWritesNothingToAnExistingPage(t *testing.T) {
	t.Parallel()

	older := []string{"bulk", "seo_meta", "seo_meta_read", "content_hash", "raw", "preview", "rendered", "meta", "terms", "settings"}
	deps, server, sc, id := existingPage(t, []wptest.Option{wptest.WithPluginVersion("1.4.1"), wptest.WithCapabilities(older...)},
		wptest.Item{Title: "Espresso", Content: hisIntro})
	server.ResetRequests()

	result, err := drive(t, steps.Publish(deps), sc)
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}
	if result.Next != run.TransitionPause || !strings.Contains(result.Message, "too old") {
		t.Fatalf("result = %+v, want a pause saying the plugin is too old", result)
	}
	if writes := writesSince(server, 0); len(writes) != 0 {
		t.Errorf("%d writes reached a site whose plugin cannot guard them", len(writes))
	}
	if held, _ := server.Lookup(id); held.Content != hisIntro {
		t.Errorf("the body changed: %q", held.Content)
	}
}
