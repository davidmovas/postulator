package handlers

import (
	"github.com/davidmovas/postulator/internal/domain/settings"
	"github.com/davidmovas/postulator/internal/dto"
	"github.com/davidmovas/postulator/pkg/ctx"
)

type SettingsHandler struct {
	service settings.Service
}

func NewSettingsHandler(service settings.Service) *SettingsHandler {
	return &SettingsHandler{
		service: service,
	}
}

func (h *SettingsHandler) GetHealthCheckSettings() *dto.Response[*dto.HealthCheckSettings] {
	s, err := h.service.GetHealthCheckSettings(ctx.FastCtx())
	if err != nil {
		return fail[*dto.HealthCheckSettings](err)
	}

	return ok(dto.NewHealthCheckSettings(s))
}

func (h *SettingsHandler) UpdateHealthCheckSettings(settings *dto.HealthCheckSettings) *dto.Response[string] {
	entity, err := settings.ToEntity()
	if err != nil {
		return fail[string](err)
	}

	if err = h.service.UpdateHealthCheckSettings(ctx.FastCtx(), entity); err != nil {
		return fail[string](err)
	}

	return ok("Settings updated successfully")
}
