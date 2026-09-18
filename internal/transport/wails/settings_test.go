package wails_test

import (
	"context"
	"encoding/json"
	"testing"

	"go.uber.org/zap"

	"github.com/davidmovas/postulator/internal/application/models"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/settings"
	"github.com/davidmovas/postulator/internal/transport/wails"
)

type declarationsFake struct {
	registry  *settings.Registry
	describes error
	panics    bool
}

func (d declarationsFake) Schema() ([]settings.Descriptor, error) {
	if d.panics {
		panic("the registry went away")
	}
	if d.describes != nil {
		return nil, d.describes
	}
	return d.registry.Schema()
}

func (d declarationsFake) Validate(key string, value json.RawMessage) error {
	if d.panics {
		panic("the registry went away")
	}
	return d.registry.Validate(key, value)
}

func (d declarationsFake) Apply(values *settings.Values, stored map[string]json.RawMessage) ([]string, error) {
	return d.registry.Apply(values, stored)
}

type storeFake struct {
	stored map[string]json.RawMessage
	fail   error
}

func (s *storeFake) Get(_ context.Context, key string) (value json.RawMessage, found bool, err error) {
	if s.fail != nil {
		return nil, false, s.fail
	}
	value, found = s.stored[key]
	return value, found, nil
}

func (s *storeFake) Set(_ context.Context, key string, value json.RawMessage) error {
	if s.fail != nil {
		return s.fail
	}
	s.stored[key] = value
	return nil
}

func (s *storeFake) All(context.Context) (map[string]json.RawMessage, error) {
	if s.fail != nil {
		return nil, s.fail
	}
	return s.stored, nil
}

type providerKeyFake struct {
	mode    failure
	fails   bool
	written string
}

func (p *providerKeyFake) SetProviderKey(_ context.Context, req models.SetProviderKeyRequest) (models.SetProviderKeyResponse, error) {
	if p.fails {
		return answer[models.SetProviderKeyResponse](p.mode)
	}
	p.written = req.APIKey
	return models.SetProviderKeyResponse{Provider: req.Provider}, nil
}

func settingsHarness(t *testing.T) (deps wails.SettingsDeps, workers *settings.Setting[int], store *storeFake, keys *providerKeyFake) {
	t.Helper()

	registry := settings.New()
	workers = registry.Int("runs.workers", 2, settings.IntRange(1, 16))
	store = &storeFake{stored: map[string]json.RawMessage{}}
	keys = &providerKeyFake{}

	return wails.SettingsDeps{
		Declarations: declarationsFake{registry: registry},
		Values:       registry.NewValues(),
		Store:        store,
		Models:       keys,
	}, workers, store, keys
}

func TestSettingsServiceDescribesEveryDeclaration(t *testing.T) {
	t.Parallel()

	deps, _, _, _ := settingsHarness(t)

	described, err := wails.NewSettingsService(zap.NewNop(), deps).Schema(context.Background(), wails.SettingsSchemaRequest{})
	if err != nil {
		t.Fatalf("Schema: %v", err)
	}
	if len(described.Settings) != 1 || described.Settings[0].Key != "runs.workers" {
		t.Fatalf("Schema() = %+v, want the one declared setting", described.Settings)
	}
	if string(described.Settings[0].Default) != "2" || described.Settings[0].Max != 16 {
		t.Errorf("descriptor = %+v, want the default and the range", described.Settings[0])
	}
}

func TestSettingsServiceReadsTheStoredValueOrTheDefault(t *testing.T) {
	t.Parallel()

	deps, _, store, _ := settingsHarness(t)
	service := wails.NewSettingsService(zap.NewNop(), deps)

	unset, err := service.Get(context.Background(), wails.GetSettingRequest{Key: "runs.workers"})
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if string(unset.Value) != "2" || !unset.IsDefault {
		t.Errorf("Get() = %+v, want the declared default", unset)
	}

	store.stored["runs.workers"] = json.RawMessage("7")
	set, err := service.Get(context.Background(), wails.GetSettingRequest{Key: "runs.workers"})
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if string(set.Value) != "7" || set.IsDefault {
		t.Errorf("Get() = %+v, want the stored value", set)
	}

	if _, err = service.Get(context.Background(), wails.GetSettingRequest{Key: "runs.nope"}); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("Get() of an undeclared key = %v, want NOT_FOUND", err)
	}
}

func TestSettingsServiceWritesAndRefreshesTheLiveValue(t *testing.T) {
	t.Parallel()

	deps, workers, store, _ := settingsHarness(t)
	service := wails.NewSettingsService(zap.NewNop(), deps)

	written, err := service.Set(context.Background(), wails.SetSettingRequest{Key: "runs.workers", Value: json.RawMessage("8")})
	if err != nil {
		t.Fatalf("Set: %v", err)
	}
	if written.Key != "runs.workers" || string(written.Value) != "8" {
		t.Errorf("Set() = %+v, want the key and the value it wrote", written)
	}
	if string(store.stored["runs.workers"]) != "8" {
		t.Errorf("stored = %s, want 8", store.stored["runs.workers"])
	}
	if got := workers.Get(deps.Values); got != 8 {
		t.Errorf("the live value is %d, want the 8 that was just written", got)
	}
}

