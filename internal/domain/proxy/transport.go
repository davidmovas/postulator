package proxy

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/davidmovas/postulator/internal/domain/entities"
	"golang.org/x/net/proxy"
)

type ChainTransport struct {
	nodes     []entities.ProxyNode
	transport http.RoundTripper
}

func NewChainTransport(nodes []entities.ProxyNode) (*ChainTransport, error) {
	if len(nodes) == 0 {
		return &ChainTransport{
			transport: http.DefaultTransport,
		}, nil
	}

	transport, err := buildChainedTransport(nodes)
	if err != nil {
		return nil, err
	}

	return &ChainTransport{
		nodes:     nodes,
		transport: transport,
	}, nil
}

func (t *ChainTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return t.transport.RoundTrip(req)
}

func buildChainedTransport(nodes []entities.ProxyNode) (http.RoundTripper, error) {
	if len(nodes) == 0 {
		return http.DefaultTransport, nil
	}

	if len(nodes) == 1 {
		return buildSingleNodeTransport(nodes[0])
	}

	return buildMultiNodeTransport(nodes)
}

func buildSingleNodeTransport(node entities.ProxyNode) (http.RoundTripper, error) {
	switch node.Type {
	case entities.ProxyTypeHTTP:
		return buildHTTPProxyTransport(node)
	case entities.ProxyTypeSOCKS5, entities.ProxyTypeTor:
		return buildSOCKS5Transport(node)
	default:
		return nil, fmt.Errorf("unsupported proxy type: %s", node.Type)
	}
}

func buildHTTPProxyTransport(node entities.ProxyNode) (http.RoundTripper, error) {
	proxyURL := buildProxyURL(node)
	parsedURL, err := url.Parse(proxyURL)
	if err != nil {
		return nil, fmt.Errorf("invalid proxy URL: %w", err)
	}

	return &http.Transport{
		Proxy: http.ProxyURL(parsedURL),
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
		IdleConnTimeout:       90 * time.Second,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
	}, nil
}

func buildSOCKS5Transport(node entities.ProxyNode) (http.RoundTripper, error) {
	addr := fmt.Sprintf("%s:%d", node.Host, node.Port)

	var auth *proxy.Auth
	if node.Username != "" {
		auth = &proxy.Auth{
			User:     node.Username,
			Password: node.Password,
		}
	}

	dialer, err := proxy.SOCKS5("tcp", addr, auth, proxy.Direct)
	if err != nil {
		return nil, fmt.Errorf("failed to create SOCKS5 dialer: %w", err)
	}

	return &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialer.Dial(network, addr)
		},
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
		IdleConnTimeout:       90 * time.Second,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
	}, nil
}

func buildMultiNodeTransport(nodes []entities.ProxyNode) (http.RoundTripper, error) {
	if len(nodes) < 2 {
		return buildSingleNodeTransport(nodes[0])
	}

	var chain proxy.Dialer = proxy.Direct

	for i := len(nodes) - 1; i >= 0; i-- {
		node := nodes[i]
		addr := fmt.Sprintf("%s:%d", node.Host, node.Port)

		var auth *proxy.Auth
		if node.Username != "" {
			auth = &proxy.Auth{
				User:     node.Username,
				Password: node.Password,
			}
		}

		var err error
		chain, err = proxy.SOCKS5("tcp", addr, auth, chain)
		if err != nil {
			return nil, fmt.Errorf("failed to create chain dialer for node %s: %w", node.ID, err)
		}
	}

	return &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return chain.Dial(network, addr)
		},
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
		IdleConnTimeout:       90 * time.Second,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
	}, nil
}

func buildProxyURL(node entities.ProxyNode) string {
	scheme := "http"
	if node.Type == entities.ProxyTypeSOCKS5 || node.Type == entities.ProxyTypeTor {
		scheme = "socks5"
	}

	if node.Username != "" {
		return fmt.Sprintf("%s://%s:%s@%s:%d", scheme, node.Username, node.Password, node.Host, node.Port)
	}

	return fmt.Sprintf("%s://%s:%d", scheme, node.Host, node.Port)
}

func BuildProxyURLForNode(node entities.ProxyNode) string {
	return buildProxyURL(node)
}
