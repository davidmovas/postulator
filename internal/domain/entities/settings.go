package entities

import (
	"time"

	"github.com/davidmovas/postulator/pkg/errors"
)

const (
	SettingsKeyHealthCheck = "health_check"
	SettingsKeyProxy       = "proxy"
)

type ProxyType string

const (
	ProxyTypeHTTP   ProxyType = "http"
	ProxyTypeSOCKS5 ProxyType = "socks5"
	ProxyTypeTor    ProxyType = "tor"
)

type ProxyStatus string

const (
	ProxyStatusDisabled    ProxyStatus = "disabled"
	ProxyStatusConnecting  ProxyStatus = "connecting"
	ProxyStatusConnected   ProxyStatus = "connected"
	ProxyStatusError       ProxyStatus = "error"
	ProxyStatusTorNotFound ProxyStatus = "tor_not_found"
)

type ProxyNode struct {
	ID       string    `json:"id"`
	Type     ProxyType `json:"type"`
	Host     string    `json:"host"`
	Port     int       `json:"port"`
	Username string    `json:"username,omitempty"`
	Password string    `json:"password,omitempty"`
	Enabled  bool      `json:"enabled"`
	Order    int       `json:"order"`
}

type ProxySettings struct {
	Enabled             bool        `json:"enabled"`
	Mode                string      `json:"mode"`
	Nodes               []ProxyNode `json:"nodes"`
	RotationEnabled     bool        `json:"rotation_enabled"`
	RotationInterval    int         `json:"rotation_interval"`
	HealthCheckEnabled  bool        `json:"health_check_enabled"`
	HealthCheckInterval int         `json:"health_check_interval"`
	NotifyOnFailure     bool        `json:"notify_on_failure"`
	NotifyOnRecover     bool        `json:"notify_on_recover"`
	CurrentNodeID       string      `json:"current_node_id,omitempty"`
}

type ProxyHealth struct {
	NodeID      string      `json:"node_id"`
	Status      ProxyStatus `json:"status"`
	LatencyMs   int         `json:"latency_ms"`
	LastChecked int64       `json:"last_checked"`
	Error       string      `json:"error,omitempty"`
	ExternalIP  string      `json:"external_ip,omitempty"`
}

type ProxyState struct {
	Status        ProxyStatus   `json:"status"`
	ActiveNodeID  string        `json:"active_node_id,omitempty"`
	ExternalIP    string        `json:"external_ip,omitempty"`
	LatencyMs     int           `json:"latency_ms"`
	NodesHealth   []ProxyHealth `json:"nodes_health"`
	LastError     string        `json:"last_error,omitempty"`
	LastCheckedAt int64         `json:"last_checked_at"`
}

func DefaultProxySettings() *ProxySettings {
	return &ProxySettings{
		Enabled:             false,
		Mode:                "single",
		Nodes:               []ProxyNode{},
		RotationEnabled:     false,
		RotationInterval:    300,
		HealthCheckEnabled:  true,
		HealthCheckInterval: 60,
		NotifyOnFailure:     true,
		NotifyOnRecover:     true,
	}
}

func (s *ProxySettings) Validate() error {
	// Validate nodes if present
	for _, node := range s.Nodes {
		if !node.Enabled {
			continue
		}
		if node.Host == "" {
			return errors.Validation("Proxy host is required")
		}
		if node.Port < 1 || node.Port > 65535 {
			return errors.Validation("Proxy port must be between 1 and 65535")
		}
		if node.Type != ProxyTypeHTTP && node.Type != ProxyTypeSOCKS5 && node.Type != ProxyTypeTor {
			return errors.Validation("Invalid proxy type")
		}
	}

	if s.RotationInterval < 10 {
		return errors.Validation("Rotation interval must be at least 10 seconds")
	}

	if s.HealthCheckInterval < 10 {
		return errors.Validation("Health check interval must be at least 10 seconds")
	}

	return nil
}

func (s *ProxySettings) GetEnabledNodes() []ProxyNode {
	var enabled []ProxyNode
	for _, node := range s.Nodes {
		if node.Enabled {
			enabled = append(enabled, node)
		}
	}
	return enabled
}

func (s *ProxySettings) GetNodeByID(id string) *ProxyNode {
	for i := range s.Nodes {
		if s.Nodes[i].ID == id {
			return &s.Nodes[i]
		}
	}
	return nil
}

func DefaultTorNode() ProxyNode {
	return ProxyNode{
		ID:      "tor",
		Type:    ProxyTypeTor,
		Host:    "127.0.0.1",
		Port:    9050,
		Enabled: true,
		Order:   0,
	}
}

func TorBrowserNode() ProxyNode {
	return ProxyNode{
		ID:      "tor",
		Type:    ProxyTypeTor,
		Host:    "127.0.0.1",
		Port:    9150,
		Enabled: true,
		Order:   0,
	}
}

type HealthCheckSettings struct {
	Enabled            bool `json:"enabled"`
	IntervalMinutes    int  `json:"interval_minutes"`
	MinIntervalMinutes int  `json:"min_interval_minutes"`
	NotifyWhenHidden   bool `json:"notify_when_hidden"`
	NotifyAlways       bool `json:"notify_always"`
	NotifyWithSound    bool `json:"notify_with_sound"`
	NotifyOnRecover    bool `json:"notify_on_recover"`
}

func DefaultHealthCheckSettings() *HealthCheckSettings {
	return &HealthCheckSettings{
		Enabled:            false,
		IntervalMinutes:    15,
		MinIntervalMinutes: 1,
		NotifyWhenHidden:   true,
		NotifyAlways:       false,
		NotifyWithSound:    true,
		NotifyOnRecover:    true,
	}
}

func (s *HealthCheckSettings) Validate() error {
	if s.IntervalMinutes < s.MinIntervalMinutes {
		return errors.Validation("Interval cannot be less than minimum interval")
	}
	return nil
}

type HealthCheckHistory struct {
	ID             int64
	SiteID         int64
	CheckedAt      time.Time
	Status         HealthStatus
	ResponseTimeMs int
	StatusCode     int
	ErrorMessage   string
}

type AppError struct {
	Code    string
	Message string
}

func (e *AppError) Error() string {
	return e.Message
}
