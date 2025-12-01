package proxy

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/davidmovas/postulator/internal/domain/entities"
	"github.com/davidmovas/postulator/pkg/ctx"
)

const (
	ipCheckURL   = "https://api.ipify.org?format=json"
	checkTimeout = 30 * time.Second
)

type IPResponse struct {
	IP string `json:"ip"`
}

type NodeChecker struct {
	timeout time.Duration
}

func NewNodeChecker() *NodeChecker {
	return &NodeChecker{
		timeout: checkTimeout,
	}
}

func (c *NodeChecker) CheckNode(node *entities.ProxyNode) *entities.ProxyHealth {
	startTime := time.Now()

	health := &entities.ProxyHealth{
		NodeID:      node.ID,
		Status:      entities.ProxyStatusConnecting,
		LastChecked: time.Now().Unix(),
	}

	transport, err := buildSingleNodeTransport(*node)
	if err != nil {
		health.Status = entities.ProxyStatusError
		health.Error = fmt.Sprintf("Failed to create transport: %v", err)
		return health
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   c.timeout,
	}

	// Use dedicated context for this request to avoid cancellation from parent
	reqCtx := ctx.WithTimeout(checkTimeout)
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, ipCheckURL, nil)
	if err != nil {
		health.Status = entities.ProxyStatusError
		health.Error = fmt.Sprintf("Failed to create request: %v", err)
		return health
	}

	resp, err := client.Do(req)
	if err != nil {
		health.Status = entities.ProxyStatusError
		health.Error = fmt.Sprintf("Connection failed: %v", err)
		return health
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		health.Status = entities.ProxyStatusError
		health.Error = fmt.Sprintf("Failed to read response: %v", err)
		return health
	}

	var ipResp IPResponse
	if err := json.Unmarshal(body, &ipResp); err != nil {
		health.Status = entities.ProxyStatusError
		health.Error = fmt.Sprintf("Failed to parse response: %v", err)
		return health
	}

	health.Status = entities.ProxyStatusConnected
	health.ExternalIP = ipResp.IP
	health.LatencyMs = int(time.Since(startTime).Milliseconds())

	return health
}

func (c *NodeChecker) CheckNodes(nodes []entities.ProxyNode) []entities.ProxyHealth {
	var enabledNodes []entities.ProxyNode
	for _, node := range nodes {
		if node.Enabled {
			enabledNodes = append(enabledNodes, node)
		}
	}

	if len(enabledNodes) == 0 {
		return nil
	}

	results := make([]entities.ProxyHealth, len(enabledNodes))
	var wg sync.WaitGroup

	for i, node := range enabledNodes {
		wg.Add(1)
		go func(idx int, n entities.ProxyNode) {
			defer wg.Done()
			health := c.CheckNode(&n)
			results[idx] = *health
		}(i, node)
	}

	wg.Wait()
	return results
}

func (c *NodeChecker) QuickCheck(node *entities.ProxyNode) bool {
	transport, err := buildSingleNodeTransport(*node)
	if err != nil {
		return false
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   5 * time.Second,
	}

	reqCtx := ctx.FastCtx()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodHead, "https://www.google.com", nil)
	if err != nil {
		return false
	}

	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()

	return resp.StatusCode < 500
}

func (c *NodeChecker) GetDirectIP() (string, error) {
	client := &http.Client{
		Timeout: checkTimeout,
	}

	reqCtx := ctx.WithTimeout(checkTimeout)
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, ipCheckURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var ipResp IPResponse
	if err := json.Unmarshal(body, &ipResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	return ipResp.IP, nil
}

func (c *NodeChecker) GetProxyIP(transport http.RoundTripper) (string, error) {
	client := &http.Client{
		Transport: transport,
		Timeout:   checkTimeout,
	}

	reqCtx := ctx.WithTimeout(checkTimeout)
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, ipCheckURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var ipResp IPResponse
	if err := json.Unmarshal(body, &ipResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	return ipResp.IP, nil
}

func (c *NodeChecker) CheckChain(nodes []entities.ProxyNode) *entities.ProxyHealth {
	startTime := time.Now()

	health := &entities.ProxyHealth{
		NodeID:      "chain",
		Status:      entities.ProxyStatusConnecting,
		LastChecked: time.Now().Unix(),
	}

	if len(nodes) == 0 {
		health.Status = entities.ProxyStatusError
		health.Error = "No nodes in chain"
		return health
	}

	transport, err := buildMultiNodeTransport(nodes)
	if err != nil {
		health.Status = entities.ProxyStatusError
		health.Error = fmt.Sprintf("Failed to create chain transport: %v", err)
		return health
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   60 * time.Second,
	}

	reqCtx := ctx.WithTimeout(60 * time.Second)
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, ipCheckURL, nil)
	if err != nil {
		health.Status = entities.ProxyStatusError
		health.Error = fmt.Sprintf("Failed to create request: %v", err)
		return health
	}

	resp, err := client.Do(req)
	if err != nil {
		health.Status = entities.ProxyStatusError
		health.Error = fmt.Sprintf("Chain connection failed: %v", err)
		return health
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		health.Status = entities.ProxyStatusError
		health.Error = fmt.Sprintf("Failed to read response: %v", err)
		return health
	}

	var ipResp IPResponse
	if err := json.Unmarshal(body, &ipResp); err != nil {
		health.Status = entities.ProxyStatusError
		health.Error = fmt.Sprintf("Failed to parse response: %v", err)
		return health
	}

	health.Status = entities.ProxyStatusConnected
	health.ExternalIP = ipResp.IP
	health.LatencyMs = int(time.Since(startTime).Milliseconds())

	return health
}
