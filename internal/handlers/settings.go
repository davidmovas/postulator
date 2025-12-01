package handlers

import (
	"github.com/davidmovas/postulator/internal/domain/healthcheck"
	"github.com/davidmovas/postulator/internal/domain/settings"
	"github.com/davidmovas/postulator/internal/dto"
	"github.com/davidmovas/postulator/internal/version"
	"github.com/davidmovas/postulator/pkg/ctx"
)

type SettingsHandler struct {
	service   settings.Service
	scheduler healthcheck.Scheduler
}

func NewSettingsHandler(service settings.Service, scheduler healthcheck.Scheduler) *SettingsHandler {
	return &SettingsHandler{
		service:   service,
		scheduler: scheduler,
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

	_ = h.scheduler.ApplySettings(ctx.FastCtx(), entity.Enabled, entity.IntervalMinutes)

	return ok("Settings updated successfully")
}

func (h *SettingsHandler) GetAppVersion() *dto.Response[*dto.AppVersion] {
	info := version.GetInfo()
	return ok(&dto.AppVersion{
		Version:   info.Version,
		Commit:    info.Commit,
		BuildDate: info.BuildDate,
	})
}
