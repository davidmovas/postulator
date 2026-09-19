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

type SettingsDeclarations interface {
	Schema() ([]settings.Descriptor, error)
	Validate(key string, value json.RawMessage) error
	Apply(values *settings.Values, stored map[string]json.RawMessage) ([]string, error)
}

type SettingsStore interface {
	Get(ctx context.Context, key string) (json.RawMessage, bool, error)
	Set(ctx context.Context, key string, value json.RawMessage) error
	All(ctx context.Context) (map[string]json.RawMessage, error)
}

type ProviderKeyAccess interface {
	SetProviderKey(ctx context.Context, req models.SetProviderKeyRequest) (models.SetProviderKeyResponse, error)
	ProviderKeys(ctx context.Context, req models.ProviderKeysRequest) (models.ProviderKeysResponse, error)
	DeleteProviderKey(ctx context.Context, req models.DeleteProviderKeyRequest) (models.DeleteProviderKeyResponse, error)
}

type SettingsAccess struct {
	Declarations SettingsDeclarations
	Values       *settings.Values
	Store        SettingsStore
	Models       ProviderKeyAccess
}

type LockControl interface {
	Locked() bool
	Protected() (bool, error)
	Lock() error
	Unlock(ctx context.Context, password string) error
	SetMasterPassword(ctx context.Context, current, next string) error
}

