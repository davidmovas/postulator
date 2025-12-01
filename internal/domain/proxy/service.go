package proxy

import (
	"context"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/davidmovas/postulator/internal/domain/entities"
	"github.com/davidmovas/postulator/internal/domain/settings"
	"github.com/davidmovas/postulator/internal/infra/events"
	"github.com/davidmovas/postulator/pkg/logger"
)

var _ Service = (*service)(nil)

type service struct {
	settingsService settings.Service
	eventBus        *events.EventBus
	logger          *logger.Logger
	detector        *TorDetector
	checker         *NodeChecker

	mu            sync.RWMutex
	state         *entities.ProxyState
	currentNodes  []entities.ProxyNode
	transport     http.RoundTripper
	rotationIndex int

	healthTicker   *time.Ticker
	rotationTicker *time.Ticker
	stopCh         chan struct{}
	running        bool
}

func NewService(
	settingsService settings.Service,
	eventBus *events.EventBus,
	logger *logger.Logger,
) Service {
	return &service{
		settingsService: settingsService,
		eventBus:        eventBus,
		logger:          logger.WithScope("service").WithScope("proxy"),
		detector:        NewTorDetector(),
		checker:         NewNodeChecker(),
		state: &entities.ProxyState{
			Status:      entities.ProxyStatusDisabled,
			NodesHealth: []entities.ProxyHealth{},
		},
		stopCh: make(chan struct{}),
	}
}

func (s *service) GetSettings(ctx context.Context) (*entities.ProxySettings, error) {
	return s.settingsService.GetProxySettings(ctx)
}

func (s *service) UpdateSettings(ctx context.Context, settings *entities.ProxySettings) error {
	if err := s.settingsService.UpdateProxySettings(ctx, settings); err != nil {
		return err
	}

	s.eventBus.Publish(ctx, events.Event{
		Type:      events.ProxySettingsUpdated,
		Timestamp: time.Now(),
		Data:      settings,
	})

	if settings.Enabled {
		return s.applySettings(ctx, settings)
	}

	return s.Disable(ctx)
}

func (s *service) GetState(ctx context.Context) *entities.ProxyState {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stateCopy := *s.state
	stateCopy.NodesHealth = make([]entities.ProxyHealth, len(s.state.NodesHealth))
	copy(stateCopy.NodesHealth, s.state.NodesHealth)

	return &stateCopy
}

func (s *service) TestNode(_ context.Context, node *entities.ProxyNode) *entities.ProxyHealth {
	return s.checker.CheckNode(node)
}

func (s *service) TestAllNodes(ctx context.Context) []entities.ProxyHealth {
	settings, err := s.GetSettings(ctx)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get proxy settings for testing")
		return nil
	}

	return s.checker.CheckNodes(settings.Nodes)
}

func (s *service) DetectTor(ctx context.Context) *TorDetectionResult {
	return s.detector.Detect(ctx)
}

func (s *service) Enable(ctx context.Context) error {
	settings, err := s.GetSettings(ctx)
	if err != nil {
		return err
	}

	settings.Enabled = true
	return s.UpdateSettings(ctx, settings)
}

func (s *service) Disable(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.stopBackgroundTasks()

	s.state = &entities.ProxyState{
		Status:        entities.ProxyStatusDisabled,
		NodesHealth:   []entities.ProxyHealth{},
		LastCheckedAt: time.Now().Unix(),
	}
	s.transport = nil
	s.currentNodes = nil

	s.eventBus.Publish(ctx, events.Event{
		Type:      events.ProxyStateChanged,
		Timestamp: time.Now(),
		Data:      s.state,
	})

	s.logger.Info("Proxy disabled")
	return nil
}

func (s *service) GetHTTPTransport() http.RoundTripper {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.transport == nil {
		return http.DefaultTransport
	}
	return s.transport
}

