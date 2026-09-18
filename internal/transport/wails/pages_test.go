package wails_test

import (
	"context"
	"testing"

	"go.uber.org/zap"

	"github.com/davidmovas/postulator/internal/application/pages"
	"github.com/davidmovas/postulator/internal/kernel/paging"
	"github.com/davidmovas/postulator/internal/transport/wails"
)

type pagesFake struct{ mode failure }

func (f pagesFake) Create(context.Context, pages.CreateRequest) (pages.CreateResponse, error) {
	return answer[pages.CreateResponse](f.mode)
}

func (f pagesFake) Update(context.Context, pages.UpdateRequest) (pages.UpdateResponse, error) {
	return answer[pages.UpdateResponse](f.mode)
}

func (f pagesFake) Delete(context.Context, pages.DeleteRequest) (pages.DeleteResponse, error) {
	return answer[pages.DeleteResponse](f.mode)
}

func (f pagesFake) Get(context.Context, pages.GetRequest) (pages.GetResponse, error) {
	return answer[pages.GetResponse](f.mode)
}

func (f pagesFake) List(context.Context, pages.ListRequest) (paging.List[pages.Page], error) {
	return answer[paging.List[pages.Page]](f.mode)
}

func (f pagesFake) Tree(context.Context, pages.TreeRequest) (pages.TreeResponse, error) {
	return answer[pages.TreeResponse](f.mode)
}

func (f pagesFake) MapToEntity(context.Context, pages.MapToEntityRequest) (pages.MapToEntityResponse, error) {
	return answer[pages.MapToEntityResponse](f.mode)
}

func (f pagesFake) Unmap(context.Context, pages.UnmapRequest) (pages.UnmapResponse, error) {
	return answer[pages.UnmapResponse](f.mode)
}

func (f pagesFake) SetCanonical(context.Context, pages.SetCanonicalRequest) (pages.SetCanonicalResponse, error) {
	return answer[pages.SetCanonicalResponse](f.mode)
}

func (f pagesFake) ReplaceLinks(context.Context, pages.ReplaceLinksRequest) (pages.ReplaceLinksResponse, error) {
	return answer[pages.ReplaceLinksResponse](f.mode)
}

func TestPagesServiceConvertsEveryFailure(t *testing.T) {
	t.Parallel()

	assertMethodNames(t, wails.NewPagesService(zap.NewNop(), ready[wails.PagesUseCase](pagesFake{})), []string{
		"Create", "Delete", "Get", "List", "MapToEntity", "ReplaceLinks", "SetCanonical", "Tree", "Unmap", "Update",
	})
	assertEveryMethodConverts(t, wails.NewPagesService(zap.NewNop(), ready[wails.PagesUseCase](pagesFake{mode: missing})), missingBody)
	assertEveryMethodConverts(t, wails.NewPagesService(zap.NewNop(), ready[wails.PagesUseCase](pagesFake{mode: panicking})), panicBody)
}
