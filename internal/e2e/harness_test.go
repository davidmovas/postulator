//go:build e2e

package e2e_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap/zaptest"

	"github.com/davidmovas/postulator/internal/adapters/llm/fake"
	"github.com/davidmovas/postulator/internal/app"
	"github.com/davidmovas/postulator/internal/runtime/steps"
)

const (
	pollInterval = 250 * time.Millisecond
	pollTimeout  = 4 * time.Minute

	guideJudge = `{"score":0.9,"issues":[],"suggestions":["Add a photograph of the plated dish."]}`

	repairSentence = `{"sentence":"It sits on our menu next to the other main courses we serve."}`
)

type target struct {
	path    string
	keyword string
}

var generated = []target{
	{path: "/menu/drinks/", keyword: "drinks and cocktails"},
	{path: "/menu/main-courses/steaks/", keyword: "grilled steaks"},
	{path: "/menu/main-courses/seafood/", keyword: "fresh seafood"},
	{path: "/menu/main-courses/pasta/", keyword: "handmade pasta"},
	{path: "/menu/desserts/", keyword: "house desserts"},
}

func draftOf(keyword string) string {
	return `{"title":"How we serve ` + keyword + `: Step-by-Step Guide | Shop",` +
		`"h1":"How we serve ` + keyword + `",` +
		`"sections":[` +
		`{"heading":"Introduction","html":"<p>This guide to ` + keyword + ` sits on our menu beside the rest ` +
		`of the main courses we serve, and it takes about ten minutes to read from start to finish.</p>"},` +
		`{"heading":"Step-by-Step Instructions","html":"<p>Pick the produce in the morning, season it simply, ` +
		`wait until the pan smokes and plate it with a side the kitchen prepared the same day.</p>"},` +
		`{"heading":"Tips and Common Mistakes","html":"<p>Never crowd the pan, never skip the resting time ` +
		`and never send a cold plate to a table that is waiting.</p>"},` +
		`{"heading":"Frequently Asked Questions","html":"<p>Ask the kitchen for the sides of the day before ` +
		`you order, and read the card once more if you are unsure.</p>"}],` +
		`"summary":"A short guide to ` + keyword + ` and where the dish sits on the card."}`
}

func metaOf(keyword string) string {
	return `{"title":"How we serve ` + keyword + `: Step-by-Step Guide | Shop",` +
		`"description":"What the kitchen does with ` + keyword + ` and what to order beside it.",` +
		`"canonical":"","ogTitle":"","ogDescription":""}`
}

func replies() []fake.Reply {
	out := make([]fake.Reply, 0, 2*len(generated)+2)
	for _, page := range generated {
		out = append(out,
			fake.Reply{Step: steps.NameGenerateBody, Match: page.path, Text: draftOf(page.keyword)},
			fake.Reply{Step: steps.NameGenerateMeta, Match: page.path, Text: metaOf(page.keyword)},
		)
	}
	return append(out,
		fake.Reply{Step: steps.NameJudge, Text: guideJudge},
		fake.Reply{Step: steps.NameRepairLinks, Text: repairSentence},
	)
}

type environment struct {
	baseURL string
	user    string
	pass    string
}

func repoRoot(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("working directory: %v", err)
	}
	for {
		if _, statErr := os.Stat(filepath.Join(dir, "go.mod")); statErr == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("no go.mod above %s", dir)
		}
		dir = parent
	}
}

func loadEnvironment(t *testing.T) environment {
	t.Helper()

	path := os.Getenv("POSTULATOR_E2E_ENV")
	if path == "" {
		path = filepath.Join(repoRoot(t), "docker", "e2e", ".env.generated")
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%s is missing; run `task e2e:up` first: %v", path, err)
	}

	values := make(map[string]string)
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			t.Fatalf("%s has a line without '=': %q", path, line)
		}
		values[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}

	env := environment{
		baseURL: strings.TrimSuffix(values["E2E_WP_URL"], "/"),
		user:    values["E2E_WP_USER"],
		pass:    values["E2E_WP_APP_PASSWORD"],
	}
	if env.baseURL == "" || env.user == "" || env.pass == "" {
		t.Fatalf("%s does not carry the site, the user and the application password", path)
	}
	return env
}

// companion holds the one manifest probe this package makes. The full loop needs the plugin
// and the degraded loop needs its absence, so each of them skips on the stack it cannot use.
var companion struct {
	once    sync.Once
	err     error
	present bool
}

func pluginIsActive(env environment) (bool, error) {
	companion.once.Do(func() {
		request, err := http.NewRequest(http.MethodGet, env.baseURL+"/wp-json/postulator/v1/manifest", http.NoBody)
		if err != nil {
			companion.err = err
			return
		}
		request.SetBasicAuth(env.user, env.pass)

		response, err := (&http.Client{Timeout: 60 * time.Second}).Do(request)
		if err != nil {
			companion.err = err
			return
		}
		defer func() {
			if closeErr := response.Body.Close(); closeErr != nil {
				companion.err = closeErr
			}
		}()

		companion.present = response.StatusCode == http.StatusOK
	})
	return companion.present, companion.err
}

