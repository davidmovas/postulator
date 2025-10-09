package app

import (
	"Postulator/internal/domain/entities"
	"Postulator/internal/dto"
	"Postulator/pkg/errors"
	"context"
)

// AI Provider bindings

func (a *App) CreateAIProvider(req *dto.AIProviderCreate) *dto.Response[string] {
	if a.aiProvSvc == nil {
		if err := a.BuildServices(); err != nil {
			return dtoErr[string](errors.Internal(err))
		}
	}
	if req == nil {
		return dtoErr[string](errors.Validation("provider payload is required"))
	}
	entity := &entities.AIProvider{Name: req.Name, APIKey: req.APIKey, Model: req.Model, IsActive: req.IsActive}
	if err := a.aiProvSvc.CreateProvider(context.Background(), entity); err != nil {
		return dtoErr[string](asAppErr(err))
	}
	return &dto.Response[string]{Success: true, Data: "created"}
}

func (a *App) GetAIProvider(id int64) *dto.Response[*dto.AIProvider] {
	if a.aiProvSvc == nil {
		if err := a.BuildServices(); err != nil {
			return dtoErr[*dto.AIProvider](errors.Internal(err))
		}
	}
	p, err := a.aiProvSvc.GetProvider(context.Background(), id)
	if err != nil {
		return dtoErr[*dto.AIProvider](asAppErr(err))
	}
	return &dto.Response[*dto.AIProvider]{Success: true, Data: dto.FromAIProvider(p)}
}

func (a *App) ListAIProviders() *dto.Response[[]*dto.AIProvider] {
	if a.aiProvSvc == nil {
		if err := a.BuildServices(); err != nil {
			return dtoErr[[]*dto.AIProvider](errors.Internal(err))
		}
	}
	items, err := a.aiProvSvc.ListProviders(context.Background())
	if err != nil {
		return dtoErr[[]*dto.AIProvider](asAppErr(err))
	}
	return &dto.Response[[]*dto.AIProvider]{Success: true, Data: dto.FromAIProviders(items)}
}

func (a *App) ListActiveAIProviders() *dto.Response[[]*dto.AIProvider] {
	if a.aiProvSvc == nil {
		if err := a.BuildServices(); err != nil {
			return dtoErr[[]*dto.AIProvider](errors.Internal(err))
		}
	}
	items, err := a.aiProvSvc.ListActiveProviders(context.Background())
	if err != nil {
		return dtoErr[[]*dto.AIProvider](asAppErr(err))
	}
	return &dto.Response[[]*dto.AIProvider]{Success: true, Data: dto.FromAIProviders(items)}
}

func (a *App) UpdateAIProvider(req *dto.AIProviderUpdate) *dto.Response[string] {
	if a.aiProvSvc == nil {
		if err := a.BuildServices(); err != nil {
			return dtoErr[string](errors.Internal(err))
		}
	}
	if req == nil {
		return dtoErr[string](errors.Validation("provider payload is required"))
	}
	entity := &entities.AIProvider{ID: req.ID, Name: req.Name, APIKey: req.APIKey, Model: req.Model, IsActive: req.IsActive}
	if err := a.aiProvSvc.UpdateProvider(context.Background(), entity); err != nil {
		return dtoErr[string](asAppErr(err))
	}
	return &dto.Response[string]{Success: true, Data: "updated"}
}

func (a *App) DeleteAIProvider(id int64) *dto.Response[string] {
	if a.aiProvSvc == nil {
		if err := a.BuildServices(); err != nil {
			return dtoErr[string](errors.Internal(err))
		}
	}
	if err := a.aiProvSvc.DeleteProvider(context.Background(), id); err != nil {
		return dtoErr[string](asAppErr(err))
	}
	return &dto.Response[string]{Success: true, Data: "deleted"}
}

func (a *App) SetAIProviderStatus(id int64, active bool) *dto.Response[string] {
	if a.aiProvSvc == nil {
		if err := a.BuildServices(); err != nil {
			return dtoErr[string](errors.Internal(err))
		}
	}
	if err := a.aiProvSvc.SetProviderStatus(context.Background(), id, active); err != nil {
		return dtoErr[string](asAppErr(err))
	}
	return &dto.Response[string]{Success: true, Data: "updated"}
}

func (a *App) GetAvailableModels(providerName string) *dto.Response[[]string] {
	if a.aiProvSvc == nil {
		if err := a.BuildServices(); err != nil {
			return dtoErr[[]string](errors.Internal(err))
		}
	}
	models := a.aiProvSvc.GetAvailableModels(providerName)
	out := make([]string, 0, len(models))
	for _, m := range models {
		out = append(out, string(m))
	}
	return &dto.Response[[]string]{Success: true, Data: out}
}

func (a *App) ValidateModel(providerName, model string) *dto.Response[string] {
	if a.aiProvSvc == nil {
		if err := a.BuildServices(); err != nil {
			return dtoErr[string](errors.Internal(err))
		}
	}
	if err := a.aiProvSvc.ValidateModel(providerName, model); err != nil {
		return dtoErr[string](asAppErr(err))
	}
	return &dto.Response[string]{Success: true, Data: "valid"}
}
