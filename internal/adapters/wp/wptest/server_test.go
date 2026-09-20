package wptest_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
)

func call(t *testing.T, server *wptest.Server, method, path string, body []byte, authenticated bool) (response *http.Response, payload []byte) {
	t.Helper()

	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}

	request, err := http.NewRequestWithContext(t.Context(), method, server.URL()+path, reader)
	if err != nil {
		t.Fatalf("build the request: %v", err)
	}
	if authenticated {
		request.SetBasicAuth(wptest.DefaultUser, wptest.DefaultPassword)
	}

	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	response, err = client.Do(request)
	if err != nil {
		t.Fatalf("send the request: %v", err)
	}
	defer func() { _ = response.Body.Close() }()

	payload, err = io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read the response: %v", err)
	}
	return response, payload
}

func decode(t *testing.T, payload []byte, out any) {
	t.Helper()

	if err := json.Unmarshal(payload, out); err != nil {
		t.Fatalf("decode %s: %v", payload, err)
	}
}

func TestTheRestRootListsItsNamespaces(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	response, payload := call(t, server, http.MethodGet, "/wp-json", nil, true)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.StatusCode)
	}

	var root struct {
		Name       string   `json:"name"`
		Home       string   `json:"home"`
		Namespaces []string `json:"namespaces"`
	}
	decode(t, payload, &root)

	want := []string{"oembed/1.0", "wp/v2", "wc/v3", "postulator/v1"}
	if len(root.Namespaces) != len(want) {
		t.Fatalf("namespaces = %v, want %v", root.Namespaces, want)
	}
	if root.Home != server.URL() {
		t.Errorf("home = %q, want %q", root.Home, server.URL())
	}
}

func TestTheRestRootCanHideThePluginAndTheNamespaces(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		option wptest.Option
		want   int
	}{
		{name: "without the plugin", option: wptest.WithoutPlugin(), want: 3},
		{name: "without any namespace", option: wptest.WithoutNamespaces(), want: 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server := wptest.New(t, tc.option)
			_, payload := call(t, server, http.MethodGet, "/wp-json", nil, true)

			var root struct {
				Namespaces []string `json:"namespaces"`
			}
			decode(t, payload, &root)
			if len(root.Namespaces) != tc.want {
				t.Errorf("namespaces = %v, want %d of them", root.Namespaces, tc.want)
			}
		})
	}
}

func TestEveryRouteDemandsTheApplicationPassword(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	response, payload := call(t, server, http.MethodGet, "/wp-json", nil, false)
	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", response.StatusCode)
	}

	var failure struct {
		Code string `json:"code"`
	}
	decode(t, payload, &failure)
	if failure.Code != "rest_not_logged_in" {
		t.Errorf("code = %q, want rest_not_logged_in", failure.Code)
	}
}

func TestTheCredentialsAreConfigurable(t *testing.T) {
	t.Parallel()

	server := wptest.New(t, wptest.WithCredentials("editor", "one two three"))
	response, _ := call(t, server, http.MethodGet, "/wp-json", nil, true)
	if response.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401 for the default credentials", response.StatusCode)
	}
}

func TestTheRootRedirectsWithoutAskingForCredentials(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		mode     wptest.Redirect
		status   int
		contains string
	}{
		{name: "https upgrade", mode: wptest.RedirectHTTPS, status: http.StatusMovedPermanently, contains: "https://"},
		{name: "login page", mode: wptest.RedirectLogin, status: http.StatusFound, contains: "wp-login.php"},
		{name: "admin", mode: wptest.RedirectAdmin, status: http.StatusFound, contains: "/wp-admin/"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server := wptest.New(t, wptest.WithRedirect(tc.mode))
			response, _ := call(t, server, http.MethodGet, "/wp-json", nil, false)
			if response.StatusCode != tc.status {
				t.Fatalf("status = %d, want %d", response.StatusCode, tc.status)
			}
			if location := response.Header.Get("Location"); !bytes.Contains([]byte(location), []byte(tc.contains)) {
				t.Errorf("location = %q, want it to contain %q", location, tc.contains)
			}
		})
	}
}

func TestFailNextFiresExactlyThatManyTimes(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	server.FailNext(http.StatusServiceUnavailable, 2)

	for attempt := range 3 {
		response, _ := call(t, server, http.MethodGet, "/wp-json", nil, true)
		want := http.StatusServiceUnavailable
		if attempt == 2 {
			want = http.StatusOK
		}
		if response.StatusCode != want {
			t.Errorf("attempt %d status = %d, want %d", attempt, response.StatusCode, want)
		}
	}
}

func TestRateLimitNextCarriesRetryAfter(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	server.RateLimitNext(3 * time.Second)

	response, _ := call(t, server, http.MethodGet, "/wp-json", nil, true)
	if response.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429", response.StatusCode)
	}
	if got := response.Header.Get("Retry-After"); got != "3" {
		t.Errorf("Retry-After = %q, want 3", got)
	}
}

func TestRequestsAreRecorded(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	call(t, server, http.MethodGet, "/wp-json?probe=1", nil, true)

	recorded, ok := server.LastRequest()
	if !ok {
		t.Fatal("no request was recorded")
	}
	if recorded.Method != http.MethodGet || recorded.Path != "/wp-json" {
		t.Errorf("recorded %s %s", recorded.Method, recorded.Path)
	}
	if recorded.Query.Get("probe") != "1" {
		t.Errorf("query = %v", recorded.Query)
	}
	if recorded.Header.Get("Authorization") == "" {
		t.Error("the recorder must keep the request headers")
	}

	server.ResetRequests()
	if len(server.Requests()) != 0 {
		t.Error("ResetRequests must empty the recording")
	}
}

