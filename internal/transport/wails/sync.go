package wails

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"

	"github.com/davidmovas/postulator/internal/application/sync"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/middleware"
)

const packageFileMode = 0o600

type syncUseCase interface {
	SyncSite(ctx context.Context, req sync.SyncSiteRequest) (sync.SyncSiteResponse, error)
	CheckPlugin(ctx context.Context, req sync.CheckPluginRequest) (sync.CheckPluginResponse, error)
	PluginPackage(ctx context.Context, req sync.PluginPackageRequest) (sync.PluginPackageResponse, error)
}

type SavePluginPackageRequest struct {
	Path string `json:"path"`
}

type SavePluginPackageResponse struct {
	Path     string `json:"path"`
	Filename string `json:"filename"`
	Bytes    int    `json:"bytes"`
}

type SyncService struct {
	syncSite          middleware.Handler[sync.SyncSiteRequest, sync.SyncSiteResponse]
	checkPlugin       middleware.Handler[sync.CheckPluginRequest, sync.CheckPluginResponse]
	savePluginPackage middleware.Handler[SavePluginPackageRequest, SavePluginPackageResponse]
}

func NewSyncService(logger *zap.Logger, useCase syncUseCase) *SyncService {
	return &SyncService{
		syncSite:          Wrap(logger, "sync.syncSite", useCase.SyncSite),
		checkPlugin:       Wrap(logger, "sync.checkPlugin", useCase.CheckPlugin),
		savePluginPackage: Wrap(logger, "sync.savePluginPackage", savePluginPackage(useCase)),
	}
}

func savePluginPackage(useCase syncUseCase) middleware.Handler[SavePluginPackageRequest, SavePluginPackageResponse] {
	return func(c context.Context, req SavePluginPackageRequest) (SavePluginPackageResponse, error) {
		target := strings.TrimSpace(req.Path)
		if target == "" {
			return SavePluginPackageResponse{}, errors.New(errors.Invalid, "the destination path must not be empty").
				WithDetail("field", "path")
		}

		packaged, err := useCase.PluginPackage(c, sync.PluginPackageRequest{})
		if err != nil {
			return SavePluginPackageResponse{}, err
		}

		if info, statErr := os.Stat(target); statErr == nil && info.IsDir() {
			target = filepath.Join(target, packaged.Filename)
		}
		if err = os.WriteFile(target, packaged.Bytes, packageFileMode); err != nil {
			return SavePluginPackageResponse{}, errors.New(errors.Invalid, "the companion plugin archive could not be written").
				WithDetail("path", target).WithInternal(err)
		}

		return SavePluginPackageResponse{Path: target, Filename: packaged.Filename, Bytes: len(packaged.Bytes)}, nil
	}
}

func (s *SyncService) SyncSite(c context.Context, req sync.SyncSiteRequest) (sync.SyncSiteResponse, error) {
	return s.syncSite(c, req)
}

func (s *SyncService) CheckPlugin(c context.Context, req sync.CheckPluginRequest) (sync.CheckPluginResponse, error) {
	return s.checkPlugin(c, req)
}

func (s *SyncService) SavePluginPackage(c context.Context, req SavePluginPackageRequest) (SavePluginPackageResponse, error) {
	return s.savePluginPackage(c, req)
}
