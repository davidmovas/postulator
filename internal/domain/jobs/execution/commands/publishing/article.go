package publishing

import (
	"fmt"
	"strings"
	"time"

	"github.com/davidmovas/postulator/internal/domain/articles"
	"github.com/davidmovas/postulator/internal/domain/entities"
	"github.com/davidmovas/postulator/internal/domain/jobs/execution/commands"
	"github.com/davidmovas/postulator/internal/domain/jobs/execution/fault"
	"github.com/davidmovas/postulator/internal/domain/jobs/execution/pipeline"
	"github.com/davidmovas/postulator/internal/domain/jobs/execution/pipevents"
	"github.com/davidmovas/postulator/internal/domain/stats"
	"github.com/davidmovas/postulator/internal/infra/events"
	"github.com/davidmovas/postulator/internal/infra/wp"
)

type PublishArticleCommand struct {
	*commands.BaseCommand
	wpClient          wp.Client
	executionProvider commands.ExecutionProvider
	articleRepo       articles.Repository
	statsRecorder     stats.Recorder
}

func NewPublishArticleCommand(
	executionProvider commands.ExecutionProvider,
	articleRepo articles.Repository,
	wpClient wp.Client,
	statsRecorder stats.Recorder,
) *PublishArticleCommand {
	return &PublishArticleCommand{
		BaseCommand: commands.NewBaseCommand(
			"publish_article",
			pipeline.StateOutputValidated,
			pipeline.StatePublished,
		),
		wpClient:          wpClient,
		executionProvider: executionProvider,
		articleRepo:       articleRepo,
		statsRecorder:     statsRecorder,
	}
}

func (c *PublishArticleCommand) Execute(ctx *pipeline.Context) error {
	if !ctx.HasExecution() {
		return fault.NewFatalError(fault.ErrCodeInvalidJob, c.Name(), "execution not created")
	}

	if !ctx.HasGeneration() {
		return fault.NewFatalError(fault.ErrCodeInvalidJob, c.Name(), "content not generated")
	}

	if !ctx.HasSelection() {
		return fault.NewFatalError(fault.ErrCodeInvalidJob, c.Name(), "topic or category not selected")
	}

	ctx.Execution.Execution.Status = entities.ExecutionStatusPublishing
	if err := c.executionProvider.Update(ctx.Context(), ctx.Execution.Execution); err != nil {
		return fault.WrapError(err, fault.ErrCodeDatabaseError, c.Name(), "failed to update execution status")
	}

	desiredStatus := entities.StatusPublished
	wpStatus := "publish"
	if ctx.Job.RequiresValidation {
		desiredStatus = entities.StatusDraft
		wpStatus = "draft"
	}

	var categoryIDs []int
	for _, cat := range ctx.Selection.Categories {
		categoryIDs = append(categoryIDs, int(cat.ID))
	}

	article := &entities.Article{
		SiteID:        ctx.Job.SiteID,
		JobID:         &ctx.Job.ID,
		TopicID:       ctx.Selection.VariationTopic.ID,
		Title:         ctx.Generation.GeneratedTitle,
		Excerpt:       &ctx.Generation.GeneratedExcerpt,
		OriginalTitle: ctx.Selection.OriginalTopic.Title,
		Content:       ctx.Generation.GeneratedContent,
		WPCategoryIDs: categoryIDs,
		Status:        desiredStatus,
		Source:        entities.SourceGenerated,
		IsEdited:      false,
	}

	wordCount := len(strings.Fields(ctx.Generation.GeneratedContent))
	article.WordCount = &wordCount

	wpPostID, err := c.wpClient.CreatePost(ctx.Context(), ctx.Validated.Site, article, &wp.PostOptions{Status: wpStatus})
	if err != nil {
		_ = c.statsRecorder.RecordArticleFailed(ctx.Context(), ctx.Job.SiteID)
		return fault.WrapError(err, fault.ErrCodePublishFailed, c.Name(), "failed to create post in WordPress")
	}

	article.WPPostID = wpPostID
	article.WPPostURL = fmt.Sprintf("%s/?p=%d", ctx.Validated.Site.URL, wpPostID)

	if err = c.articleRepo.Create(ctx.Context(), article); err != nil {
		_ = c.statsRecorder.RecordArticleFailed(ctx.Context(), ctx.Job.SiteID)
		return fault.WrapError(err, fault.ErrCodeArticleSaveError, c.Name(), "failed to create article record")
	}

	ctx.InitPublicationPhase(article)
	ctx.Execution.Execution.ArticleID = &article.ID

	if ctx.Job.RequiresValidation {
		ctx.Execution.Execution.Status = entities.ExecutionStatusPendingValidation
		if err = c.executionProvider.Update(ctx.Context(), ctx.Execution.Execution); err != nil {
			return fault.WrapError(err, fault.ErrCodeDatabaseError, c.Name(), "failed to update execution to pending validation")
		}
		return nil
	}

	now := time.Now()
	ctx.Execution.Execution.PublishedAt = &now
	ctx.Execution.Execution.Status = entities.ExecutionStatusPublished

	if err = c.executionProvider.Update(ctx.Context(), ctx.Execution.Execution); err != nil {
		_ = c.statsRecorder.RecordArticleFailed(ctx.Context(), ctx.Job.SiteID)
		return fault.WrapError(err, fault.ErrCodeDatabaseError, c.Name(), "failed to update execution after publication")
	}

	events.Publish(ctx.Context(), events.NewEvent(
		pipevents.EventArticlePublished,
		&pipevents.ArticlePublishedEvent{
			JobID:     ctx.Job.ID,
			ArticleID: article.ID,
			SiteID:    ctx.Job.SiteID,
			Title:     article.Title,
			WPPostID:  article.WPPostID,
			WPPostURL: article.WPPostURL,
			Status:    string(article.Status),
		},
	))

	return nil
}

func (c *PublishArticleCommand) NextState() pipeline.State {
	return pipeline.StatePublished
}
