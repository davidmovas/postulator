package articles

import (
	"context"
	"strings"
	"time"

	"github.com/davidmovas/postulator/internal/domain/entities"
	"github.com/davidmovas/postulator/internal/domain/jobs/execution"
	"github.com/davidmovas/postulator/internal/domain/sites"
	"github.com/davidmovas/postulator/internal/domain/stats"
	"github.com/davidmovas/postulator/internal/infra/wp"
	"github.com/davidmovas/postulator/pkg/errors"
	"github.com/davidmovas/postulator/pkg/logger"
)

var _ Service = (*service)(nil)

type service struct {
	repo         Repository
	siteService  sites.Service
	wp           wp.Client
	statsService stats.Service
	logger       *logger.Logger
}

func NewService(
	repo Repository,
	siteService sites.Service,
	wp wp.Client,
	statsService stats.Service,
	logger *logger.Logger,
) Service {
	return &service{
		repo:         repo,
		siteService:  siteService,
		wp:           wp,
		statsService: statsService,
		logger:       logger.WithScope("service").WithScope("articles"),
	}
}

func (s *service) CreateArticle(ctx context.Context, article *entities.Article) error {
	if err := s.validateArticle(article); err != nil {
		return err
	}

	now := time.Now()
	article.CreatedAt = now
	article.UpdatedAt = now

	if article.WordCount == nil {
		count := s.calculateWordCount(article.Content)
		article.WordCount = &count
	}

	if err := s.repo.Create(ctx, article); err != nil {
		s.logger.ErrorWithErr(err, "Failed to create article")
		return err
	}

	s.logger.Info("Article created successfully")
	return nil
}

func (s *service) GetArticle(ctx context.Context, id int64) (*entities.Article, error) {
	article, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get article")
		return nil, err
	}

	return article, nil
}

func (s *service) ListArticles(ctx context.Context, siteID int64, limit, offset int) ([]*entities.Article, int, error) {
	articles, err := s.repo.ListBySite(ctx, siteID, limit, offset)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to list articles")
		return nil, 0, err
	}

	total, err := s.repo.CountBySite(ctx, siteID)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to count articles")
		return nil, 0, err
	}

	return articles, total, nil
}

func (s *service) UpdateArticle(ctx context.Context, article *entities.Article) error {
	if err := s.validateArticle(article); err != nil {
		return err
	}

	article.UpdatedAt = time.Now()
	article.IsEdited = true

	if err := s.repo.Update(ctx, article); err != nil {
		s.logger.ErrorWithErr(err, "Failed to update article")
		return err
	}

	s.logger.Info("Article updated successfully")
	return nil
}

func (s *service) DeleteArticle(ctx context.Context, id int64) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		s.logger.ErrorWithErr(err, "Failed to delete article")
		return err
	}

	s.logger.Info("Article deleted successfully")
	return nil
}

func (s *service) ImportFromWordPress(ctx context.Context, siteID int64, wpPostID int) (*entities.Article, error) {
	site, err := s.siteService.GetSiteWithPassword(ctx, siteID)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get site for import")
		return nil, err
	}

	existingArticle, err := s.repo.GetByWPPostID(ctx, siteID, wpPostID)
	if err == nil && existingArticle != nil {
		return existingArticle, nil
	}

	wpPost, err := s.wp.GetPost(ctx, site, wpPostID)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get post from WordPress")
		return nil, err
	}

	article := wpPost
	article.SiteID = siteID

	if err = s.repo.Create(ctx, article); err != nil {
		s.logger.ErrorWithErr(err, "Failed to create imported article")
		return nil, err
	}

	s.logger.Info("Article imported from WordPress successfully")
	return article, nil
}