func (s *service) ApplyToClient(client HTTPClientWithProxy) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.state.Status != entities.ProxyStatusConnected || len(s.currentNodes) == 0 {
		client.DisableProxy()
		return nil
	}

	node := s.currentNodes[0]
	if s.state.ActiveNodeID != "" {
		for _, n := range s.currentNodes {
			if n.ID == s.state.ActiveNodeID {
				node = n
				break
			}
		}
	}

	proxyURL := BuildProxyURLForNode(node)
	client.EnableProxy(proxyURL)

	return nil
}

func (s *service) applySettings(ctx context.Context, settings *entities.ProxySettings) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.stopBackgroundTasks()

	enabledNodes := settings.GetEnabledNodes()
	if len(enabledNodes) == 0 {
		// No nodes configured yet, stay disabled
		s.state = &entities.ProxyState{
			Status:        entities.ProxyStatusDisabled,
			NodesHealth:   []entities.ProxyHealth{},
			LastCheckedAt: time.Now().Unix(),
		}
		s.transport = nil
		s.currentNodes = nil
		return nil
	}

	sort.Slice(enabledNodes, func(i, j int) bool {
		return enabledNodes[i].Order < enabledNodes[j].Order
	})

	s.currentNodes = enabledNodes
	s.state.Status = entities.ProxyStatusConnecting

	if settings.Mode == "chain" {
		transport, err := NewChainTransport(enabledNodes)
		if err != nil {
			s.state.Status = entities.ProxyStatusError
			s.state.LastError = err.Error()
			return err
		}
		s.transport = transport
	} else {
		node := enabledNodes[0]
		transport, err := buildSingleNodeTransport(node)
		if err != nil {
			s.state.Status = entities.ProxyStatusError
			s.state.LastError = err.Error()
			return err
		}
		s.transport = transport
		s.state.ActiveNodeID = node.ID
	}

	s.running = true
	s.stopCh = make(chan struct{})

	go s.initialHealthCheck(ctx, settings)

	if settings.HealthCheckEnabled {
		s.startHealthChecker(ctx, settings)
	}

	if settings.RotationEnabled && settings.Mode == "single" && len(enabledNodes) > 1 {
		s.startRotation(ctx, settings)
	}

	s.logger.Info("Proxy enabled")
	return nil
}

func (s *service) initialHealthCheck(ctx context.Context, settings *entities.ProxySettings) {
	health := s.checker.CheckNodes(s.currentNodes)

	s.mu.Lock()
	s.state.NodesHealth = health
	s.state.LastCheckedAt = time.Now().Unix()

	hasHealthy := false
	for _, h := range health {
		if h.Status == entities.ProxyStatusConnected {
			hasHealthy = true
			if s.state.ActiveNodeID == "" || s.state.ActiveNodeID == h.NodeID {
				s.state.ExternalIP = h.ExternalIP
				s.state.LatencyMs = h.LatencyMs
			}
		}
	}

	if hasHealthy {
		s.state.Status = entities.ProxyStatusConnected
		s.state.LastError = ""
	} else {
		s.state.Status = entities.ProxyStatusError
		if len(health) > 0 {
			s.state.LastError = health[0].Error
		}
	}
	s.mu.Unlock()

	s.eventBus.Publish(ctx, events.Event{
		Type:      events.ProxyStateChanged,
		Timestamp: time.Now(),
		Data:      s.GetState(ctx),
	})
}

func (s *service) startHealthChecker(ctx context.Context, settings *entities.ProxySettings) {
	interval := time.Duration(settings.HealthCheckInterval) * time.Second
	s.healthTicker = time.NewTicker(interval)

	go func() {
		for {
			select {
			case <-s.stopCh:
				return
			case <-s.healthTicker.C:
				s.runHealthCheck(ctx, settings)
			}
		}
	}()
}

