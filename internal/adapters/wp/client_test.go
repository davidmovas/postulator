package wp

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"slices"
	"testing"
	"time"

	"golang.org/x/time/rate"

	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/settings"
)

func TestNewRejectsAnUnusableConfig(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		config Config
	}{
		{name: "empty base url", config: Config{Username: "u", AppPassword: "p"}},
		{name: "no host", config: Config{BaseURL: "not a url", Username: "u", AppPassword: "p"}},
		{name: "plain http without consent", config: Config{BaseURL: "http://example.com", Username: "u", AppPassword: "p"}},
		{name: "unknown scheme", config: Config{BaseURL: "ftp://example.com", Username: "u", AppPassword: "p"}},
		{name: "no user", config: Config{BaseURL: "https://example.com", AppPassword: "p"}},
		{name: "no application password", config: Config{BaseURL: "https://example.com", Username: "u"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			client, err := New(tc.config)
			if client != nil {
				t.Fatal("a rejected config must not produce a client")
			}
			if !errors.IsCode(err, errors.Invalid) {
				t.Errorf("code = %q, want %q", errors.CodeOf(err), errors.Invalid)
			}
		})
	}
}

func TestNewAcceptsPlainHTTPOnlyWithConsent(t *testing.T) {
	t.Parallel()

	client, err := New(Config{BaseURL: "http://example.com/blog/", Username: "u", AppPassword: "p", AllowInsecure: true})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if got := client.resolve(coreNamespace, "/pages", nil); got != "http://example.com/blog/wp-json/wp/v2/pages" {
		t.Errorf("resolve = %q", got)
	}
}

