package handlers

import (
	"github.com/davidmovas/postulator/internal/domain/articles"
	"github.com/davidmovas/postulator/internal/domain/entities"
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

func (h *ArticlesHandler) CreateAndPublishArticle(article *dto.Article) *dto.Response[*dto.Article] {
	entity, err := article.ToEntity()
	if err != nil {
		return fail[*dto.Article](err)
	}

	result, err := h.service.CreateAndPublishArticle(ctx.LongCtx(), entity)
	if err != nil {
		return fail[*dto.Article](err)
	}

	return ok(dto.NewArticle(result))
}

func (h *ArticlesHandler) UpdateAndSyncArticle(article *dto.Article) *dto.Response[*dto.Article] {
	entity, err := article.ToEntity()
	if err != nil {
		return fail[*dto.Article](err)
	}

	result, err := h.service.UpdateAndSyncArticle(ctx.LongCtx(), entity)
	if err != nil {
		return fail[*dto.Article](err)
	}

	return ok(dto.NewArticle(result))
}

func (h *ArticlesHandler) GetArticle(id int64) *dto.Response[*dto.Article] {
	article, err := h.service.GetArticle(ctx.FastCtx(), id)
	if err != nil {
		return fail[*dto.Article](err)
	}

	return ok(dto.NewArticle(article))
}

func (h *ArticlesHandler) ListArticles(filter *dto.ArticleListFilter) *dto.Response[*dto.ArticleListResult] {
	var status *entities.ArticleStatus
	if filter.Status != nil {
		s := entities.ArticleStatus(*filter.Status)
		status = &s
	}

	var source *entities.Source
	if filter.Source != nil {
		s := entities.Source(*filter.Source)
		source = &s
	}

	domainFilter := &articles.ListFilter{
		SiteID:     filter.SiteID,
		Status:     status,
		Source:     source,
		CategoryID: filter.CategoryID,
		Search:     filter.Search,
		SortBy:     filter.SortBy,
		SortOrder:  filter.SortOrder,
		Limit:      filter.Limit,
		Offset:     filter.Offset,
	}

	result, err := h.service.ListArticles(ctx.FastCtx(), domainFilter)
	if err != nil {
		return fail[*dto.ArticleListResult](err)
	}

	var dtoArticles []*dto.Article
	for _, article := range result.Articles {
		dtoArticles = append(dtoArticles, dto.NewArticle(article))
	}

	return ok(&dto.ArticleListResult{
		Articles: dtoArticles,
		Total:    result.Total,
	})
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

func (h *ArticlesHandler) BulkDeleteArticles(ids []int64) *dto.Response[string] {
	if err := h.service.BulkDeleteArticles(ctx.FastCtx(), ids); err != nil {
		return fail[string](err)
	}

	return ok("Articles deleted successfully")
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

func (h *ArticlesHandler) PublishToWordPress(id int64) *dto.Response[string] {
	article, err := h.service.GetArticle(ctx.FastCtx(), id)
	if err != nil {
		return fail[string](err)
	}

	if err = h.service.PublishToWordPress(ctx.FastCtx(), article); err != nil {
		return fail[string](err)
	}

	return ok("Article published successfully")
}

func (h *ArticlesHandler) BulkPublishToWordPress(ids []int64) *dto.Response[int] {
	count, err := h.service.BulkPublishToWordPress(ctx.FastCtx(), ids)
	if err != nil {
		return fail[int](err)
	}

	return ok(count)
}

func (h *ArticlesHandler) UpdateInWordPress(id int64) *dto.Response[string] {
	article, err := h.service.GetArticle(ctx.FastCtx(), id)
	if err != nil {
		return fail[string](err)
	}

	if err = h.service.UpdateInWordPress(ctx.FastCtx(), article); err != nil {
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

func (h *ArticlesHandler) BulkDeleteFromWordPress(ids []int64) *dto.Response[int] {
	count, err := h.service.BulkDeleteFromWordPress(ctx.FastCtx(), ids)
	if err != nil {
		return fail[int](err)
	}

	return ok(count)
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

func (h *ArticlesHandler) GenerateContent(input *dto.GenerateContentInput) *dto.Response[*dto.GenerateContentResult] {
	domainInput := &articles.GenerateContentInput{
		SiteID:            input.SiteID,
		ProviderID:        input.ProviderID,
		PromptID:          input.PromptID,
		TopicID:           input.TopicID,
		CustomTopicTitle:  input.CustomTopicTitle,
		PlaceholderValues: input.PlaceholderValues,
		UseWebSearch:      input.UseWebSearch,
	}

	result, err := h.service.GenerateContent(ctx.LongCtx(), domainInput)
	if err != nil {
		return fail[*dto.GenerateContentResult](err)
	}

	return ok(&dto.GenerateContentResult{
		Title:           result.Title,
		Content:         result.Content,
		Excerpt:         result.Excerpt,
		MetaDescription: result.MetaDescription,
		TopicID:         result.TopicID,
	})
}
