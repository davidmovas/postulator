package browser

import (
	"context"
	"net/url"
	"strings"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const MissingCode = "tor_missing"

type opener interface {
	Locate(configured string) (path, source string, ok bool)
	Open(path, address string) error
}

type Deps struct {
	Browser opener
	TorPath func() string
}

type Service struct {
	deps Deps
}

func New(deps Deps) *Service {
	return &Service{deps: deps}
}

func (s *Service) configured() string {
	if s.deps.TorPath == nil {
		return ""
	}
	return s.deps.TorPath()
}

func (s *Service) Open(_ context.Context, req OpenRequest) (OpenResponse, error) {
	address, err := web(req.URL)
	if err != nil {
		return OpenResponse{}, err
	}

	path, _, found := s.deps.Browser.Locate(s.configured())
	if !found {
		return OpenResponse{}, errors.New(errors.Invalid, "Tor Browser is not installed where this application looked").
			WithDetail("code", MissingCode)
	}
	if openErr := s.deps.Browser.Open(path, address); openErr != nil {
		return OpenResponse{}, openErr
	}
	return OpenResponse{}, nil
}

func (s *Service) Locate(_ context.Context, _ LocateRequest) (LocateResponse, error) {
	path, source, found := s.deps.Browser.Locate(s.configured())
	return LocateResponse{Path: path, Source: source, Installed: found}, nil
}

func web(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", invalid("a link needs an address")
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", invalid("that address cannot be read")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", invalid("only an http or https address opens in a browser")
	}
	if parsed.Host == "" {
		return "", invalid("that address names no host")
	}
	return trimmed, nil
}

func invalid(message string) error {
	return errors.New(errors.Invalid, message).WithDetail("field", "url")
}
