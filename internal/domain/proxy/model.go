package proxy

import (
	"context"
	"net/http"

	"github.com/davidmovas/postulator/internal/domain/entities"
)

type Service interface {
	GetSettings(ctx context.Context) (*entities.ProxySettings, error)
	UpdateSettings(ctx context.Context, settings *entities.ProxySettings) error
	GetState(ctx context.Context) *entities.ProxyState
	TestNode(ctx context.Context, node *entities.ProxyNode) *entities.ProxyHealth
	TestAllNodes(ctx context.Context) []entities.ProxyHealth
	DetectTor(ctx context.Context) *TorDetectionResult
	Enable(ctx context.Context) error
	Disable(ctx context.Context) error
	GetHTTPTransport() http.RoundTripper
	ApplyToClient(client HTTPClientWithProxy) error
	CompareIPs() *IPComparison
}

type IPComparison struct {
	DirectIP    string `json:"direct_ip"`
	DirectError string `json:"direct_error,omitempty"`
	ProxyIP     string `json:"proxy_ip"`
	ProxyError  string `json:"proxy_error,omitempty"`
	IsAnonymous bool   `json:"is_anonymous"`
}

type HTTPClientWithProxy interface {
	EnableProxy(proxyURL string)
	DisableProxy()
}

type TorDetectionResult struct {
	Found       bool   `json:"found"`
	Port        int    `json:"port"`
	ServiceType string `json:"service_type"`
}

type HealthChecker interface {
	Start(ctx context.Context) error
	Stop() error
	CheckNow(ctx context.Context)
}

type Rotator interface {
	Start(ctx context.Context) error
	Stop() error
	RotateNow(ctx context.Context)
	GetCurrentNode() *entities.ProxyNode
}
