package wails_test

import (
	"context"
	"testing"

	"go.uber.org/zap"

	"github.com/davidmovas/postulator/internal/application/sites"
	"github.com/davidmovas/postulator/internal/kernel/paging"
	"github.com/davidmovas/postulator/internal/transport/wails"
)

type sitesFake struct{ mode failure }

func (f sitesFake) Create(context.Context, sites.CreateRequest) (sites.CreateResponse, error) {
	return answer[sites.CreateResponse](f.mode)
}

func (f sitesFake) Update(context.Context, sites.UpdateRequest) (sites.UpdateResponse, error) {
	return answer[sites.UpdateResponse](f.mode)
}

func (f sitesFake) Delete(context.Context, sites.DeleteRequest) (sites.DeleteResponse, error) {
	return answer[sites.DeleteResponse](f.mode)
}

func (f sitesFake) Get(context.Context, sites.GetRequest) (sites.GetResponse, error) {
	return answer[sites.GetResponse](f.mode)
}

func (f sitesFake) List(context.Context, sites.ListRequest) (paging.List[sites.Site], error) {
	return answer[paging.List[sites.Site]](f.mode)
}

func TestSitesServiceConvertsEveryFailure(t *testing.T) {
	t.Parallel()

	assertMethodNames(t, wails.NewSitesService(zap.NewNop(), ready[wails.SitesUseCase](sitesFake{})), []string{
		"Create", "Delete", "Get", "List", "Update",
	})
	assertEveryMethodConverts(t, wails.NewSitesService(zap.NewNop(), ready[wails.SitesUseCase](sitesFake{mode: missing})), missingBody)
	assertEveryMethodConverts(t, wails.NewSitesService(zap.NewNop(), ready[wails.SitesUseCase](sitesFake{mode: panicking})), panicBody)
}