func TestSettingsServiceRefusesAValueTheDeclarationRejects(t *testing.T) {
	t.Parallel()

	deps, workers, store, _ := settingsHarness(t)
	service := wails.NewSettingsService(zap.NewNop(), deps)

	cases := []struct {
		name string
		req  wails.SetSettingRequest
		want errors.Code
	}{
		{
			name: "out of range",
			req:  wails.SetSettingRequest{Key: "runs.workers", Value: json.RawMessage("99")},
			want: errors.Invalid,
		},
		{
			name: "not declared",
			req:  wails.SetSettingRequest{Key: "runs.nope", Value: json.RawMessage("1")},
			want: errors.NotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := service.Set(context.Background(), tc.req); !errors.IsCode(err, tc.want) {
				t.Fatalf("Set() = %v, want %s", err, tc.want)
			}
			if len(store.stored) != 0 {
				t.Fatalf("stored = %v, want nothing written", store.stored)
			}
			if got := workers.Get(deps.Values); got != 2 {
				t.Fatalf("the live value is %d, want the untouched default", got)
			}
		})
	}
}

func TestSettingsServiceDelegatesTheProviderKey(t *testing.T) {
	t.Parallel()

	deps, _, _, keys := settingsHarness(t)

	written, err := wails.NewSettingsService(zap.NewNop(), deps).SetProviderKey(context.Background(),
		models.SetProviderKeyRequest{Provider: "openai", APIKey: "sk-secret"})
	if err != nil {
		t.Fatalf("SetProviderKey: %v", err)
	}
	if written.Provider != "openai" {
		t.Errorf("Provider = %q, want openai", written.Provider)
	}
	if keys.written != "sk-secret" {
		t.Errorf("the models use case received %q, want the key it was handed", keys.written)
	}

	encoded, marshalErr := json.Marshal(written)
	if marshalErr != nil {
		t.Fatalf("Marshal: %v", marshalErr)
	}
	if string(encoded) != `{"provider":"openai"}` {
		t.Errorf("response = %s, want the provider alone", encoded)
	}
}

func TestSettingsServiceConvertsEveryFailure(t *testing.T) {
	t.Parallel()

	registry := settings.New()
	registry.Int("runs.workers", 2, settings.IntRange(1, 16))

	assertMethodNames(t, wails.NewSettingsService(zap.NewNop(), wails.SettingsDeps{
		Declarations: declarationsFake{registry: registry},
		Values:       registry.NewValues(),
		Store:        &storeFake{stored: map[string]json.RawMessage{}},
		Models:       &providerKeyFake{},
	}), []string{"Get", "Schema", "Set", "SetProviderKey"})

	assertEveryMethodConverts(t, wails.NewSettingsService(zap.NewNop(), wails.SettingsDeps{
		Declarations: declarationsFake{registry: registry, panics: true},
		Values:       registry.NewValues(),
		Store:        &storeFake{stored: map[string]json.RawMessage{}},
		Models:       &providerKeyFake{mode: panicking, fails: true},
	}), panicBody)
}

func TestSettingsServiceReportsWhatItsCollaboratorsRefuse(t *testing.T) {
	t.Parallel()

	broken := errors.New(errors.External, "the settings table is unreadable")
	registry := settings.New()
	registry.Int("runs.workers", 2, settings.IntRange(1, 16))

	cases := []struct {
		name string
		deps wails.SettingsDeps
		call func(*wails.SettingsService) error
	}{
		{
			name: "the declarations cannot be described",
			deps: wails.SettingsDeps{
				Declarations: declarationsFake{registry: registry, describes: broken},
				Values:       registry.NewValues(),
				Store:        &storeFake{stored: map[string]json.RawMessage{}},
				Models:       &providerKeyFake{},
			},
			call: func(service *wails.SettingsService) error {
				_, err := service.Schema(context.Background(), wails.SettingsSchemaRequest{})
				return err
			},
		},
		{
			name: "the store cannot be read",
			deps: wails.SettingsDeps{
				Declarations: declarationsFake{registry: registry},
				Values:       registry.NewValues(),
				Store:        &storeFake{stored: map[string]json.RawMessage{}, fail: broken},
				Models:       &providerKeyFake{},
			},
			call: func(service *wails.SettingsService) error {
				_, err := service.Get(context.Background(), wails.GetSettingRequest{Key: "runs.workers"})
				return err
			},
		},
		{
			name: "the store cannot be written",
			deps: wails.SettingsDeps{
				Declarations: declarationsFake{registry: registry},
				Values:       registry.NewValues(),
				Store:        &storeFake{stored: map[string]json.RawMessage{}, fail: broken},
				Models:       &providerKeyFake{},
			},
			call: func(service *wails.SettingsService) error {
				_, err := service.Set(context.Background(), wails.SetSettingRequest{
					Key: "runs.workers", Value: json.RawMessage("4"),
				})
				return err
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.call(wails.NewSettingsService(zap.NewNop(), tc.deps))
			if !errors.IsCode(err, errors.External) {
				t.Fatalf("error = %v, want %s", err, errors.External)
			}
		})
	}
}
