package stats

import (
	"context"
	"time"

	"github.com/davidmovas/postulator/pkg/logger"
)

var _ Recorder = (*recorder)(nil)

type recorder struct {
	repo   Repository
	logger *logger.Logger
}

func NewRecorder(repo Repository, logger *logger.Logger) Recorder {
	return &recorder{repo: repo, logger: logger}
}

func (r *recorder) RecordArticlePublished(ctx context.Context, siteID int64, wordCount int) error {
	now := time.Now()

	if err := r.repo.IncrementSiteStats(ctx, siteID, now, "articles_published", 1); err != nil {
		r.logger.ErrorWithErr(err, "Failed to record article published")
		return err
	}

	if wordCount > 0 {
		if err := r.repo.IncrementSiteStats(ctx, siteID, now, "total_words", wordCount); err != nil {
			r.logger.ErrorWithErr(err, "Failed to record word count")
			return err
		}
	}

	return nil
}

func (r *recorder) RecordArticleFailed(ctx context.Context, siteID int64) error {
	now := time.Now()

	if err := r.repo.IncrementSiteStats(ctx, siteID, now, "articles_failed", 1); err != nil {
		r.logger.ErrorWithErr(err, "Failed to record article failed")
		return err
	}

	return nil
}

func (r *recorder) RecordLinksCreated(ctx context.Context, siteID int64, internalLinks, externalLinks int) error {
	now := time.Now()

	if internalLinks > 0 {
		if err := r.repo.IncrementSiteStats(ctx, siteID, now, "internal_links_created", internalLinks); err != nil {
			r.logger.ErrorWithErr(err, "Failed to record internal links")
			return err
		}
	}

	if externalLinks > 0 {
		if err := r.repo.IncrementSiteStats(ctx, siteID, now, "external_links_created", externalLinks); err != nil {
			r.logger.ErrorWithErr(err, "Failed to record external links")
			return err
		}
	}

	return nil
}
