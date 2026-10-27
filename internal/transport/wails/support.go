package wails

import (
	"context"

	"go.uber.org/zap"

	"github.com/davidmovas/postulator/internal/application/support"
	"github.com/davidmovas/postulator/internal/kernel/middleware"
)

type SupportUseCase interface {
	Write(ctx context.Context, req support.WriteRequest) (support.WriteResponse, error)
}

type SupportService struct {
	write middleware.Handler[support.WriteRequest, support.WriteResponse]
}

func NewSupportService(logger *zap.Logger, useCase Source[SupportUseCase]) *SupportService {
	return &SupportService{write: Wrap(logger, "support.writeBundle", call(useCase, SupportUseCase.Write))}
}

func (s *SupportService) WriteBundle(c context.Context, req support.WriteRequest) (support.WriteResponse, error) {
	return s.write(c, req)
}