func (s *service) runHealthCheck(ctx context.Context, settings *entities.ProxySettings) {
	s.mu.RLock()
	nodes := s.currentNodes
	previousState := s.state.Status
	s.mu.RUnlock()

	health := s.checker.CheckNodes(nodes)

	s.mu.Lock()
	s.state.NodesHealth = health
	s.state.LastCheckedAt = time.Now().Unix()

	hasHealthy := false
	for _, h := range health {
		if h.Status == entities.ProxyStatusConnected {
			hasHealthy = true
			if h.NodeID == s.state.ActiveNodeID {
				s.state.ExternalIP = h.ExternalIP
				s.state.LatencyMs = h.LatencyMs
			}
		}
	}

	if hasHealthy {
		s.state.Status = entities.ProxyStatusConnected
		if previousState == entities.ProxyStatusError && settings.NotifyOnRecover {
			go s.notifyRecovery(ctx)
		}
	} else {
		s.state.Status = entities.ProxyStatusError
		if previousState == entities.ProxyStatusConnected && settings.NotifyOnFailure {
			go s.notifyFailure(ctx, s.state.LastError)
		}
	}
	s.mu.Unlock()

	s.eventBus.Publish(ctx, events.Event{
		Type:      events.ProxyStateChanged,
		Timestamp: time.Now(),
		Data:      s.GetState(ctx),
	})
}

func (s *service) startRotation(ctx context.Context, settings *entities.ProxySettings) {
	interval := time.Duration(settings.RotationInterval) * time.Second
	s.rotationTicker = time.NewTicker(interval)

	go func() {
		for {
			select {
			case <-s.stopCh:
				return
			case <-s.rotationTicker.C:
				s.rotate(ctx)
			}
		}
	}()
}

func (s *service) rotate(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.currentNodes) < 2 {
		return
	}

	s.rotationIndex = (s.rotationIndex + 1) % len(s.currentNodes)
	node := s.currentNodes[s.rotationIndex]

	transport, err := buildSingleNodeTransport(node)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to rotate to node")
		return
	}

	s.transport = transport
	s.state.ActiveNodeID = node.ID

	s.logger.Infof("Rotated to proxy node: %s", node.ID)

	s.eventBus.Publish(ctx, events.Event{
		Type:      events.ProxyNodeRotated,
		Timestamp: time.Now(),
		Data:      node,
	})
}

func (s *service) stopBackgroundTasks() {
	if !s.running {
		return
	}

	close(s.stopCh)

	if s.healthTicker != nil {
		s.healthTicker.Stop()
		s.healthTicker = nil
	}

	if s.rotationTicker != nil {
		s.rotationTicker.Stop()
		s.rotationTicker = nil
	}

	s.running = false
}

func (s *service) notifyRecovery(ctx context.Context) {
	s.eventBus.Publish(ctx, events.Event{
		Type:      events.ProxyRecovered,
		Timestamp: time.Now(),
	})
}

func (s *service) notifyFailure(ctx context.Context, errMsg string) {
	s.eventBus.Publish(ctx, events.Event{
		Type:      events.ProxyFailed,
		Timestamp: time.Now(),
		Data:      errMsg,
	})
}

func (s *service) Initialize(ctx context.Context) error {
	settings, err := s.GetSettings(ctx)
	if err != nil {
		return err
	}

	if settings.Enabled {
		return s.applySettings(ctx, settings)
	}

	return nil
}

func (s *service) CompareIPs() *IPComparison {
	result := &IPComparison{}

	// Get direct IP (without proxy)
	directIP, err := s.checker.GetDirectIP()
	if err != nil {
		result.DirectError = err.Error()
	} else {
		result.DirectIP = directIP
	}

	// Get proxy IP (if proxy is active)
	s.mu.RLock()
	transport := s.transport
	status := s.state.Status
	s.mu.RUnlock()

	if status != entities.ProxyStatusConnected || transport == nil {
		result.ProxyError = "Proxy is not connected"
	} else {
		proxyIP, err := s.checker.GetProxyIP(transport)
		if err != nil {
			result.ProxyError = err.Error()
		} else {
			result.ProxyIP = proxyIP
		}
	}

	// Check if anonymous (IPs are different)
	if result.DirectIP != "" && result.ProxyIP != "" {
		result.IsAnonymous = result.DirectIP != result.ProxyIP
	}

	return result
}
