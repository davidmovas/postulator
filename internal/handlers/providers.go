package handlers

import (
	"github.com/davidmovas/postulator/internal/domain/entities"
	"github.com/davidmovas/postulator/internal/domain/providers"
	"github.com/davidmovas/postulator/internal/dto"
	"github.com/davidmovas/postulator/pkg/ctx"
)

type ProvidersHandler struct {
	service providers.Service
}

func NewProvidersHandler(service providers.Service) *ProvidersHandler {
	return &ProvidersHandler{
		service: service,
	}
}

func (h *ProvidersHandler) CreateProvider(provider *dto.Provider) *dto.Response[string] {
	entity, err := provider.ToEntity()
	if err != nil {
		return fail[string](err)
	}

	if err = h.service.CreateProvider(ctx.FastCtx(), entity); err != nil {
		return fail[string](err)
	}

	return ok("Provider created successfully")
}

func (h *ProvidersHandler) GetProvider(id int64) *dto.Response[*dto.Provider] {
	provider, err := h.service.GetProvider(ctx.FastCtx(), id)
	if err != nil {
		return fail[*dto.Provider](err)
	}

	return ok(dto.NewProvider(provider))
}

func (h *ProvidersHandler) ListProviders() *dto.Response[[]*dto.Provider] {
	listProviders, err := h.service.ListProviders(ctx.FastCtx())
	if err != nil {
		return fail[[]*dto.Provider](err)
	}

	var dtoProviders []*dto.Provider
	for _, provider := range listProviders {
		dtoProviders = append(dtoProviders, dto.NewProvider(provider))
	}

	return ok(dtoProviders)
}

func (h *ProvidersHandler) ListActiveProviders() *dto.Response[[]*dto.Provider] {
	activeProviders, err := h.service.ListActiveProviders(ctx.FastCtx())
	if err != nil {
		return fail[[]*dto.Provider](err)
	}

	var dtoProviders []*dto.Provider
	for _, provider := range activeProviders {
		dtoProviders = append(dtoProviders, dto.NewProvider(provider))
	}

	return ok(dtoProviders)
}

func (h *ProvidersHandler) UpdateProvider(provider *dto.Provider) *dto.Response[string] {
	entity, err := provider.ToEntity()
	if err != nil {
		return fail[string](err)
	}

	if err = h.service.UpdateProvider(ctx.FastCtx(), entity); err != nil {
		return fail[string](err)
	}

	return ok("Provider updated successfully")
}

func (h *ProvidersHandler) DeleteProvider(id int64) *dto.Response[string] {
	if err := h.service.DeleteProvider(ctx.FastCtx(), id); err != nil {
		return fail[string](err)
	}

	return ok("Provider deleted successfully")
}

func (h *ProvidersHandler) SetProviderStatus(id int64, isActive bool) *dto.Response[string] {
	if err := h.service.SetProviderStatus(ctx.FastCtx(), id, isActive); err != nil {
		return fail[string](err)
	}

	return ok("Provider status updated successfully")
}

func (h *ProvidersHandler) GetAvailableModels(providerType string) *dto.Response[[]*dto.Model] {
	models, err := h.service.GetAvailableModels(entities.Type(providerType))
	if err != nil {
		return fail[[]*dto.Model](err)
	}

	var dtoModels []*dto.Model
	for _, model := range models {
		dtoModels = append(dtoModels, dto.NewModel(model))
	}

	return ok(dtoModels)
}

func (h *ProvidersHandler) ValidateModel(providerType, model string) *dto.Response[string] {
	if err := h.service.ValidateModel(entities.Type(providerType), model); err != nil {
		return fail[string](err)
	}

	return ok("Model is valid")
}