func TestNewAppliesTheDefaults(t *testing.T) {
	t.Parallel()

	client, err := New(Config{BaseURL: "https://example.com", Username: "u", AppPassword: "p"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if client.http.Timeout != DefaultTimeout {
		t.Errorf("timeout = %s, want %s", client.http.Timeout, DefaultTimeout)
	}
	if client.retries != DefaultRetries {
		t.Errorf("retries = %d, want %d", client.retries, DefaultRetries)
	}
	if client.limiter.Limit() != rate.Limit(DefaultRateLimitPerSecond) {
		t.Errorf("rate = %v, want %v", client.limiter.Limit(), rate.Limit(DefaultRateLimitPerSecond))
	}
	if client.transport.TLSClientConfig != nil {
		t.Error("the transport must keep the standard TLS configuration")
	}
	if client.transport.Proxy != nil {
		t.Error("no proxy is configured, so the transport must not consult one")
	}
	if client.http.CheckRedirect != nil {
		t.Error("ordinary calls follow redirects")
	}
	if client.probe.CheckRedirect == nil {
		t.Error("the probe client must refuse to follow redirects")
	}
	if client.http.Transport != client.probe.Transport {
		t.Error("both clients must share one transport")
	}
}

func TestTheLastOptionOfAKindWins(t *testing.T) {
	t.Parallel()

	client, err := New(
		Config{BaseURL: "https://example.com", Username: "u", AppPassword: "p"},
		WithTimeout(time.Second), WithTimeout(7*time.Second),
		WithRetries(9), WithRetries(1),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if client.http.Timeout != 7*time.Second {
		t.Errorf("timeout = %s, want 7s", client.http.Timeout)
	}
	if client.retries != 1 {
		t.Errorf("retries = %d, want 1", client.retries)
	}
}

func TestWithRateLimitZeroLiftsTheLimit(t *testing.T) {
	t.Parallel()

	client, err := New(Config{BaseURL: "https://example.com", Username: "u", AppPassword: "p"}, WithRateLimit(0))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if client.limiter.Limit() != rate.Inf {
		t.Errorf("rate = %v, want infinite", client.limiter.Limit())
	}
}

func TestTheHTTPProxyReachesTheTransport(t *testing.T) {
	t.Parallel()

	client, err := New(Config{BaseURL: "https://example.com", Username: "u", AppPassword: "p"}, WithProxy("http://127.0.0.1:3128"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if client.transport.Proxy == nil {
		t.Fatal("the transport has no proxy function")
	}

	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://example.com/wp-json", http.NoBody)
	if err != nil {
		t.Fatalf("build the request: %v", err)
	}
	proxied, err := client.transport.Proxy(request)
	if err != nil {
		t.Fatalf("proxy lookup: %v", err)
	}
	if proxied == nil || proxied.Host != "127.0.0.1:3128" {
		t.Errorf("proxy = %v, want 127.0.0.1:3128", proxied)
	}
}

func TestAnUnusableProxyURLIsRejected(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		proxy string
	}{
		{name: "no host", proxy: "http://"},
		{name: "unknown scheme", proxy: "ftp://127.0.0.1:1080"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := New(Config{BaseURL: "https://example.com", Username: "u", AppPassword: "p"}, WithProxy(tc.proxy))
			if !errors.IsCode(err, errors.Invalid) {
				t.Errorf("code = %q, want %q", errors.CodeOf(err), errors.Invalid)
			}
		})
	}
}

type socksRequest struct {
	methods []byte
	host    string
	port    uint16
}

func socksListener(t *testing.T) (address string, seen <-chan socksRequest) {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })

	recorded := make(chan socksRequest, 1)
	go func() {
		conn, acceptErr := listener.Accept()
		if acceptErr != nil {
			return
		}
		defer func() { _ = conn.Close() }()

		request, readErr := readSocksRequest(conn)
		if readErr != nil {
			return
		}
		recorded <- request
	}()
	return listener.Addr().String(), recorded
}

func readSocksRequest(conn net.Conn) (socksRequest, error) {
	greeting := make([]byte, 2)
	if _, err := io.ReadFull(conn, greeting); err != nil {
		return socksRequest{}, err
	}
	methods := make([]byte, greeting[1])
	if _, err := io.ReadFull(conn, methods); err != nil {
		return socksRequest{}, err
	}
	if _, err := conn.Write([]byte{0x05, 0x00}); err != nil {
		return socksRequest{}, err
	}

	header := make([]byte, 5)
	if _, err := io.ReadFull(conn, header); err != nil {
		return socksRequest{}, err
	}
	name := make([]byte, header[4])
	if _, err := io.ReadFull(conn, name); err != nil {
		return socksRequest{}, err
	}
	port := make([]byte, 2)
	if _, err := io.ReadFull(conn, port); err != nil {
		return socksRequest{}, err
	}
	return socksRequest{methods: methods, host: string(name), port: binary.BigEndian.Uint16(port)}, nil
}

func TestTheSocksProxyDialerCarriesTheConnect(t *testing.T) {
	t.Parallel()

	address, seen := socksListener(t)
	client, err := New(
		Config{BaseURL: "https://example.com", Username: "u", AppPassword: "p"},
		WithProxy("socks5://operator:secret@"+address),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if client.transport.Proxy != nil {
		t.Error("a socks5 proxy is a dialer, not an http proxy function")
	}

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	conn, err := client.transport.DialContext(ctx, "tcp", "example.invalid:443")
	if err == nil {
		_ = conn.Close()
	}

	select {
	case request := <-seen:
		if request.host != "example.invalid" {
			t.Errorf("host = %q, want example.invalid", request.host)
		}
		if request.port != 443 {
			t.Errorf("port = %d, want 443", request.port)
		}
		if !slices.Contains(request.methods, byte(0x02)) {
			t.Error("the dialer did not offer username and password authentication")
		}
	case <-ctx.Done():
		t.Fatal("the socks5 proxy never saw a connect request")
	}
}

func TestFromSettingsConfiguresTheClient(t *testing.T) {
	t.Parallel()

	registry := settings.Default()
	values := registry.NewValues()
	unknown, err := registry.Apply(values, map[string]json.RawMessage{
		"wp.timeout":            json.RawMessage(`"9s"`),
		"wp.retries":            json.RawMessage(`2`),
		"wp.rateLimitPerSecond": json.RawMessage(`7`),
		"wp.proxyUrl":           json.RawMessage(`"http://127.0.0.1:3128"`),
	})
	if err != nil {
		t.Fatalf("apply the settings: %v", err)
	}
	if len(unknown) != 0 {
		t.Fatalf("unknown settings %v", unknown)
	}

	client, err := New(Config{BaseURL: "https://example.com", Username: "u", AppPassword: "p"}, FromSettings(values)...)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if client.http.Timeout != 9*time.Second {
		t.Errorf("timeout = %s, want 9s", client.http.Timeout)
	}
	if client.retries != 2 {
		t.Errorf("retries = %d, want 2", client.retries)
	}
	if client.limiter.Limit() != rate.Limit(7) {
		t.Errorf("rate = %v, want 7", client.limiter.Limit())
	}
	if client.transport.Proxy == nil {
		t.Error("the proxy setting did not reach the transport")
	}
}

func TestTheProxySettingRejectsRubbish(t *testing.T) {
	t.Parallel()

	registry := settings.Default()
	_, err := registry.Apply(registry.NewValues(), map[string]json.RawMessage{
		"wp.proxyUrl": json.RawMessage(`"ftp://127.0.0.1:1080"`),
	})
	if !errors.IsCode(err, errors.Invalid) {
		t.Errorf("code = %q, want %q", errors.CodeOf(err), errors.Invalid)
	}
}

func TestTheDefaultBackoffGrowsAndIsCapped(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		attempt int
		want    time.Duration
	}{
		{name: "first retry", attempt: 0, want: 500 * time.Millisecond},
		{name: "second retry", attempt: 1, want: time.Second},
		{name: "third retry", attempt: 2, want: 2 * time.Second},
		{name: "fourth retry", attempt: 3, want: 4 * time.Second},
		{name: "capped", attempt: 9, want: 8 * time.Second},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := defaultBackoff(tc.attempt); got != tc.want {
				t.Errorf("defaultBackoff(%d) = %s, want %s", tc.attempt, got, tc.want)
			}
		})
	}
}
