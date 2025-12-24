package generation

import (
	"context"
	"fmt"
	"time"

	"github.com/davidmovas/postulator/internal/domain/articles"
	"github.com/davidmovas/postulator/internal/domain/entities"
	"github.com/davidmovas/postulator/internal/domain/sitemap"
	"github.com/davidmovas/postulator/internal/domain/sites"
	"github.com/davidmovas/postulator/internal/infra/wp"
	"github.com/davidmovas/postulator/pkg/logger"
)

type PublishResult struct {
	ArticleID int64
	WPPageID  int
	WPURL     string
}

type Publisher struct {
	sitemapSvc sitemap.Service
	articleSvc articles.Service
	siteSvc    sites.Service
	wpClient   wp.Client
	logger     *logger.Logger
}

func NewPublisher(
	sitemapSvc sitemap.Service,
	articleSvc articles.Service,
	siteSvc sites.Service,
	wpClient wp.Client,
	logger *logger.Logger,
) *Publisher {
	return &Publisher{
		sitemapSvc: sitemapSvc,
		articleSvc: articleSvc,
		siteSvc:    siteSvc,
		wpClient:   wpClient,
		logger:     logger.WithScope("page_publisher"),
	}
}

type PublishRequest struct {
	Node           *entities.SitemapNode
	Content        *PageContent
	SiteID         int64
	PublishAs      PublishAs
	ParentWPPageID *int
}

func (p *Publisher) Publish(ctx context.Context, req PublishRequest) (*PublishResult, error) {
	site, err := p.siteSvc.GetSiteWithPassword(ctx, req.SiteID)
	if err != nil {
		return nil, fmt.Errorf("failed to get site: %w", err)
	}

	wpStatus := mapPublishAsToWPStatus(req.PublishAs)

	wpPage := &wp.WPPage{
		Title:   req.Content.Title,
		Slug:    req.Node.Slug,
		Content: req.Content.Content,
		Excerpt: req.Content.Excerpt,
		Status:  wpStatus,
	}

	if req.ParentWPPageID != nil {
		wpPage.ParentID = *req.ParentWPPageID
	}

	p.logger.Debugf("Creating WP page for node %d: %s", req.Node.ID, req.Node.Title)

	wpPageID, err := p.wpClient.CreatePage(ctx, site, wpPage, &wp.PageCreateOptions{
		Status: wpStatus,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create WP page: %w", err)
	}

	createdPage, err := p.wpClient.GetPage(ctx, site, wpPageID)
	if err != nil {
		p.logger.ErrorWithErr(err, fmt.Sprintf("Failed to get created page %d", wpPageID))
	}

	wpURL := ""
	if createdPage != nil {
		wpURL = createdPage.Link
		wpPage.Slug = createdPage.Slug
	}

	now := time.Now()
	var publishedAt *time.Time
	if wpStatus == "publish" {
		publishedAt = &now
	}

	excerpt := req.Content.Excerpt
	metaDesc := req.Content.MetaDescription
	wordCount := countWords(req.Content.Content)

	article := &entities.Article{
		SiteID:          req.SiteID,
		Title:           req.Content.Title,
		OriginalTitle:   req.Node.Title,
		Content:         req.Content.Content,
		Excerpt:         &excerpt,
		MetaDescription: &metaDesc,
		ContentType:     entities.ContentTypePage,
		WPPageID:        &wpPageID,
		WPPostURL:       wpURL,
		Status:          mapWPStatusToArticleStatus(wpStatus),
		WordCount:       &wordCount,
		Source:          entities.SourceGenerated,
		Slug:            &wpPage.Slug,
		SitemapNodeID:   &req.Node.ID,
		CreatedAt:       now,
		UpdatedAt:       now,
		PublishedAt:     publishedAt,
	}

	if req.ParentWPPageID != nil {
		article.ParentPageID = req.ParentWPPageID
	}

	if err := p.articleSvc.CreateArticle(ctx, article); err != nil {
		p.logger.ErrorWithErr(err, "Failed to save article record, but WP page was created")
	}

	if err := p.updateNodeAfterPublish(ctx, req.Node, article.ID, wpPageID, wpURL, wpStatus); err != nil {
		p.logger.ErrorWithErr(err, "Failed to update node after publish")
	}

	p.logger.Infof("Published page for node %d: WP ID=%d, URL=%s", req.Node.ID, wpPageID, wpURL)

	return &PublishResult{
		ArticleID: article.ID,
		WPPageID:  wpPageID,
		WPURL:     wpURL,
	}, nil
}

func (p *Publisher) updateNodeAfterPublish(
	ctx context.Context,
	node *entities.SitemapNode,
	articleID int64,
	wpPageID int,
	wpURL string,
	wpStatus string,
) error {
	node.ArticleID = &articleID
	node.WPPageID = &wpPageID
	node.WPURL = &wpURL
	node.WPTitle = &node.Title
	node.WPSlug = &node.Slug
	node.ContentType = entities.NodeContentTypePage // Set content type for publish status changes
	node.GenerationStatus = entities.GenStatusGenerated
	node.PublishStatus = mapWPStatusToPublishStatus(wpStatus)
	node.IsSynced = true
	now := time.Now()
	node.LastSyncedAt = &now

	return p.sitemapSvc.UpdateNode(ctx, node)
}

func mapPublishAsToWPStatus(publishAs PublishAs) string {
	switch publishAs {
	case PublishAsPublish:
		return "publish"
	case PublishAsDraft:
		return "draft"
	case PublishAsPending:
		return "pending"
	default:
		return "draft"
	}
}

func mapWPStatusToArticleStatus(wpStatus string) entities.ArticleStatus {
	switch wpStatus {
	case "publish":
		return entities.StatusPublished
	case "draft":
		return entities.StatusDraft
	case "pending":
		return entities.StatusPending
	case "private":
		return entities.StatusPrivate
	default:
		return entities.StatusDraft
	}
}

func mapWPStatusToPublishStatus(wpStatus string) entities.NodePublishStatus {
	switch wpStatus {
	case "publish":
		return entities.PubStatusPublished
	case "draft":
		return entities.PubStatusDraft
	case "pending":
		return entities.PubStatusPending
	default:
		return entities.PubStatusDraft
	}
}

func countWords(content string) int {
	count := 0
	inWord := false
	for _, r := range content {
		if r == ' ' || r == '\n' || r == '\t' || r == '<' || r == '>' {
			inWord = false
		} else if !inWord {
			inWord = true
			count++
		}
	}
	return count
}
