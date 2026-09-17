package wails

import (
	"context"

	"go.uber.org/zap"

	"github.com/davidmovas/postulator/internal/kernel/middleware"
)

type BuildInfo struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildDate string `json:"buildDate"`
}

type PingRequest struct{}

type HealthService struct {
	ping middleware.Handler[PingRequest, BuildInfo]
}

func NewHealthService(logger *zap.Logger, build BuildInfo) *HealthService {
	return &HealthService{
		ping: Wrap(logger, "health.ping", func(_ context.Context, _ PingRequest) (BuildInfo, error) {
			return build, nil
		}),
	}
}

func (s *HealthService) Ping(c context.Context, req PingRequest) (BuildInfo, error) {
	return s.ping(c, req)
}
