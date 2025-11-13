package execution

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/davidmovas/postulator/internal/domain/entities"
	"github.com/davidmovas/postulator/internal/infra/wp"
)

func (e *Executor) publishArticle(ctx context.Context, pctx *pipelineContext) (*entities.Article, error) {
	e.logger.Infof("Job %d: Publishing article to WordPress", pctx.Job.ID)

	desiredStatus := entities.StatusPublished
	wpStatus := "publish"
	if pctx.Job.RequiresValidation {
		desiredStatus = entities.StatusDraft
		wpStatus = "draft"
	}

	article := &entities.Article{
		SiteID:        pctx.Job.SiteID,
		JobID:         &pctx.Job.ID,
		TopicID:       pctx.Topic.ID,
		Title:         pctx.GeneratedTitle,
		OriginalTitle: pctx.Topic.Title,
		Content:       pctx.GeneratedContent,
		WPCategoryIDs: []int{pctx.Category.WPCategoryID},
		Status:        desiredStatus,
		Source:        entities.SourceGenerated,
		IsEdited:      false,
	}

	wordCount := len(strings.Fields(pctx.GeneratedContent))
	article.WordCount = &wordCount

	wpPostID, err := e.wpClient.CreatePost(ctx, pctx.Site, article, &wp.PostOptions{Status: wpStatus})
	if err != nil {
		return nil, fmt.Errorf("failed to create post in WordPress: %w", err)
	}

	article.WPPostID = wpPostID
	article.WPPostURL = fmt.Sprintf("%s/?p=%d", pctx.Site.URL, wpPostID)

	if err := e.articleRepo.Create(ctx, article); err != nil {
		e.logger.Errorf("Failed to create article record: %v", err)
		return nil, fmt.Errorf("failed to create article record: %w", err)
	}

	e.logger.Infof("Job %d: Article created (ID: %d, WP ID: %d, URL: %s)",
		pctx.Job.ID, article.ID, article.WPPostID, article.WPPostURL)

	return article, nil
}

func (e *Executor) createDraftArticle(ctx context.Context, pctx *pipelineContext) (*entities.Article, error) {
	e.logger.Infof("Job %d: Creating draft article for validation", pctx.Job.ID)

	article := &entities.Article{
		SiteID:        pctx.Job.SiteID,
		JobID:         &pctx.Job.ID,
		TopicID:       pctx.Topic.ID,
		Title:         pctx.GeneratedTitle,
		OriginalTitle: pctx.Topic.Title,
		Content:       pctx.GeneratedContent,
		WPCategoryIDs: []int{pctx.Category.WPCategoryID},
		Status:        entities.StatusDraft,
		Source:        entities.SourceGenerated,
		IsEdited:      false,
	}

	wordCount := len(strings.Fields(pctx.GeneratedContent))
	article.WordCount = &wordCount

	if err := e.articleRepo.Create(ctx, article); err != nil {
		return nil, fmt.Errorf("failed to create draft article: %w", err)
	}

	e.logger.Infof("Job %d: Draft article created (ID: %d)", pctx.Job.ID, article.ID)
	return article, nil
}

func (e *Executor) publishValidatedArticle(ctx context.Context, exec *entities.Execution) error {
	e.logger.Infof("Publishing validated article for execution %d", exec.ID)

	article, err := e.articleRepo.GetByID(ctx, *exec.ArticleID)
	if err != nil {
		return fmt.Errorf("failed to get article: %w", err)
	}

	site, err := e.siteService.GetSiteWithPassword(ctx, exec.SiteID)
	if err != nil {
		return fmt.Errorf("failed to get site: %w", err)
	}

	exec.Status = entities.ExecutionStatusPublishing
	if err := e.execRepo.Update(ctx, exec); err != nil {
		e.logger.Warnf("Failed to update execution status: %v", err)
	}

	// If the article already exists as a draft on WP, promote it; otherwise create as publish
	if article.WPPostID > 0 {
		article.Status = entities.StatusPublished
		if err := e.wpClient.UpdatePost(ctx, site, article); err != nil {
			exec.Status = entities.ExecutionStatusFailed
			errMsg := err.Error()
			exec.ErrorMessage = &errMsg
			_ = e.execRepo.Update(ctx, exec)
			return fmt.Errorf("failed to publish draft in WordPress: %w", err)
		}
	} else {
		wpPostID, err := e.wpClient.CreatePost(ctx, site, article, &wp.PostOptions{Status: "publish"})
		if err != nil {
			exec.Status = entities.ExecutionStatusFailed
			errMsg := err.Error()
			exec.ErrorMessage = &errMsg
			_ = e.execRepo.Update(ctx, exec)
			return fmt.Errorf("failed to publish to WordPress: %w", err)
		}
		article.WPPostID = wpPostID
		article.WPPostURL = fmt.Sprintf("%s/?p=%d", site.URL, wpPostID)
	}
	article.Status = entities.StatusPublished

	if err := e.articleRepo.Update(ctx, article); err != nil {
		e.logger.Errorf("Failed to update article: %v", err)
	}

	now := time.Now()
	exec.Status = entities.ExecutionStatusPublished
	exec.PublishedAt = &now
	exec.CompletedAt = &now

	if err := e.execRepo.Update(ctx, exec); err != nil {
		e.logger.Warnf("Failed to update execution: %v", err)
	}

	e.logger.Infof("Validated article published successfully (Article ID: %d, WP ID: %d)",
		article.ID, article.WPPostID)

	return nil
}
