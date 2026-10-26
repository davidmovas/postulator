package wails_test

import (
	"context"
	"testing"

	"go.uber.org/zap"

	"github.com/davidmovas/postulator/internal/application/support"
	"github.com/davidmovas/postulator/internal/transport/wails"
)

type supportFake struct{ mode failure }

func (f supportFake) Write(context.Context, support.WriteRequest) (support.WriteResponse, error) {
	return answer[support.WriteResponse](f.mode)
}

func TestSupportServiceConvertsEveryFailure(t *testing.T) {
	t.Parallel()

	assertMethodNames(t, wails.NewSupportService(zap.NewNop(), ready[wails.SupportUseCase](supportFake{})), []string{"WriteBundle"})
	assertEveryMethodConverts(t, wails.NewSupportService(zap.NewNop(), ready[wails.SupportUseCase](supportFake{mode: missing})), missingBody)
	assertEveryMethodConverts(t, wails.NewSupportService(zap.NewNop(), ready[wails.SupportUseCase](supportFake{mode: panicking})), panicBody)
}
