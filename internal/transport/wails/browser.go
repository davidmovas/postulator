package wails

import (
	"context"

	"go.uber.org/zap"

	"github.com/davidmovas/postulator/internal/application/browser"
	"github.com/davidmovas/postulator/internal/kernel/middleware"
)

type BrowserUseCase interface {
	Open(ctx context.Context, req browser.OpenRequest) (browser.OpenResponse, error)
	Locate(ctx context.Context, req browser.LocateRequest) (browser.LocateResponse, error)
}

type BrowserService struct {
	open   middleware.Handler[browser.OpenRequest, browser.OpenResponse]
	locate middleware.Handler[browser.LocateRequest, browser.LocateResponse]
}

func NewBrowserService(logger *zap.Logger, useCase Source[BrowserUseCase]) *BrowserService {
	return &BrowserService{
		open:   Wrap(logger, "browser.open", call(useCase, BrowserUseCase.Open)),
		locate: Wrap(logger, "browser.locate", call(useCase, BrowserUseCase.Locate)),
	}
}

func (s *BrowserService) Open(c context.Context, req browser.OpenRequest) (browser.OpenResponse, error) {
	return s.open(c, req)
}

func (s *BrowserService) Locate(c context.Context, req browser.LocateRequest) (browser.LocateResponse, error) {
	return s.locate(c, req)
}
