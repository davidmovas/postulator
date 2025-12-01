package handlers

import (
	"github.com/davidmovas/postulator/internal/domain/entities"
	"github.com/davidmovas/postulator/internal/domain/proxy"
	"github.com/davidmovas/postulator/internal/dto"
	"github.com/davidmovas/postulator/internal/infra/wp"
	"github.com/davidmovas/postulator/pkg/ctx"
)

type ProxyHandler struct {
	service  proxy.Service
	wpClient wp.Client
}

func NewProxyHandler(service proxy.Service, wpClient wp.Client) *ProxyHandler {
	return &ProxyHandler{
		service:  service,
		wpClient: wpClient,
	}
}

func (h *ProxyHandler) GetSettings() *dto.Response[*dto.ProxySettings] {
	settings, err := h.service.GetSettings(ctx.FastCtx())
	if err != nil {
		return fail[*dto.ProxySettings](err)
	}
	return ok(dto.NewProxySettings(settings))
}

func (h *ProxyHandler) UpdateSettings(settings *dto.ProxySettings) *dto.Response[string] {
	entity, err := settings.ToEntity()
	if err != nil {
		return fail[string](err)
	}

	if err = h.service.UpdateSettings(ctx.FastCtx(), entity); err != nil {
		return fail[string](err)
	}

	h.service.ApplyToClient(h.wpClient)

	return ok("Proxy settings updated successfully")
}

func (h *ProxyHandler) GetState() *dto.Response[*dto.ProxyState] {
	state := h.service.GetState(ctx.FastCtx())
	return ok(dto.NewProxyState(state))
}

func (h *ProxyHandler) TestNode(node *dto.ProxyNode) *dto.Response[*dto.ProxyHealth] {
	entityNode := node.ToEntity()
	health := h.service.TestNode(ctx.FastCtx(), entityNode)

	return ok(&dto.ProxyHealth{
		NodeID:      health.NodeID,
		Status:      string(health.Status),
		LatencyMs:   health.LatencyMs,
		LastChecked: health.LastChecked,
		Error:       health.Error,
		ExternalIP:  health.ExternalIP,
	})
}

func (h *ProxyHandler) TestAllNodes() *dto.Response[[]dto.ProxyHealth] {
	results := h.service.TestAllNodes(ctx.FastCtx())

	healthList := make([]dto.ProxyHealth, len(results))
	for i, r := range results {
		healthList[i] = dto.ProxyHealth{
			NodeID:      r.NodeID,
			Status:      string(r.Status),
			LatencyMs:   r.LatencyMs,
			LastChecked: r.LastChecked,
			Error:       r.Error,
			ExternalIP:  r.ExternalIP,
		}
	}

	return ok(healthList)
}

func (h *ProxyHandler) DetectTor() *dto.Response[*dto.TorDetectionResult] {
	result := h.service.DetectTor(ctx.FastCtx())
	return ok(&dto.TorDetectionResult{
		Found:       result.Found,
		Port:        result.Port,
		ServiceType: result.ServiceType,
	})
}

func (h *ProxyHandler) Enable() *dto.Response[string] {
	if err := h.service.Enable(ctx.FastCtx()); err != nil {
		return fail[string](err)
	}

	h.service.ApplyToClient(h.wpClient)

	return ok("Proxy enabled")
}

func (h *ProxyHandler) Disable() *dto.Response[string] {
	if err := h.service.Disable(ctx.FastCtx()); err != nil {
		return fail[string](err)
	}

	h.wpClient.DisableProxy()

	return ok("Proxy disabled")
}

func (h *ProxyHandler) AddTorNode() *dto.Response[*dto.ProxyNode] {
	torResult := h.service.DetectTor(ctx.FastCtx())

	var node entities.ProxyNode
	if torResult.Found {
		if torResult.Port == 9150 {
			node = entities.TorBrowserNode()
		} else {
			node = entities.DefaultTorNode()
		}
	} else {
		node = entities.DefaultTorNode()
	}

	return ok(dto.NewProxyNode(&node))
}

func (h *ProxyHandler) GetDefaultTorNode() *dto.Response[*dto.ProxyNode] {
	node := entities.DefaultTorNode()
	return ok(dto.NewProxyNode(&node))
}

func (h *ProxyHandler) GetTorBrowserNode() *dto.Response[*dto.ProxyNode] {
	node := entities.TorBrowserNode()
	return ok(dto.NewProxyNode(&node))
}

func (h *ProxyHandler) CompareIPs() *dto.Response[*dto.IPComparison] {
	result := h.service.CompareIPs()
	return ok(&dto.IPComparison{
		DirectIP:    result.DirectIP,
		DirectError: result.DirectError,
		ProxyIP:     result.ProxyIP,
		ProxyError:  result.ProxyError,
		IsAnonymous: result.IsAnonymous,
	})
}
