package wptest

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"time"
)

type Redirect string

const (
	RedirectNone  Redirect = ""
	RedirectHTTPS Redirect = "https"
	RedirectLogin Redirect = "login"
	RedirectAdmin Redirect = "admin"
)

type fault struct {
	retryAfter time.Duration
	status     int
	afterWrite bool
}

func (s *Server) FailNext(status, times int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for range times {
		s.faults = append(s.faults, fault{status: status})
	}
}

func (s *Server) FailAfterNext(status, times int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for range times {
		s.faults = append(s.faults, fault{status: status, afterWrite: true})
	}
}

func (s *Server) RateLimitNext(retryAfter time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.faults = append(s.faults, fault{status: http.StatusTooManyRequests, retryAfter: retryAfter})
}

func (s *Server) takeFault() (fault, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.faults) == 0 {
		return fault{}, false
	}
	next := s.faults[0]
	s.faults = s.faults[1:]
	return next, true
}

func (s *Server) injectFaults(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		injected, ok := s.takeFault()
		if !ok {
			next.ServeHTTP(w, r)
			return
		}

		if injected.afterWrite {
			next.ServeHTTP(httptest.NewRecorder(), r)
		}
		if injected.retryAfter > 0 {
			w.Header().Set("Retry-After", strconv.Itoa(int(injected.retryAfter.Seconds())))
		}
		s.fail(w, injected.status, faultCode(injected.status), "the injected fault fired")
	})
}

func faultCode(status int) string {
	if status == http.StatusTooManyRequests {
		return "too_many_requests"
	}
	return "internal_server_error"
}

func redirectTarget(mode Redirect, host, base string) (location string, status int) {
	switch mode {
	case RedirectHTTPS:
		return "https://" + host + rootPath, http.StatusMovedPermanently
	case RedirectLogin:
		return base + "/wp-login.php?redirect_to=%2Fwp-json", http.StatusFound
	case RedirectAdmin:
		return base + "/wp-admin/", http.StatusFound
	default:
		return "", 0
	}
}

func WithBrokenContentHash() Option {
	return func(s *Server) { s.brokenHash = true }
}

func WithBrokenPreviewExpiry() Option {
	return func(s *Server) { s.brokenExpiry = true }
}
