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
	removed string
}

func (p *providerKeyFake) SetProviderKey(_ context.Context, req models.SetProviderKeyRequest) (models.SetProviderKeyResponse, error) {
	if p.fails {
		return answer[models.SetProviderKeyResponse](p.mode)
	}
	p.written = req.APIKey
	return models.SetProviderKeyResponse{Provider: req.Provider}, nil
}

func (p *providerKeyFake) ProviderKeys(context.Context, models.ProviderKeysRequest) (models.ProviderKeysResponse, error) {
	if p.fails {
		return answer[models.ProviderKeysResponse](p.mode)
	}
	return models.ProviderKeysResponse{Providers: []models.ProviderKey{
		{Provider: "anthropic"},
		{Provider: "openai", Configured: p.written != ""},
	}}, nil
}

func (p *providerKeyFake) DeleteProviderKey(_ context.Context, req models.DeleteProviderKeyRequest) (models.DeleteProviderKeyResponse, error) {
	if p.fails {
		return answer[models.DeleteProviderKeyResponse](p.mode)
	}
	p.removed = req.Provider
	p.written = ""
	return models.DeleteProviderKeyResponse(req), nil
}

func settingsHarness(t *testing.T) (deps wails.SettingsDeps, workers *settings.Setting[int], store *storeFake, keys *providerKeyFake, live *settings.Values) {
	t.Helper()

	registry := settings.New()
	workers = registry.Int("runs.workers", 2, settings.IntRange(1, 16))
	store = &storeFake{stored: map[string]json.RawMessage{}}
	keys = &providerKeyFake{}
	live = registry.NewValues()

	return wails.SettingsDeps{
		Access: ready(wails.SettingsAccess{
			Declarations: declarationsFake{registry: registry},
			Values:       live,
			Store:        store,
			Models:       keys,
		}),
		Backup: ready[wails.BackupControl](&backupFake{}),
		Lock:   &lockFake{},
	}, workers, store, keys, live
}

func TestSettingsServiceDescribesEveryDeclaration(t *testing.T) {
	t.Parallel()

	deps, _, _, _, _ := settingsHarness(t)

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

	deps, _, store, _, _ := settingsHarness(t)
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

	deps, workers, store, _, live := settingsHarness(t)
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
	if got := workers.Get(live); got != 8 {
		t.Errorf("the live value is %d, want the 8 that was just written", got)
	}
}

func TestSettingsServiceRefusesAValueTheDeclarationRejects(t *testing.T) {
	t.Parallel()

	deps, workers, store, _, live := settingsHarness(t)
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
			if got := workers.Get(live); got != 2 {
				t.Fatalf("the live value is %d, want the untouched default", got)
			}
		})
	}
}

