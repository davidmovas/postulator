package wails

import (
	"context"
	"encoding/json"
	"strings"

	"go.uber.org/zap"

	"github.com/davidmovas/postulator/internal/application/models"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/middleware"
	"github.com/davidmovas/postulator/internal/kernel/settings"
)

type settingsDeclarations interface {
	Schema() ([]settings.Descriptor, error)
	Validate(key string, value json.RawMessage) error
	Apply(values *settings.Values, stored map[string]json.RawMessage) ([]string, error)
}

type settingsStore interface {
	Get(ctx context.Context, key string) (json.RawMessage, bool, error)
	Set(ctx context.Context, key string, value json.RawMessage) error
	All(ctx context.Context) (map[string]json.RawMessage, error)
}

type providerKeyWriter interface {
	SetProviderKey(ctx context.Context, req models.SetProviderKeyRequest) (models.SetProviderKeyResponse, error)
}

type SettingsSchemaRequest struct{}

type SettingsSchemaResponse struct {
	Settings []settings.Descriptor `json:"settings"`
}

type GetSettingRequest struct {
	Key string `json:"key"`
}

type GetSettingResponse struct {
	Key       string          `json:"key"`
	Value     json.RawMessage `json:"value"`
	IsDefault bool            `json:"isDefault"`
}

type SetSettingRequest struct {
	Key   string          `json:"key"`
	Value json.RawMessage `json:"value"`
}

type SetSettingResponse struct {
	Key   string          `json:"key"`
	Value json.RawMessage `json:"value"`
}

type SettingsDeps struct {
	Declarations settingsDeclarations
	Values       *settings.Values
	Store        settingsStore
	Models       providerKeyWriter
}

type SettingsService struct {
	schema         middleware.Handler[SettingsSchemaRequest, SettingsSchemaResponse]
	get            middleware.Handler[GetSettingRequest, GetSettingResponse]
	set            middleware.Handler[SetSettingRequest, SetSettingResponse]
	setProviderKey middleware.Handler[models.SetProviderKeyRequest, models.SetProviderKeyResponse]
}

func NewSettingsService(logger *zap.Logger, deps SettingsDeps) *SettingsService {
	return &SettingsService{
		schema:         Wrap(logger, "settings.schema", describeSettings(deps)),
		get:            Wrap(logger, "settings.get", readSetting(deps)),
		set:            Wrap(logger, "settings.set", writeSetting(deps)),
		setProviderKey: Wrap(logger, "settings.setProviderKey", deps.Models.SetProviderKey),
	}
}

func describeSettings(deps SettingsDeps) middleware.Handler[SettingsSchemaRequest, SettingsSchemaResponse] {
	return func(_ context.Context, _ SettingsSchemaRequest) (SettingsSchemaResponse, error) {
		declared, err := deps.Declarations.Schema()
		if err != nil {
			return SettingsSchemaResponse{}, err
		}
		return SettingsSchemaResponse{Settings: declared}, nil
	}
}

func readSetting(deps SettingsDeps) middleware.Handler[GetSettingRequest, GetSettingResponse] {
	return func(c context.Context, req GetSettingRequest) (GetSettingResponse, error) {
		key := strings.TrimSpace(req.Key)
		declared, err := declaration(deps, key)
		if err != nil {
			return GetSettingResponse{}, err
		}

		stored, found, err := deps.Store.Get(c, key)
		if err != nil {
			return GetSettingResponse{}, err
		}
		if !found {
			return GetSettingResponse{Key: key, Value: declared.Default, IsDefault: true}, nil
		}
		return GetSettingResponse{Key: key, Value: stored}, nil
	}
}

func writeSetting(deps SettingsDeps) middleware.Handler[SetSettingRequest, SetSettingResponse] {
	return func(c context.Context, req SetSettingRequest) (SetSettingResponse, error) {
		key := strings.TrimSpace(req.Key)
		if err := deps.Declarations.Validate(key, req.Value); err != nil {
			return SetSettingResponse{}, err
		}
		if err := deps.Store.Set(c, key, req.Value); err != nil {
			return SetSettingResponse{}, err
		}

		stored, err := deps.Store.All(c)
		if err != nil {
			return SetSettingResponse{}, err
		}
		if _, err = deps.Declarations.Apply(deps.Values, stored); err != nil {
			return SetSettingResponse{}, err
		}
		return SetSettingResponse{Key: key, Value: req.Value}, nil
	}
}

func declaration(deps SettingsDeps, key string) (settings.Descriptor, error) {
	declared, err := deps.Declarations.Schema()
	if err != nil {
		return settings.Descriptor{}, err
	}
	for index := range declared {
		if declared[index].Key == key {
			return declared[index], nil
		}
	}
	return settings.Descriptor{}, errors.New(errors.NotFound, "setting "+key+" is not declared").WithDetail("key", key)
}

func (s *SettingsService) Schema(c context.Context, req SettingsSchemaRequest) (SettingsSchemaResponse, error) {
	return s.schema(c, req)
}

func (s *SettingsService) Get(c context.Context, req GetSettingRequest) (GetSettingResponse, error) {
	return s.get(c, req)
}

func (s *SettingsService) Set(c context.Context, req SetSettingRequest) (SetSettingResponse, error) {
	return s.set(c, req)
}

func (s *SettingsService) SetProviderKey(c context.Context, req models.SetProviderKeyRequest) (models.SetProviderKeyResponse, error) {
	return s.setProviderKey(c, req)
}
