package wails

import (
	"context"

	"go.uber.org/zap"

	"github.com/davidmovas/postulator/internal/application/observe"
	"github.com/davidmovas/postulator/internal/application/pages"
	"github.com/davidmovas/postulator/internal/application/sitemodel"
	"github.com/davidmovas/postulator/internal/kernel/middleware"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

type PagesUseCase interface {
	Create(ctx context.Context, req pages.CreateRequest) (pages.CreateResponse, error)
	Update(ctx context.Context, req pages.UpdateRequest) (pages.UpdateResponse, error)
	Delete(ctx context.Context, req pages.DeleteRequest) (pages.DeleteResponse, error)
	Get(ctx context.Context, req pages.GetRequest) (pages.GetResponse, error)
	List(ctx context.Context, req pages.ListRequest) (paging.List[pages.Page], error)
	Tree(ctx context.Context, req pages.TreeRequest) (pages.TreeResponse, error)
	MapToEntity(ctx context.Context, req pages.MapToEntityRequest) (pages.MapToEntityResponse, error)
	Unmap(ctx context.Context, req pages.UnmapRequest) (pages.UnmapResponse, error)
	SetCanonical(ctx context.Context, req pages.SetCanonicalRequest) (pages.SetCanonicalResponse, error)
	ReplaceLinks(ctx context.Context, req pages.ReplaceLinksRequest) (pages.ReplaceLinksResponse, error)
	PreviewLink(ctx context.Context, req pages.PreviewLinkRequest) (pages.PreviewLinkResponse, error)
	SetType(ctx context.Context, req pages.SetTypeRequest) (pages.SetTypeResponse, error)
}

type ModelUseCase interface {
	Model(ctx context.Context, req observe.ModelRequest) (observe.ModelView, error)
	SummaryOf(ctx context.Context, req observe.SummaryRequest) (observe.Summary, error)
}

type PackUseCase interface {
	Pack(ctx context.Context, req sitemodel.PackRequest) (sitemodel.PackView, error)
}

type PagesService struct {
	create       middleware.Handler[pages.CreateRequest, pages.CreateResponse]
	update       middleware.Handler[pages.UpdateRequest, pages.UpdateResponse]
	remove       middleware.Handler[pages.DeleteRequest, pages.DeleteResponse]
	get          middleware.Handler[pages.GetRequest, pages.GetResponse]
	list         middleware.Handler[pages.ListRequest, paging.List[pages.Page]]
	tree         middleware.Handler[pages.TreeRequest, pages.TreeResponse]
	mapToEntity  middleware.Handler[pages.MapToEntityRequest, pages.MapToEntityResponse]
	unmap        middleware.Handler[pages.UnmapRequest, pages.UnmapResponse]
	setCanonical middleware.Handler[pages.SetCanonicalRequest, pages.SetCanonicalResponse]
	replaceLinks middleware.Handler[pages.ReplaceLinksRequest, pages.ReplaceLinksResponse]
	previewLink  middleware.Handler[pages.PreviewLinkRequest, pages.PreviewLinkResponse]
	setType      middleware.Handler[pages.SetTypeRequest, pages.SetTypeResponse]
	model        middleware.Handler[observe.ModelRequest, observe.ModelView]
	modelSummary middleware.Handler[observe.SummaryRequest, observe.Summary]
	writerPack   middleware.Handler[sitemodel.PackRequest, sitemodel.PackView]
}

func NewPagesService(logger *zap.Logger, useCase Source[PagesUseCase], models Source[ModelUseCase], packs Source[PackUseCase]) *PagesService {
	return &PagesService{
		create:       Wrap(logger, "pages.create", call(useCase, PagesUseCase.Create)),
		update:       Wrap(logger, "pages.update", call(useCase, PagesUseCase.Update)),
		remove:       Wrap(logger, "pages.delete", call(useCase, PagesUseCase.Delete)),
		get:          Wrap(logger, "pages.get", call(useCase, PagesUseCase.Get)),
		list:         Wrap(logger, "pages.list", call(useCase, PagesUseCase.List)),
		tree:         Wrap(logger, "pages.tree", call(useCase, PagesUseCase.Tree)),
		mapToEntity:  Wrap(logger, "pages.mapToEntity", call(useCase, PagesUseCase.MapToEntity)),
		unmap:        Wrap(logger, "pages.unmap", call(useCase, PagesUseCase.Unmap)),
		setCanonical: Wrap(logger, "pages.setCanonical", call(useCase, PagesUseCase.SetCanonical)),
		replaceLinks: Wrap(logger, "pages.replaceLinks", call(useCase, PagesUseCase.ReplaceLinks)),
		previewLink:  Wrap(logger, "pages.previewLink", call(useCase, PagesUseCase.PreviewLink)),
		setType:      Wrap(logger, "pages.setType", call(useCase, PagesUseCase.SetType)),
		model:        Wrap(logger, "pages.model", call(models, ModelUseCase.Model)),
		modelSummary: Wrap(logger, "pages.modelSummary", call(models, ModelUseCase.SummaryOf)),
		writerPack:   Wrap(logger, "pages.writerPack", call(packs, PackUseCase.Pack)),
	}
}

func (s *PagesService) WriterPack(c context.Context, req sitemodel.PackRequest) (sitemodel.PackView, error) {
	return s.writerPack(c, req)
}

func (s *PagesService) Model(c context.Context, req observe.ModelRequest) (observe.ModelView, error) {
	return s.model(c, req)
}

func (s *PagesService) ModelSummary(c context.Context, req observe.SummaryRequest) (observe.Summary, error) {
	return s.modelSummary(c, req)
}

func (s *PagesService) Create(c context.Context, req pages.CreateRequest) (pages.CreateResponse, error) {
	return s.create(c, req)
}

func (s *PagesService) Update(c context.Context, req pages.UpdateRequest) (pages.UpdateResponse, error) {
	return s.update(c, req)
}

func (s *PagesService) Delete(c context.Context, req pages.DeleteRequest) (pages.DeleteResponse, error) {
	return s.remove(c, req)
}

func (s *PagesService) Get(c context.Context, req pages.GetRequest) (pages.GetResponse, error) {
	return s.get(c, req)
}

func (s *PagesService) List(c context.Context, req pages.ListRequest) (paging.List[pages.Page], error) {
	return s.list(c, req)
}

func (s *PagesService) Tree(c context.Context, req pages.TreeRequest) (pages.TreeResponse, error) {
	return s.tree(c, req)
}

func (s *PagesService) MapToEntity(c context.Context, req pages.MapToEntityRequest) (pages.MapToEntityResponse, error) {
	return s.mapToEntity(c, req)
}

func (s *PagesService) Unmap(c context.Context, req pages.UnmapRequest) (pages.UnmapResponse, error) {
	return s.unmap(c, req)
}

func (s *PagesService) SetCanonical(c context.Context, req pages.SetCanonicalRequest) (pages.SetCanonicalResponse, error) {
	return s.setCanonical(c, req)
}

func (s *PagesService) ReplaceLinks(c context.Context, req pages.ReplaceLinksRequest) (pages.ReplaceLinksResponse, error) {
	return s.replaceLinks(c, req)
}

func (s *PagesService) PreviewLink(c context.Context, req pages.PreviewLinkRequest) (pages.PreviewLinkResponse, error) {
	return s.previewLink(c, req)
}

func (s *PagesService) SetType(c context.Context, req pages.SetTypeRequest) (pages.SetTypeResponse, error) {
	return s.setType(c, req)
}