func TestSettingsServiceDelegatesTheProviderKey(t *testing.T) {
	t.Parallel()

	deps, _, _, keys, _ := settingsHarness(t)

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

func TestSettingsServiceReportsAndRevokesProviderKeysWithoutTheirValues(t *testing.T) {
	t.Parallel()

	deps, _, _, keys, _ := settingsHarness(t)
	service := wails.NewSettingsService(zap.NewNop(), deps)

	if _, err := service.SetProviderKey(context.Background(),
		models.SetProviderKeyRequest{Provider: "openai", APIKey: "sk-secret"}); err != nil {
		t.Fatalf("SetProviderKey: %v", err)
	}

	reported, err := service.ProviderKeys(context.Background(), models.ProviderKeysRequest{})
	if err != nil {
		t.Fatalf("ProviderKeys: %v", err)
	}

	encoded, marshalErr := json.Marshal(reported)
	if marshalErr != nil {
		t.Fatalf("Marshal: %v", marshalErr)
	}
	if string(encoded) != `{"providers":[{"provider":"anthropic","configured":false},{"provider":"openai","configured":true}]}` {
		t.Fatalf("ProviderKeys = %s, want booleans alone", encoded)
	}

	revoked, err := service.DeleteProviderKey(context.Background(),
		models.DeleteProviderKeyRequest{Provider: "openai"})
	if err != nil {
		t.Fatalf("DeleteProviderKey: %v", err)
	}
	if revoked.Provider != "openai" || keys.removed != "openai" {
		t.Fatalf("DeleteProviderKey = %+v, the use case received %q", revoked, keys.removed)
	}

	reported, err = service.ProviderKeys(context.Background(), models.ProviderKeysRequest{})
	if err != nil {
		t.Fatalf("ProviderKeys: %v", err)
	}
	if reported.Providers[1].Configured {
		t.Fatalf("ProviderKeys = %+v, want openai unconfigured after the revoke", reported.Providers)
	}
}

func TestSettingsServiceConvertsEveryFailure(t *testing.T) {
	t.Parallel()

	registry := settings.New()
	registry.Int("runs.workers", 2, settings.IntRange(1, 16))

	assertMethodNames(t, wails.NewSettingsService(zap.NewNop(), wails.SettingsDeps{
		Access: ready(wails.SettingsAccess{
			Declarations: declarationsFake{registry: registry},
			Values:       registry.NewValues(),
			Store:        &storeFake{stored: map[string]json.RawMessage{}},
			Models:       &providerKeyFake{},
		}),
		Backup: ready[wails.BackupControl](&backupFake{}),
		Lock:   &lockFake{},
	}), []string{
		"DeleteProviderKey", "ExportBackup", "Get", "ImportBackup", "Lock", "LockState",
		"ProviderKeys", "Schema", "Set", "SetMasterPassword", "SetProviderKey", "Unlock",
	})

	assertEveryMethodConverts(t, wails.NewSettingsService(zap.NewNop(), wails.SettingsDeps{
		Access: ready(wails.SettingsAccess{
			Declarations: declarationsFake{registry: registry, panics: true},
			Values:       registry.NewValues(),
			Store:        &storeFake{stored: map[string]json.RawMessage{}},
			Models:       &providerKeyFake{mode: panicking, fails: true},
		}),
		Backup: ready[wails.BackupControl](&backupFake{mode: panicking, fails: true}),
		Lock:   &lockFake{mode: panicking, fails: true},
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
				Access: ready(wails.SettingsAccess{
					Declarations: declarationsFake{registry: registry, describes: broken},
					Values:       registry.NewValues(),
					Store:        &storeFake{stored: map[string]json.RawMessage{}},
					Models:       &providerKeyFake{},
				}),
				Backup: ready[wails.BackupControl](&backupFake{}),
				Lock:   &lockFake{},
			},
			call: func(service *wails.SettingsService) error {
				_, err := service.Schema(context.Background(), wails.SettingsSchemaRequest{})
				return err
			},
		},
		{
			name: "the store cannot be read",
			deps: wails.SettingsDeps{
				Access: ready(wails.SettingsAccess{
					Declarations: declarationsFake{registry: registry},
					Values:       registry.NewValues(),
					Store:        &storeFake{stored: map[string]json.RawMessage{}, fail: broken},
					Models:       &providerKeyFake{},
				}),
				Backup: ready[wails.BackupControl](&backupFake{}),
				Lock:   &lockFake{},
			},
			call: func(service *wails.SettingsService) error {
				_, err := service.Get(context.Background(), wails.GetSettingRequest{Key: "runs.workers"})
				return err
			},
		},
		{
			name: "the store cannot be written",
			deps: wails.SettingsDeps{
				Access: ready(wails.SettingsAccess{
					Declarations: declarationsFake{registry: registry},
					Values:       registry.NewValues(),
					Store:        &storeFake{stored: map[string]json.RawMessage{}, fail: broken},
					Models:       &providerKeyFake{},
				}),
				Backup: ready[wails.BackupControl](&backupFake{}),
				Lock:   &lockFake{},
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

type lockFake struct {
	mode      failure
	fails     bool
	locked    bool
	protected bool
	current   string
	next      string
	password  string
}

func (l *lockFake) Locked() bool {
	return l.locked
}

func (l *lockFake) Protected() (bool, error) {
	if l.fails {
		_, err := answer[bool](l.mode)
		return false, err
	}
	return l.protected, nil
}

func (l *lockFake) Lock() error {
	if l.fails {
		_, err := answer[bool](l.mode)
		return err
	}
	l.locked = true
	return nil
}

func (l *lockFake) Unlock(_ context.Context, password string) error {
	if l.fails {
		_, err := answer[bool](l.mode)
		return err
	}
	l.password = password
	l.locked = false
	return nil
}

func (l *lockFake) SetMasterPassword(_ context.Context, current, next string) error {
	if l.fails {
		_, err := answer[bool](l.mode)
		return err
	}
	l.current, l.next = current, next
	l.protected = next != ""
	return nil
}

type backupFake struct {
	mode    failure
	fails   bool
	written string
	read    string
}

func (b *backupFake) ExportBackup(_ context.Context, path, password string) (int64, error) {
	if b.fails {
		return answer[int64](b.mode)
	}
	b.written = path + "|" + password
	return 4096, nil
}

func (b *backupFake) ImportBackup(_ context.Context, path, password string) error {
	if b.fails {
		_, err := answer[int64](b.mode)
		return err
	}
	b.read = path + "|" + password
	return nil
}

func lockHarness(t *testing.T) (*wails.SettingsService, *lockFake, *backupFake) {
	t.Helper()

	registry := settings.New()
	registry.Int("runs.workers", 2, settings.IntRange(1, 16))
	guard := &lockFake{protected: true, locked: true}
	backup := &backupFake{}

	return wails.NewSettingsService(zap.NewNop(), wails.SettingsDeps{
		Access: ready(wails.SettingsAccess{
			Declarations: declarationsFake{registry: registry},
			Values:       registry.NewValues(),
			Store:        &storeFake{stored: map[string]json.RawMessage{}},
			Models:       &providerKeyFake{},
		}),
		Backup: ready[wails.BackupControl](backup),
		Lock:   guard,
	}), guard, backup
}

func TestSettingsServiceDrivesTheLock(t *testing.T) {
	t.Parallel()

	service, guard, _ := lockHarness(t)

	state, err := service.LockState(context.Background(), wails.LockStateRequest{})
	if err != nil {
		t.Fatalf("LockState: %v", err)
	}
	if !state.Locked || !state.Protected {
		t.Fatalf("LockState = %+v, want a locked and protected application", state)
	}

	state, err = service.Unlock(context.Background(), wails.UnlockRequest{Password: "hunter2"})
	if err != nil {
		t.Fatalf("Unlock: %v", err)
	}
	if state.Locked || guard.password != "hunter2" {
		t.Fatalf("Unlock = %+v with the password %q", state, guard.password)
	}

	state, err = service.Lock(context.Background(), wails.LockRequest{})
	if err != nil {
		t.Fatalf("Lock: %v", err)
	}
	if !state.Locked {
		t.Fatalf("Lock = %+v, want a locked application", state)
	}

	state, err = service.SetMasterPassword(context.Background(),
		wails.SetMasterPasswordRequest{Current: "hunter2", New: ""})
	if err != nil {
		t.Fatalf("SetMasterPassword: %v", err)
	}
	if state.Protected || guard.current != "hunter2" || guard.next != "" {
		t.Fatalf("SetMasterPassword = %+v after %q -> %q", state, guard.current, guard.next)
	}
}

func TestSettingsServiceCarriesTheBackupPathAndPassword(t *testing.T) {
	t.Parallel()

	service, _, backup := lockHarness(t)

	written, err := service.ExportBackup(context.Background(),
		wails.ExportBackupRequest{Path: "  C:/backups/postulator.pstx  ", Password: "hunter2"})
	if err != nil {
		t.Fatalf("ExportBackup: %v", err)
	}
	if written.Path != "C:/backups/postulator.pstx" || written.Bytes != 4096 {
		t.Fatalf("ExportBackup = %+v", written)
	}
	if backup.written != "C:/backups/postulator.pstx|hunter2" {
		t.Fatalf("the archive received %q", backup.written)
	}

	restored, err := service.ImportBackup(context.Background(),
		wails.ImportBackupRequest{Path: "C:/backups/postulator.pstx", Password: "hunter2"})
	if err != nil {
		t.Fatalf("ImportBackup: %v", err)
	}
	if restored.Path != "C:/backups/postulator.pstx" || backup.read != "C:/backups/postulator.pstx|hunter2" {
		t.Fatalf("ImportBackup = %+v, the archive received %q", restored, backup.read)
	}
}