type BackupControl interface {
	ExportBackup(ctx context.Context, path, password string) (int64, error)
	ImportBackup(ctx context.Context, path, password string) error
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

type LockStateRequest struct{}

type LockStateResponse struct {
	Locked    bool `json:"locked"`
	Protected bool `json:"protected"`
}

type LockRequest struct{}

type UnlockRequest struct {
	Password string `json:"password"`
}

type SetMasterPasswordRequest struct {
	Current string `json:"current"`
	New     string `json:"new"`
}

type ExportBackupRequest struct {
	Path     string `json:"path"`
	Password string `json:"password"`
}

type ExportBackupResponse struct {
	Path  string `json:"path"`
	Bytes int64  `json:"bytes"`
}

type ImportBackupRequest struct {
	Path     string `json:"path"`
	Password string `json:"password"`
}

type ImportBackupResponse struct {
	Path string `json:"path"`
}

type SettingsDeps struct {
	Access Source[SettingsAccess]
	Backup Source[BackupControl]
	Lock   LockControl
}

type SettingsService struct {
	schema            middleware.Handler[SettingsSchemaRequest, SettingsSchemaResponse]
	get               middleware.Handler[GetSettingRequest, GetSettingResponse]
	set               middleware.Handler[SetSettingRequest, SetSettingResponse]
	setProviderKey    middleware.Handler[models.SetProviderKeyRequest, models.SetProviderKeyResponse]
	providerKeys      middleware.Handler[models.ProviderKeysRequest, models.ProviderKeysResponse]
	deleteProviderKey middleware.Handler[models.DeleteProviderKeyRequest, models.DeleteProviderKeyResponse]
	lockState         middleware.Handler[LockStateRequest, LockStateResponse]
	lock              middleware.Handler[LockRequest, LockStateResponse]
	unlock            middleware.Handler[UnlockRequest, LockStateResponse]
	setMasterPassword middleware.Handler[SetMasterPasswordRequest, LockStateResponse]
	exportBackup      middleware.Handler[ExportBackupRequest, ExportBackupResponse]
	importBackup      middleware.Handler[ImportBackupRequest, ImportBackupResponse]
}

func NewSettingsService(logger *zap.Logger, deps SettingsDeps) *SettingsService {
	return &SettingsService{
		schema:            Wrap(logger, "settings.schema", describeSettings(deps)),
		get:               Wrap(logger, "settings.get", readSetting(deps)),
		set:               Wrap(logger, "settings.set", writeSetting(deps)),
		setProviderKey:    Wrap(logger, "settings.setProviderKey", writeProviderKey(deps)),
		providerKeys:      Wrap(logger, "settings.providerKeys", readProviderKeys(deps)),
		deleteProviderKey: Wrap(logger, "settings.deleteProviderKey", removeProviderKey(deps)),
		lockState:         Wrap(logger, "settings.lockState", readLockState(deps)),
		lock:              Wrap(logger, "settings.lock", lockApplication(deps)),
		unlock:            Wrap(logger, "settings.unlock", unlockApplication(deps)),
		setMasterPassword: Wrap(logger, "settings.setMasterPassword", writeMasterPassword(deps)),
		exportBackup:      Wrap(logger, "settings.exportBackup", writeBackup(deps)),
		importBackup:      Wrap(logger, "settings.importBackup", readBackup(deps)),
	}
}

func describeSettings(deps SettingsDeps) middleware.Handler[SettingsSchemaRequest, SettingsSchemaResponse] {
	return func(_ context.Context, _ SettingsSchemaRequest) (SettingsSchemaResponse, error) {
		access, err := deps.Access()
		if err != nil {
			return SettingsSchemaResponse{}, err
		}

		declared, err := access.Declarations.Schema()
		if err != nil {
			return SettingsSchemaResponse{}, err
		}
		return SettingsSchemaResponse{Settings: declared}, nil
	}
}

func readSetting(deps SettingsDeps) middleware.Handler[GetSettingRequest, GetSettingResponse] {
	return func(c context.Context, req GetSettingRequest) (GetSettingResponse, error) {
		access, err := deps.Access()
		if err != nil {
			return GetSettingResponse{}, err
		}

		key := strings.TrimSpace(req.Key)
		declared, err := declaration(access, key)
		if err != nil {
			return GetSettingResponse{}, err
		}

		stored, found, err := access.Store.Get(c, key)
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
		access, err := deps.Access()
		if err != nil {
			return SetSettingResponse{}, err
		}

		key := strings.TrimSpace(req.Key)
		if err := access.Declarations.Validate(key, req.Value); err != nil {
			return SetSettingResponse{}, err
		}
		if err := access.Store.Set(c, key, req.Value); err != nil {
			return SetSettingResponse{}, err
		}

		stored, err := access.Store.All(c)
		if err != nil {
			return SetSettingResponse{}, err
		}
		if _, err = access.Declarations.Apply(access.Values, stored); err != nil {
			return SetSettingResponse{}, err
		}
		return SetSettingResponse{Key: key, Value: req.Value}, nil
	}
}

func writeProviderKey(deps SettingsDeps) middleware.Handler[models.SetProviderKeyRequest, models.SetProviderKeyResponse] {
	return func(c context.Context, req models.SetProviderKeyRequest) (models.SetProviderKeyResponse, error) {
		access, err := deps.Access()
		if err != nil {
			return models.SetProviderKeyResponse{}, err
		}
		return access.Models.SetProviderKey(c, req)
	}
}

func readProviderKeys(deps SettingsDeps) middleware.Handler[models.ProviderKeysRequest, models.ProviderKeysResponse] {
	return func(c context.Context, req models.ProviderKeysRequest) (models.ProviderKeysResponse, error) {
		access, err := deps.Access()
		if err != nil {
			return models.ProviderKeysResponse{}, err
		}
		return access.Models.ProviderKeys(c, req)
	}
}

func removeProviderKey(deps SettingsDeps) middleware.Handler[models.DeleteProviderKeyRequest, models.DeleteProviderKeyResponse] {
	return func(c context.Context, req models.DeleteProviderKeyRequest) (models.DeleteProviderKeyResponse, error) {
		access, err := deps.Access()
		if err != nil {
			return models.DeleteProviderKeyResponse{}, err
		}
		return access.Models.DeleteProviderKey(c, req)
	}
}

func declaration(access SettingsAccess, key string) (settings.Descriptor, error) {
	declared, err := access.Declarations.Schema()
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

func readLockState(deps SettingsDeps) middleware.Handler[LockStateRequest, LockStateResponse] {
	return func(_ context.Context, _ LockStateRequest) (LockStateResponse, error) {
		return lockState(deps)
	}
}

func lockApplication(deps SettingsDeps) middleware.Handler[LockRequest, LockStateResponse] {
	return func(_ context.Context, _ LockRequest) (LockStateResponse, error) {
		if err := deps.Lock.Lock(); err != nil {
			return LockStateResponse{}, err
		}
		return lockState(deps)
	}
}

func unlockApplication(deps SettingsDeps) middleware.Handler[UnlockRequest, LockStateResponse] {
	return func(c context.Context, req UnlockRequest) (LockStateResponse, error) {
		if err := deps.Lock.Unlock(c, req.Password); err != nil {
			return LockStateResponse{}, err
		}
		return lockState(deps)
	}
}

func writeMasterPassword(deps SettingsDeps) middleware.Handler[SetMasterPasswordRequest, LockStateResponse] {
	return func(c context.Context, req SetMasterPasswordRequest) (LockStateResponse, error) {
		if err := deps.Lock.SetMasterPassword(c, req.Current, req.New); err != nil {
			return LockStateResponse{}, err
		}
		return lockState(deps)
	}
}

func lockState(deps SettingsDeps) (LockStateResponse, error) {
	protected, err := deps.Lock.Protected()
	if err != nil {
		return LockStateResponse{}, err
	}
	return LockStateResponse{Locked: deps.Lock.Locked(), Protected: protected}, nil
}

func writeBackup(deps SettingsDeps) middleware.Handler[ExportBackupRequest, ExportBackupResponse] {
	return func(c context.Context, req ExportBackupRequest) (ExportBackupResponse, error) {
		backup, err := deps.Backup()
		if err != nil {
			return ExportBackupResponse{}, err
		}

		path := strings.TrimSpace(req.Path)
		written, err := backup.ExportBackup(c, path, req.Password)
		if err != nil {
			return ExportBackupResponse{}, err
		}
		return ExportBackupResponse{Path: path, Bytes: written}, nil
	}
}

func readBackup(deps SettingsDeps) middleware.Handler[ImportBackupRequest, ImportBackupResponse] {
	return func(c context.Context, req ImportBackupRequest) (ImportBackupResponse, error) {
		backup, err := deps.Backup()
		if err != nil {
			return ImportBackupResponse{}, err
		}

		path := strings.TrimSpace(req.Path)
		if err := backup.ImportBackup(c, path, req.Password); err != nil {
			return ImportBackupResponse{}, err
		}
		return ImportBackupResponse{Path: path}, nil
	}
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

func (s *SettingsService) ProviderKeys(c context.Context, req models.ProviderKeysRequest) (models.ProviderKeysResponse, error) {
	return s.providerKeys(c, req)
}

func (s *SettingsService) DeleteProviderKey(c context.Context, req models.DeleteProviderKeyRequest) (models.DeleteProviderKeyResponse, error) {
	return s.deleteProviderKey(c, req)
}

func (s *SettingsService) LockState(c context.Context, req LockStateRequest) (LockStateResponse, error) {
	return s.lockState(c, req)
}

func (s *SettingsService) Lock(c context.Context, req LockRequest) (LockStateResponse, error) {
	return s.lock(c, req)
}

func (s *SettingsService) Unlock(c context.Context, req UnlockRequest) (LockStateResponse, error) {
	return s.unlock(c, req)
}

func (s *SettingsService) SetMasterPassword(c context.Context, req SetMasterPasswordRequest) (LockStateResponse, error) {
	return s.setMasterPassword(c, req)
}

func (s *SettingsService) ExportBackup(c context.Context, req ExportBackupRequest) (ExportBackupResponse, error) {
	return s.exportBackup(c, req)
}

func (s *SettingsService) ImportBackup(c context.Context, req ImportBackupRequest) (ImportBackupResponse, error) {
	return s.importBackup(c, req)
}
