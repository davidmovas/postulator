package wails_test

import (
	"context"
	"testing"

	"go.uber.org/zap"

	"github.com/davidmovas/postulator/internal/application/browser"
	"github.com/davidmovas/postulator/internal/transport/wails"
)

type browserFake struct{ mode failure }

func (f browserFake) Open(context.Context, browser.OpenRequest) (browser.OpenResponse, error) {
	return answer[browser.OpenResponse](f.mode)
}

func (f browserFake) Locate(context.Context, browser.LocateRequest) (browser.LocateResponse, error) {
	return answer[browser.LocateResponse](f.mode)
}

func TestBrowserServiceConvertsEveryFailure(t *testing.T) {
	t.Parallel()

	assertMethodNames(t, wails.NewBrowserService(zap.NewNop(), ready[wails.BrowserUseCase](browserFake{})),
		[]string{"Locate", "Open"})
	assertEveryMethodConverts(t, wails.NewBrowserService(zap.NewNop(), ready[wails.BrowserUseCase](browserFake{mode: missing})), missingBody)
	assertEveryMethodConverts(t, wails.NewBrowserService(zap.NewNop(), ready[wails.BrowserUseCase](browserFake{mode: panicking})), panicBody)
}
