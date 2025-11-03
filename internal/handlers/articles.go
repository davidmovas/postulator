package handlers

import (
	"github.com/davidmovas/postulator/internal/domain/articles"
	"github.com/davidmovas/postulator/internal/domain/jobs/execution"
	"github.com/davidmovas/postulator/internal/dto"
	"github.com/davidmovas/postulator/pkg/ctx"
)

type ArticlesHandler struct {
	service          articles.Service
	executionService execution.Service
}

func NewArticlesHandler(service articles.Service, executionService execution.Service) *ArticlesHandler {
	return &ArticlesHandler{
		service:          service,
		executionService: executionService,
	}
}

func (h *ArticlesHandler) CreateArticle(article *dto.Article) *dto.Response[string] {
	entity, err := article.ToEntity()
	if err != nil {
		return fail[string](err)
	}

	if err = h.service.CreateArticle(ctx.FastCtx(), entity); err != nil {
		return fail[string](err)
	}

	return ok("Article created successfully")
}

func (h *ArticlesHandler) GetArticle(id int64) *dto.Response[*dto.Article] {
	article, err := h.service.GetArticle(ctx.FastCtx(), id)
	if err != nil {
		return fail[*dto.Article](err)
	}

	return ok(dto.NewArticle(article))
}

func (h *ArticlesHandler) ListArticles(siteID int64, limit, offset int) *dto.PaginatedResponse[*dto.Article] {
	listArticles, total, err := h.service.ListArticles(ctx.FastCtx(), siteID, limit, offset)
	if err != nil {
		return paginatedErr[*dto.Article](err)
	}

	var dtoArticles []*dto.Article
	for _, article := range listArticles {
		dtoArticles = append(dtoArticles, dto.NewArticle(article))
	}

	return paginated(dtoArticles, total, limit, offset)
}

func (h *ArticlesHandler) UpdateArticle(article *dto.Article) *dto.Response[string] {
	entity, err := article.ToEntity()
	if err != nil {
		return fail[string](err)
	}

	if err = h.service.UpdateArticle(ctx.FastCtx(), entity); err != nil {
		return fail[string](err)
	}

	return ok("Article updated successfully")
}

func (h *ArticlesHandler) DeleteArticle(id int64) *dto.Response[string] {
	if err := h.service.DeleteArticle(ctx.FastCtx(), id); err != nil {
		return fail[string](err)
	}

	return ok("Article deleted successfully")
}

func (h *ArticlesHandler) ImportFromWordPress(siteID int64, wpPostID int) *dto.Response[*dto.Article] {
	article, err := h.service.ImportFromWordPress(ctx.FastCtx(), siteID, wpPostID)
	if err != nil {
		return fail[*dto.Article](err)
	}

	return ok(dto.NewArticle(article))
}

func (h *ArticlesHandler) ImportAllFromSite(siteID int64) *dto.Response[int] {
	count, err := h.service.ImportAllFromSite(ctx.FastCtx(), siteID)
	if err != nil {
		return fail[int](err)
	}

	return ok(count)
}

func (h *ArticlesHandler) SyncFromWordPress(siteID int64) *dto.Response[string] {
	if err := h.service.SyncFromWordPress(ctx.FastCtx(), siteID); err != nil {
		return fail[string](err)
	}

	return ok("Articles synced successfully")
}

func (h *ArticlesHandler) PublishToWordPress(article *dto.Article) *dto.Response[string] {
	entity, err := article.ToEntity()
	if err != nil {
		return fail[string](err)
	}

	if err = h.service.PublishToWordPress(ctx.FastCtx(), entity); err != nil {
		return fail[string](err)
	}

	return ok("Article published successfully")
}

func (h *ArticlesHandler) UpdateInWordPress(article *dto.Article) *dto.Response[string] {
	entity, err := article.ToEntity()
	if err != nil {
		return fail[string](err)
	}

	if err = h.service.UpdateInWordPress(ctx.FastCtx(), entity); err != nil {
		return fail[string](err)
	}

	return ok("Article updated in WordPress successfully")
}

func (h *ArticlesHandler) DeleteFromWordPress(id int64) *dto.Response[string] {
	if err := h.service.DeleteFromWordPress(ctx.FastCtx(), id); err != nil {
		return fail[string](err)
	}

	return ok("Article deleted from WordPress successfully")
}

func (h *ArticlesHandler) CreateDraft(execID int64, title, content string) *dto.Response[*dto.Article] {
	exec, err := h.executionService.GetExecution(ctx.FastCtx(), execID)
	if err != nil {
		return fail[*dto.Article](err)
	}

	article, err := h.service.CreateDraft(ctx.FastCtx(), exec, title, content)
	if err != nil {
		return fail[*dto.Article](err)
	}

	return ok(dto.NewArticle(article))
}

func (h *ArticlesHandler) PublishDraft(id int64) *dto.Response[string] {
	if err := h.service.PublishDraft(ctx.FastCtx(), id); err != nil {
		return fail[string](err)
	}

	return ok("Draft published successfully")
}
