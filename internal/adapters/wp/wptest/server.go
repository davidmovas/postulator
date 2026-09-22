package wptest

import (
	"bytes"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"sync"
	"time"
)

const (
	DefaultUser     = "postulator"
	DefaultPassword = "abcd EFGH 1234 ijkl"

	rootPath            = "/wp-json"
	pluginNamespaceName = "postulator/v1"

	maxRequestBytes = 32 << 20
)

type TB interface {
	Helper()
	Cleanup(func())
	Errorf(format string, args ...any)
	Logf(format string, args ...any)
}

type Request struct {
	Query  url.Values
	Header http.Header
	Method string
	Path   string
	Body   []byte
}

type Server struct {
	t             TB
	http          *httptest.Server
	items         map[int64]*Item
	categories    map[int64]*Category
	uploads       map[int64]*upload
	order         []int64
	categoryOrder []int64
	uploadOrder   []int64
	requests      []Request
	faults        []fault
	clock         time.Time
	now           func() time.Time
	user          string
	password      string
	address       string
	seoPlugin     string
	redirect      Redirect
	pendingEdit   *edit
	capabilities  []string
	nextID        int64
	previewSeq    int64
	noPlugin      bool
	brokenHash    bool
	brokenExpiry  bool
	noNamespaces  bool
	mu            sync.Mutex
}

type Option func(*Server)

func WithCredentials(user, password string) Option {
	return func(s *Server) {
		s.user = user
		s.password = password
	}
}

func WithSEOPlugin(name string) Option {
	return func(s *Server) { s.seoPlugin = name }
}

func WithoutPlugin() Option {
	return func(s *Server) { s.noPlugin = true }
}

func WithCapabilities(names ...string) Option {
	return func(s *Server) { s.capabilities = slices.Clone(names) }
}

func WithoutNamespaces() Option {
	return func(s *Server) { s.noNamespaces = true }
}

func WithRedirect(mode Redirect) Option {
	return func(s *Server) { s.redirect = mode }
}

func WithAddress(address string) Option {
	return func(s *Server) { s.address = address }
}

func WithClock(now func() time.Time) Option {
	return func(s *Server) { s.now = now }
}

func New(t TB, opts ...Option) *Server {
	t.Helper()

	server := &Server{
		t:            t,
		items:        make(map[int64]*Item),
		categories:   make(map[int64]*Category),
		uploads:      make(map[int64]*upload),
		clock:        startInstant.Add(-time.Second),
		user:         DefaultUser,
		password:     DefaultPassword,
		seoPlugin:    "yoast",
		capabilities: []string{"bulk", "seo_meta", "seo_meta_read", "content_hash", "raw", "preview"},
	}
	for _, opt := range opts {
		opt(server)
	}

	server.http = httptest.NewUnstartedServer(server.handler())
	if server.address != "" {
		listener, err := net.Listen("tcp", server.address)
		if err != nil {
			t.Errorf("listen on %s: %v", server.address, err)
			return server
		}
		if closeErr := server.http.Listener.Close(); closeErr != nil {
			t.Errorf("release the default listener: %v", closeErr)
		}
		server.http.Listener = listener
	}
	server.http.Start()

	t.Cleanup(server.http.Close)
	return server
}

func (s *Server) URL() string {
	return s.http.URL
}

func (s *Server) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET "+rootPath, s.handleRoot)
	s.routeCore(mux)
	s.routeCategories(mux)
	s.routeMedia(mux)
	s.routeWoo(mux)
	s.routePlugin(mux)

	return s.record(s.redirectRoot(s.injectFaults(s.authenticate(mux))))
}

func (s *Server) record(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(io.LimitReader(r.Body, maxRequestBytes))
		if err != nil {
			s.fail(w, http.StatusBadRequest, "rest_invalid_body", "the request body could not be read")
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(body))

		s.mu.Lock()
		s.requests = append(s.requests, Request{
			Method: r.Method,
			Path:   r.URL.Path,
			Query:  r.URL.Query(),
			Header: r.Header.Clone(),
			Body:   body,
		})
		s.mu.Unlock()

		next.ServeHTTP(w, r)
	})
}

func (s *Server) redirectRoot(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		mode := s.redirect
		s.mu.Unlock()

		location, status := redirectTarget(mode, r.Host, s.http.URL)
		if status == 0 || r.URL.Path != rootPath {
			next.ServeHTTP(w, r)
			return
		}

		w.Header().Set("Location", location)
		w.WriteHeader(status)
	})
}

func (s *Server) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, password, ok := r.BasicAuth()

		s.mu.Lock()
		wantUser, wantPassword := s.user, s.password
		s.mu.Unlock()

		if !ok || user != wantUser || password != wantPassword {
			s.fail(w, http.StatusUnauthorized, "rest_not_logged_in", "You are not currently logged in.")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleRoot(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	namespaces := []string{"oembed/1.0", "wp/v2", "wc/v3"}
	if !s.noPlugin {
		namespaces = append(namespaces, pluginNamespaceName)
	}
	if s.noNamespaces {
		namespaces = []string{}
	}
	s.mu.Unlock()

	s.respond(w, http.StatusOK, map[string]any{
		"name":        "Postulator Test Site",
		"description": "a fake WordPress installation",
		"url":         s.http.URL,
		"home":        s.http.URL,
		"namespaces":  namespaces,
	})
}

func (s *Server) respond(w http.ResponseWriter, status int, payload any) {
	body, err := json.Marshal(payload)
	if err != nil {
		s.t.Errorf("encode the fake response: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(status)
	if _, err = w.Write(body); err != nil {
		s.t.Logf("write the fake response: %v", err)
	}
}

func (s *Server) fail(w http.ResponseWriter, status int, code, message string) {
	s.respond(w, status, map[string]string{"code": code, "message": message})
}

func (s *Server) Requests() []Request {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.requests)
}

func (s *Server) LastRequest() (Request, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.requests) == 0 {
		return Request{}, false
	}
	return s.requests[len(s.requests)-1], true
}

func (s *Server) ResetRequests() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requests = nil
}

func (s *Server) EnablePlugin() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.noPlugin = false
}
