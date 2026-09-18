package wp

import (
	"net"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/net/proxy"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func newTransport(proxyURL string) (*http.Transport, error) {
	transport := &http.Transport{
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          16,
		MaxIdleConnsPerHost:   4,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: time.Second,
		DialContext:           (&net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
	}
	if proxyURL == "" {
		return transport, nil
	}

	parsed, err := url.Parse(proxyURL)
	if err != nil || parsed.Host == "" {
		return nil, errors.New(errors.Invalid, "the proxy URL is not a valid absolute URL")
	}

	switch parsed.Scheme {
	case "http", "https":
		transport.Proxy = http.ProxyURL(parsed)
		return transport, nil
	case "socks5", "socks5h":
		return socksTransport(transport, parsed)
	default:
		return nil, errors.New(errors.Invalid, "the proxy URL must use http, https or socks5").WithDetail("scheme", parsed.Scheme)
	}
}

func socksTransport(transport *http.Transport, parsed *url.URL) (*http.Transport, error) {
	dialer, err := proxy.SOCKS5("tcp", parsed.Host, socksAuth(parsed), proxy.Direct)
	if err != nil {
		return nil, errors.New(errors.Invalid, "the socks5 proxy cannot be used").WithInternal(err)
	}

	contextDialer, ok := dialer.(proxy.ContextDialer)
	if !ok {
		return nil, errors.New(errors.Internal, "the socks5 proxy dialer does not support contexts")
	}
	transport.DialContext = contextDialer.DialContext
	return transport, nil
}

func socksAuth(parsed *url.URL) *proxy.Auth {
	if parsed.User == nil {
		return nil
	}
	password, _ := parsed.User.Password()
	return &proxy.Auth{User: parsed.User.Username(), Password: password}
}