func (s *service) ImportAllFromSite(ctx context.Context, siteID int64) (int, error) {
	site, err := s.siteService.GetSiteWithPassword(ctx, siteID)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get site for bulk import")
		return 0, err
	}

	wpPosts, err := s.wp.GetPosts(ctx, site)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get posts from WordPress")
		return 0, err
	}

	var importedCount int
	for _, wpPost := range wpPosts {
		existingArticle, _ := s.repo.GetByWPPostID(ctx, siteID, wpPost.WPPostID)
		if existingArticle != nil {
			continue
		}

		article := wpPost
		article.SiteID = siteID

		if err = s.repo.Create(ctx, article); err != nil {
			s.logger.ErrorWithErr(err, "Failed to create imported article")
			continue
		}

		importedCount++
	}

	s.logger.Infof("Bulk import completed, imported %d articles", importedCount)
	return importedCount, nil
}

func (s *service) SyncFromWordPress(ctx context.Context, siteID int64) error {
	site, err := s.siteService.GetSiteWithPassword(ctx, siteID)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get site for sync")
		return err
	}

	localArticles, err := s.repo.ListBySite(ctx, siteID, 10000, 0)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get local articles")
		return err
	}

	remoteArticles, err := s.wp.GetPosts(ctx, site)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get articles from WordPress")
		return err
	}

	localMap := make(map[int]*entities.Article)
	for _, article := range localArticles {
		localMap[article.WPPostID] = article
	}

	remoteMap := make(map[int]*entities.Article)
	for _, article := range remoteArticles {
		remoteMap[article.WPPostID] = article
	}

	// Статьи для создания (есть в удаленке, но нет в локальной)
	var articlesToCreate []*entities.Article
	for wpID, remoteArticle := range remoteMap {
		if _, exists := localMap[wpID]; !exists {
			remoteArticle.SiteID = siteID
			articlesToCreate = append(articlesToCreate, remoteArticle)
		}
	}

	// Статьи для удаления (есть в локальной, но нет в удаленке)
	var articlesToDelete []int64
	for wpID, localArticle := range localMap {
		if _, exists := remoteMap[wpID]; !exists {
			articlesToDelete = append(articlesToDelete, localArticle.ID)
		}
	}

	if len(articlesToCreate) > 0 {
		if err = s.repo.BulkCreate(ctx, articlesToCreate); err != nil {
			s.logger.ErrorWithErr(err, "Failed to bulk create articles")
			return err
		}
		s.logger.Infof("Articles created during sync %d", len(articlesToCreate))
	}

	if len(articlesToDelete) > 0 {
		for _, articleID := range articlesToDelete {
			if err = s.repo.Delete(ctx, articleID); err != nil {
				s.logger.ErrorWithErr(err, "Failed to delete article during sync")
				return err
			}
		}
		s.logger.Infof("Articles deleted during sync %d", len(articlesToDelete))
	}

	// Обновляем существующие статьи (синхронизируем контент)
	var articlesToUpdate []*entities.Article
	for wpID, localArticle := range localMap {
		now := time.Now()
		if remoteArticle, exists := remoteMap[wpID]; exists {
			// Проверяем, нужно ли обновить статью
			if s.needUpdate(localArticle, remoteArticle) {
				localArticle.Title = remoteArticle.Title
				localArticle.Content = remoteArticle.Content
				localArticle.Excerpt = remoteArticle.Excerpt
				localArticle.WPCategoryIDs = remoteArticle.WPCategoryIDs
				localArticle.UpdatedAt = now
				localArticle.LastSyncedAt = &now
				articlesToUpdate = append(articlesToUpdate, localArticle)
			}
		}
	}

	for _, article := range articlesToUpdate {
		if err = s.repo.Update(ctx, article); err != nil {
			s.logger.ErrorWithErr(err, "Failed to update article during sync")
			return err
		}
	}

	if len(articlesToUpdate) > 0 {
		s.logger.Infof("Articles updated during sync %d", len(articlesToUpdate))
	}

	s.logger.Info("Articles sync completed successfully")
	return nil
}

