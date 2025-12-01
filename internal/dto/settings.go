package dto

import "github.com/davidmovas/postulator/internal/domain/entities"

type HealthCheckSettings struct {
	Enabled            bool `json:"enabled"`
	IntervalMinutes    int  `json:"interval_minutes"`
	MinIntervalMinutes int  `json:"min_interval_minutes"`
	NotifyWhenHidden   bool `json:"notify_when_hidden"`
	NotifyAlways       bool `json:"notify_always"`
	NotifyWithSound    bool `json:"notify_with_sound"`
	NotifyOnRecover    bool `json:"notify_on_recover"`
}

func NewHealthCheckSettings(s *entities.HealthCheckSettings) *HealthCheckSettings {
	h := &HealthCheckSettings{}
	return h.FromEntity(s)
}

func (s *HealthCheckSettings) ToEntity() (*entities.HealthCheckSettings, error) {
	return &entities.HealthCheckSettings{
		Enabled:            s.Enabled,
		IntervalMinutes:    s.IntervalMinutes,
		MinIntervalMinutes: s.MinIntervalMinutes,
		NotifyWhenHidden:   s.NotifyWhenHidden,
		NotifyAlways:       s.NotifyAlways,
		NotifyWithSound:    s.NotifyWithSound,
		NotifyOnRecover:    s.NotifyOnRecover,
	}, nil
}

func (s *HealthCheckSettings) FromEntity(e *entities.HealthCheckSettings) *HealthCheckSettings {
	s.Enabled = e.Enabled
	s.IntervalMinutes = e.IntervalMinutes
	s.MinIntervalMinutes = e.MinIntervalMinutes
	s.NotifyWhenHidden = e.NotifyWhenHidden
	s.NotifyAlways = e.NotifyAlways
	s.NotifyWithSound = e.NotifyWithSound
	s.NotifyOnRecover = e.NotifyOnRecover
	return s
}

type ProxyNode struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	Enabled  bool   `json:"enabled"`
	Order    int    `json:"order"`
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
	NodeID      string `json:"node_id"`
	Status      string `json:"status"`
	LatencyMs   int    `json:"latency_ms"`
	LastChecked int64  `json:"last_checked"`
	Error       string `json:"error,omitempty"`
	ExternalIP  string `json:"external_ip,omitempty"`
}

type ProxyState struct {
	Status        string        `json:"status"`
	ActiveNodeID  string        `json:"active_node_id,omitempty"`
	ExternalIP    string        `json:"external_ip,omitempty"`
	LatencyMs     int           `json:"latency_ms"`
	NodesHealth   []ProxyHealth `json:"nodes_health"`
	LastError     string        `json:"last_error,omitempty"`
	LastCheckedAt int64         `json:"last_checked_at"`
}

type TorDetectionResult struct {
	Found       bool   `json:"found"`
	Port        int    `json:"port"`
	ServiceType string `json:"service_type"`
}

func NewProxySettings(e *entities.ProxySettings) *ProxySettings {
	p := &ProxySettings{}
	return p.FromEntity(e)
}

func (p *ProxySettings) ToEntity() (*entities.ProxySettings, error) {
	nodes := make([]entities.ProxyNode, len(p.Nodes))
	for i, n := range p.Nodes {
		nodes[i] = entities.ProxyNode{
			ID:       n.ID,
			Type:     entities.ProxyType(n.Type),
			Host:     n.Host,
			Port:     n.Port,
			Username: n.Username,
			Password: n.Password,
			Enabled:  n.Enabled,
			Order:    n.Order,
		}
	}

	return &entities.ProxySettings{
		Enabled:             p.Enabled,
		Mode:                p.Mode,
		Nodes:               nodes,
		RotationEnabled:     p.RotationEnabled,
		RotationInterval:    p.RotationInterval,
		HealthCheckEnabled:  p.HealthCheckEnabled,
		HealthCheckInterval: p.HealthCheckInterval,
		NotifyOnFailure:     p.NotifyOnFailure,
		NotifyOnRecover:     p.NotifyOnRecover,
		CurrentNodeID:       p.CurrentNodeID,
	}, nil
}

func (p *ProxySettings) FromEntity(e *entities.ProxySettings) *ProxySettings {
	nodes := make([]ProxyNode, len(e.Nodes))
	for i, n := range e.Nodes {
		nodes[i] = ProxyNode{
			ID:       n.ID,
			Type:     string(n.Type),
			Host:     n.Host,
			Port:     n.Port,
			Username: n.Username,
			Password: n.Password,
			Enabled:  n.Enabled,
			Order:    n.Order,
		}
	}

	p.Enabled = e.Enabled
	p.Mode = e.Mode
	p.Nodes = nodes
	p.RotationEnabled = e.RotationEnabled
	p.RotationInterval = e.RotationInterval
	p.HealthCheckEnabled = e.HealthCheckEnabled
	p.HealthCheckInterval = e.HealthCheckInterval
	p.NotifyOnFailure = e.NotifyOnFailure
	p.NotifyOnRecover = e.NotifyOnRecover
	p.CurrentNodeID = e.CurrentNodeID
	return p
}

func NewProxyState(e *entities.ProxyState) *ProxyState {
	s := &ProxyState{}
	return s.FromEntity(e)
}

func (s *ProxyState) FromEntity(e *entities.ProxyState) *ProxyState {
	health := make([]ProxyHealth, len(e.NodesHealth))
	for i, h := range e.NodesHealth {
		health[i] = ProxyHealth{
			NodeID:      h.NodeID,
			Status:      string(h.Status),
			LatencyMs:   h.LatencyMs,
			LastChecked: h.LastChecked,
			Error:       h.Error,
			ExternalIP:  h.ExternalIP,
		}
	}

	s.Status = string(e.Status)
	s.ActiveNodeID = e.ActiveNodeID
	s.ExternalIP = e.ExternalIP
	s.LatencyMs = e.LatencyMs
	s.NodesHealth = health
	s.LastError = e.LastError
	s.LastCheckedAt = e.LastCheckedAt
	return s
}

func NewProxyNode(e *entities.ProxyNode) *ProxyNode {
	return &ProxyNode{
		ID:       e.ID,
		Type:     string(e.Type),
		Host:     e.Host,
		Port:     e.Port,
		Username: e.Username,
		Password: e.Password,
		Enabled:  e.Enabled,
		Order:    e.Order,
	}
}

func (n *ProxyNode) ToEntity() *entities.ProxyNode {
	return &entities.ProxyNode{
		ID:       n.ID,
		Type:     entities.ProxyType(n.Type),
		Host:     n.Host,
		Port:     n.Port,
		Username: n.Username,
		Password: n.Password,
		Enabled:  n.Enabled,
		Order:    n.Order,
	}
}

type IPComparison struct {
	DirectIP    string `json:"direct_ip"`
	DirectError string `json:"direct_error,omitempty"`
	ProxyIP     string `json:"proxy_ip"`
	ProxyError  string `json:"proxy_error,omitempty"`
	IsAnonymous bool   `json:"is_anonymous"`
}