func TestSeedingAssignsIdsSlugsAndAMonotonicClock(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	seeded := server.Seed(
		wptest.Item{Type: wptest.TypePage, Title: "Koffein"},
		wptest.Item{Type: wptest.TypePage, Title: "Powder"},
		wptest.Item{Type: wptest.TypePage, Title: "Powder"},
	)

	if len(seeded) != 3 {
		t.Fatalf("seeded %d items, want 3", len(seeded))
	}
	if seeded[0].ID != 1 || seeded[1].ID != 2 || seeded[2].ID != 3 {
		t.Errorf("ids = %d %d %d, want 1 2 3", seeded[0].ID, seeded[1].ID, seeded[2].ID)
	}
	if seeded[1].Slug != "powder" || seeded[2].Slug != "powder-2" {
		t.Errorf("slugs = %q %q, want powder and powder-2", seeded[1].Slug, seeded[2].Slug)
	}
	if !seeded[0].Modified.Before(seeded[1].Modified) {
		t.Error("the fake clock must advance for every stored item")
	}

	stored, ok := server.Lookup(seeded[2].ID)
	if !ok || stored.Slug != "powder-2" {
		t.Errorf("Lookup = %+v, %t", stored, ok)
	}
	if len(server.Items()) != 3 {
		t.Errorf("Items = %d, want 3", len(server.Items()))
	}
}

func TestASlugIsUniquePerParentForHierarchicalTypes(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	parent := server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Koffein"})[0]
	children := server.Seed(
		wptest.Item{Type: wptest.TypePage, Title: "Powder", Parent: parent.ID},
		wptest.Item{Type: wptest.TypePost, Title: "Powder"},
	)

	if children[0].Slug != "powder" {
		t.Errorf("a child under another parent may reuse the slug, got %q", children[0].Slug)
	}
	if children[1].Slug != "powder" {
		t.Errorf("a different type may reuse the slug, got %q", children[1].Slug)
	}
}

func itoa(id int64) string {
	return strconv.FormatInt(id, 10)
}

func newUpload(t *testing.T, server *wptest.Server, filename, contentType string, payload []byte) *http.Request {
	t.Helper()

	request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, server.URL()+"/wp-json/wp/v2/media", bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("build the upload: %v", err)
	}
	request.SetBasicAuth(wptest.DefaultUser, wptest.DefaultPassword)
	request.Header.Set("Content-Type", contentType)
	request.Header.Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	return request
}

func send(t *testing.T, request *http.Request) (response *http.Response, payload []byte) {
	t.Helper()

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("send the request: %v", err)
	}
	defer func() { _ = response.Body.Close() }()

	payload, err = io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read the response: %v", err)
	}
	return response, payload
}

func TestTheFakeClockAdvancesWithEveryStoredItem(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	before := server.Now()
	seeded := server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Koffein"})[0]

	if !server.Now().After(before) {
		t.Error("storing an item must advance the clock")
	}
	if !seeded.Modified.Equal(server.Now()) {
		t.Errorf("modified = %s, clock = %s", seeded.Modified, server.Now())
	}
}

type recorder struct {
	mu       sync.Mutex
	cleanups []func()
	errors   []string
	logs     []string
	helpers  int
}

func (r *recorder) Helper() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.helpers++
}

func (r *recorder) Cleanup(fn func()) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cleanups = append(r.cleanups, fn)
}

func (r *recorder) Errorf(format string, args ...any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.errors = append(r.errors, fmt.Sprintf(format, args...))
}

func (r *recorder) Logf(format string, args ...any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.logs = append(r.logs, fmt.Sprintf(format, args...))
}

func (r *recorder) close() {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, fn := range r.cleanups {
		fn()
	}
}

func TestTheServerRunsOutsideATest(t *testing.T) {
	t.Parallel()

	reported := &recorder{}
	server := wptest.New(reported, wptest.WithCredentials("harness", "open sesame 1234"))
	t.Cleanup(reported.close)

	if reported.helpers == 0 {
		t.Error("New must mark itself a helper")
	}
	if len(reported.cleanups) != 1 {
		t.Fatalf("New registered %d cleanups, want 1", len(reported.cleanups))
	}

	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL()+"/wp-json", http.NoBody)
	if err != nil {
		t.Fatalf("build the request: %v", err)
	}
	request.SetBasicAuth("harness", "open sesame 1234")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("call the fake site: %v", err)
	}
	defer func() {
		if closeErr := response.Body.Close(); closeErr != nil {
			t.Errorf("close the body: %v", closeErr)
		}
	}()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.StatusCode)
	}
	if len(reported.errors) != 0 {
		t.Fatalf("the fake site reported %v", reported.errors)
	}
}

func TestTheServerTakesAnAddress(t *testing.T) {
	t.Parallel()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve a port: %v", err)
	}
	address := listener.Addr().String()
	if closeErr := listener.Close(); closeErr != nil {
		t.Fatalf("release the port: %v", closeErr)
	}

	server := wptest.New(t, wptest.WithAddress(address))
	if want := "http://" + address; server.URL() != want {
		t.Fatalf("URL = %q, want %q", server.URL(), want)
	}

	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL()+"/wp-json", http.NoBody)
	if err != nil {
		t.Fatalf("build the request: %v", err)
	}
	request.SetBasicAuth(wptest.DefaultUser, wptest.DefaultPassword)

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("call the fake site: %v", err)
	}
	if closeErr := response.Body.Close(); closeErr != nil {
		t.Errorf("close the body: %v", closeErr)
	}
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.StatusCode)
	}
}
