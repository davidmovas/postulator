package wp

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"go.uber.org/zap"
	"golang.org/x/time/rate"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	rootPath        = "/wp-json"
	coreNamespace   = "/wp-json/wp/v2"
	wooNamespace    = "/wp-json/wc/v3"
	pluginNamespace = "/wp-json/postulator/v1"
)

type Config struct {
	BaseURL       string
	Username      string
	AppPassword   string
	AllowInsecure bool
}

type Client struct {
	base      *url.URL
	http      *http.Client
	probe     *http.Client
	transport *http.Transport
	limiter   *rate.Limiter
	backoff   func(int) time.Duration
	logger    Logger
	username  string
	password  string
	retries   int
}

func New(cfg Config, opts ...Option) (*Client, error) {
	base, err := baseURL(cfg)
	if err != nil {
		return nil, err
	}
	if cfg.Username == "" || cfg.AppPassword == "" {
		return nil, errors.New(errors.Invalid, "the site needs a WordPress user name and an application password")
	}

	resolved := options{
		timeout: DefaultTimeout,
		retries: DefaultRetries,
		rate:    DefaultRateLimitPerSecond,
		backoff: defaultBackoff,
		logger:  zap.NewNop(),
	}
	for _, opt := range opts {
		opt(&resolved)
	}
	if resolved.retries < 0 {
		return nil, errors.New(errors.Invalid, "the retry count must not be negative")
	}

	transport, err := newTransport(resolved.proxyURL)
	if err != nil {
		return nil, err
	}

	return &Client{
		base:      base,
		http:      &http.Client{Transport: transport, Timeout: resolved.timeout},
		probe:     &http.Client{Transport: transport, Timeout: resolved.timeout, CheckRedirect: keepRedirect},
		transport: transport,
		limiter:   newLimiter(resolved.rate),
		backoff:   resolved.backoff,
		logger:    resolved.logger,
		username:  cfg.Username,
		password:  cfg.AppPassword,
		retries:   resolved.retries,
	}, nil
}

func keepRedirect(*http.Request, []*http.Request) error {
	return http.ErrUseLastResponse
}

func baseURL(cfg Config) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimRight(cfg.BaseURL, "/"))
	if err != nil || parsed.Host == "" {
		return nil, errors.New(errors.Invalid, "the site base URL is not a valid absolute URL")
	}

	switch parsed.Scheme {
	case "https":
		return parsed, nil
	case "http":
		if cfg.AllowInsecure {
			return parsed, nil
		}
		return nil, errors.New(errors.Invalid, "the site base URL must use https unless the site is marked as insecure")
	default:
		return nil, errors.New(errors.Invalid, "the site base URL must use http or https").WithDetail("scheme", parsed.Scheme)
	}
}

func (c *Client) resolve(namespace, path string, query url.Values) string {
	target := *c.base
	target.Path = c.base.Path + namespace + path
	if len(query) > 0 {
		target.RawQuery = query.Encode()
	}
	return target.String()
}