func (s *service) PublishToWordPress(ctx context.Context, article *entities.Article) error {
	site, err := s.siteService.GetSiteWithPassword(ctx, article.SiteID)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get site for publishing")
		return err
	}

	wpPostID, err := s.wp.CreatePost(ctx, site, article)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to publish article to WordPress")
		return err
	}

	now := time.Now()
	article.WPPostID = wpPostID
	article.Status = entities.StatusPublished
	article.PublishedAt = &now
	article.UpdatedAt = now

	if err = s.repo.Update(ctx, article); err != nil {
		s.logger.ErrorWithErr(err, "Failed to update article after publishing")
		return err
	}

	if err = s.statsService.RecordArticlePublished(ctx, article.SiteID, *article.WordCount); err != nil {
		s.logger.ErrorWithErr(err, "Failed to record article statistics")
	}

	s.logger.Info("Article published to WordPress successfully")
	return nil
}

func (s *service) UpdateInWordPress(ctx context.Context, article *entities.Article) error {
	site, err := s.siteService.GetSiteWithPassword(ctx, article.SiteID)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get site for update")
		return err
	}

	if err = s.wp.UpdatePost(ctx, site, article); err != nil {
		s.logger.ErrorWithErr(err, "Failed to update article in WordPress")
		return err
	}

	article.UpdatedAt = time.Now()
	article.LastSyncedAt = &article.UpdatedAt

	if err = s.repo.Update(ctx, article); err != nil {
		s.logger.ErrorWithErr(err, "Failed to update article after WordPress update")
		return err
	}

	s.logger.Info("Article updated in WordPress successfully")
	return nil
}

func (s *service) DeleteFromWordPress(ctx context.Context, id int64) error {
	article, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get article for deletion")
		return err
	}

	site, err := s.siteService.GetSiteWithPassword(ctx, article.SiteID)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get site for deletion")
		return err
	}

	if err = s.wp.DeletePost(ctx, site, article.WPPostID); err != nil {
		s.logger.ErrorWithErr(err, "Failed to delete article from WordPress")
		return err
	}

	article.Status = entities.StatusDraft
	article.WPPostID = 0
	article.WPPostURL = ""
	article.PublishedAt = nil
	article.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, article); err != nil {
		s.logger.ErrorWithErr(err, "Failed to update article after WordPress deletion")
		return err
	}

	s.logger.Info("Article deleted from WordPress successfully")
	return nil
}

func (s *service) CreateDraft(ctx context.Context, exec *execution.Execution, title, content string) (*entities.Article, error) {
	wordCount := s.calculateWordCount(content)

	article := &entities.Article{
		JobID:         &exec.JobID,
		SiteID:        exec.SiteID,
		TopicID:       exec.TopicID,
		Title:         title,
		OriginalTitle: title,
		Content:       content,
		Status:        entities.StatusDraft,
		Source:        entities.SourceGenerated,
		WordCount:     &wordCount,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := s.repo.Create(ctx, article); err != nil {
		s.logger.ErrorWithErr(err, "Failed to create draft article")
		return nil, err
	}

	s.logger.Info("Draft article created successfully")
	return article, nil
}

func (s *service) PublishDraft(ctx context.Context, id int64) error {
	article, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get draft article")
		return err
	}

	if article.Status != entities.StatusDraft {
		return errors.Validation("Article is not a draft")
	}

	return s.PublishToWordPress(ctx, article)
}

func (s *service) validateArticle(article *entities.Article) error {
	if article.SiteID <= 0 {
		return errors.Validation("Site ID is required")
	}

	if strings.TrimSpace(article.Title) == "" {
		return errors.Validation("Article title is required")
	}

	if strings.TrimSpace(article.Content) == "" {
		return errors.Validation("Article content is required")
	}

	if article.TopicID <= 0 {
		return errors.Validation("Topic ID is required")
	}

	return nil
}

func (s *service) calculateWordCount(content string) int {
	words := strings.Fields(content)
	return len(words)
}

func (s *service) needUpdate(local, remote *entities.Article) bool {
	return (local.Title != remote.Title) && (local.WPPostID != remote.WPPostID)
}