func requirePlugin(t *testing.T, env environment, want bool) {
	t.Helper()

	present, err := pluginIsActive(env)
	if err != nil {
		t.Fatalf("ask %s for the companion manifest: %v", env.baseURL, err)
	}
	if present == want {
		return
	}
	if want {
		t.Skipf("the companion plugin is not active on %s; provision the stack with E2E_PLUGIN=1 "+
			"(the default) to run the full loop", env.baseURL)
	}
	t.Skipf("the companion plugin is active on %s; provision the stack with E2E_PLUGIN=0 "+
		"(`task e2e:full:noplugin`) to run the degraded loop", env.baseURL)
}

type link struct {
	Href   string `json:"href"`
	Anchor string `json:"anchor"`
}

type itemMeta struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Canonical   string `json:"canonical"`
}

type contentItem struct {
	ID     int      `json:"id"`
	Type   string   `json:"type"`
	Slug   string   `json:"slug"`
	Path   string   `json:"path"`
	Parent int      `json:"parent"`
	Status string   `json:"status"`
	Title  string   `json:"title"`
	H1     string   `json:"h1"`
	Meta   itemMeta `json:"meta"`
	Links  []link   `json:"links"`
}

type contentPage struct {
	Items      []contentItem `json:"items"`
	NextCursor *string       `json:"nextCursor"`
}

type site struct {
	env  environment
	http *http.Client
}

func newSite(t *testing.T) *site {
	t.Helper()
	return &site{env: loadEnvironment(t), http: &http.Client{Timeout: 60 * time.Second}}
}

func (s *site) call(t *testing.T, method, path string, payload any, status int, out any) {
	t.Helper()

	var sent io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("encode %s %s: %v", method, path, err)
		}
		sent = bytes.NewReader(encoded)
	}

	request, err := http.NewRequestWithContext(t.Context(), method, s.env.baseURL+path, sent)
	if err != nil {
		t.Fatalf("build %s %s: %v", method, path, err)
	}
	request.SetBasicAuth(s.env.user, s.env.pass)
	if payload != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	response, err := s.http.Do(request)
	if err != nil {
		t.Fatalf("call %s %s: %v", method, path, err)
	}
	defer func() {
		if closeErr := response.Body.Close(); closeErr != nil {
			t.Errorf("close %s %s: %v", method, path, closeErr)
		}
	}()

	raw, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read %s %s: %v", method, path, err)
	}
	if response.StatusCode != status {
		t.Fatalf("%s %s: status %d, want %d, body %s", method, path, response.StatusCode, status, raw)
	}
	if out == nil {
		return
	}
	if err = json.Unmarshal(raw, out); err != nil {
		t.Fatalf("decode %s %s: %v, body %s", method, path, err, raw)
	}
}

func (s *site) publish(t *testing.T, title, slug, body string, parent int) int {
	t.Helper()

	payload := map[string]any{"title": title, "slug": slug, "content": body, "status": "publish"}
	if parent != 0 {
		payload["parent"] = parent
	}

	var created struct {
		ID int `json:"id"`
	}
	s.call(t, http.MethodPost, "/wp-json/wp/v2/pages", payload, http.StatusCreated, &created)
	if created.ID == 0 {
		t.Fatalf("the site created no page for %s", slug)
	}
	return created.ID
}

func (s *site) content(t *testing.T) []contentItem {
	t.Helper()

	items := make([]contentItem, 0, 32)
	path := "/wp-json/postulator/v1/content?types=page"
	for {
		var page contentPage
		s.call(t, http.MethodGet, path, nil, http.StatusOK, &page)
		items = append(items, page.Items...)
		if page.NextCursor == nil || *page.NextCursor == "" {
			return items
		}
		path = "/wp-json/postulator/v1/content?types=page&cursor=" + *page.NextCursor
	}
}

func (s *site) byPath(t *testing.T, path string) (contentItem, bool) {
	t.Helper()

	listed := s.content(t)
	for i := range listed {
		if listed[i].Path == path {
			return listed[i], true
		}
	}
	return contentItem{}, false
}

func openCore(t *testing.T) *app.Core {
	t.Helper()

	home := t.TempDir()
	core, err := app.Open(t.Context(), app.Config{
		DatabasePath: filepath.Join(home, "postulator.db"),
		KeyDir:       home,
		Provider:     fake.NewScripted(replies()...),
	}, zaptest.NewLogger(t))
	if err != nil {
		t.Fatalf("open the application: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := core.Close(); closeErr != nil {
			t.Errorf("close the application: %v", closeErr)
		}
	})
	return core
}

func waitFor(t *testing.T, what string, done func() bool) {
	t.Helper()

	deadline := time.Now().Add(pollTimeout)
	for time.Now().Before(deadline) {
		if done() {
			return
		}
		time.Sleep(pollInterval)
	}
	t.Fatalf("timed out waiting for %s", what)
}
