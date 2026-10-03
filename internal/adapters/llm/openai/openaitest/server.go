package openaitest

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
)

const (
	DefaultKey   = "test-key"
	DefaultModel = "gpt-5.6-terra"

	basePath        = "/v1"
	responsesPath   = basePath + "/responses"
	maxRequestBytes = 32 << 20
)

type TB interface {
	Helper()
	Cleanup(func())
	Errorf(format string, args ...any)
}

type Request struct {
	Header     http.Header
	Body       map[string]any
	Method     string
	Path       string
	Raw        []byte
	Authorized bool
}

type Server struct {
	t        TB
	http     *httptest.Server
	key      string
	replies  []Reply
	requests []Request
	mu       sync.Mutex
}

type Option func(*Server)

func WithKey(key string) Option {
	return func(s *Server) { s.key = key }
}

func New(t TB, opts ...Option) *Server {
	t.Helper()

	server := &Server{t: t, key: DefaultKey}
	for _, opt := range opts {
		opt(server)
	}
	server.http = httptest.NewServer(http.HandlerFunc(server.serve))
	t.Cleanup(server.http.Close)
	return server
}

func (s *Server) URL() string {
	return s.http.URL + basePath
}

func (s *Server) Close() {
	s.http.Close()
}

func (s *Server) Enqueue(replies ...Reply) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.replies = append(s.replies, replies...)
}

func (s *Server) Pending() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.replies)
}

func (s *Server) Requests() []Request {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.requests)
}

func (s *Server) serve(w http.ResponseWriter, r *http.Request) {
	raw, err := io.ReadAll(io.LimitReader(r.Body, maxRequestBytes))
	if err != nil {
		Failure(http.StatusBadRequest, Fault{Type: invalidRequest, Message: "The request body could not be read."}).serve(w, r)
		return
	}
	recorded := s.record(r, raw)

	switch {
	case r.Method != http.MethodPost || r.URL.Path != responsesPath:
		Failure(http.StatusNotFound, Fault{Type: invalidRequest, Message: "Invalid URL (" + r.Method + " " + r.URL.Path + ")"}).serve(w, r)
	case !recorded.Authorized:
		Failure(http.StatusUnauthorized, Fault{
			Type: invalidRequest, Code: "invalid_api_key",
			Message: "Incorrect API key provided. You can find your API key at https://platform.openai.com/account/api-keys.",
		}).serve(w, r)
	case recorded.Body == nil:
		Failure(http.StatusBadRequest, Fault{
			Type:    invalidRequest,
			Message: "We could not parse the JSON body of your request. The OpenAI API expects a JSON payload.",
		}).serve(w, r)
	default:
		if refusal, refused := refuse(recorded.Body); refused {
			refusal.serve(w, r)
			return
		}
		s.next().serve(w, r)
	}
}

func (s *Server) record(r *http.Request, raw []byte) Request {
	header := r.Header.Clone()
	authorized := header.Get("Authorization") == "Bearer "+s.key
	header.Del("Authorization")

	var body map[string]any
	if json.Unmarshal(raw, &body) != nil {
		body = nil
	}

	recorded := Request{Header: header, Body: body, Method: r.Method, Path: r.URL.Path, Raw: raw, Authorized: authorized}
	s.mu.Lock()
	s.requests = append(s.requests, recorded)
	s.mu.Unlock()
	return recorded
}

func (s *Server) next() Reply {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.replies) == 0 {
		s.t.Errorf("the fake OpenAI server received a request it had no reply for")
		return Failure(http.StatusInternalServerError, Fault{Type: "server_error", Message: "The fake server has no reply scripted for this request."})
	}
	reply := s.replies[0]
	s.replies = s.replies[1:]
	return reply
}
