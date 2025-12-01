package articles

import (
	"context"
	"strings"
	"time"

	"github.com/davidmovas/postulator/internal/domain/entities"
	"github.com/davidmovas/postulator/internal/domain/prompts"
	"github.com/davidmovas/postulator/internal/domain/providers"
	"github.com/davidmovas/postulator/internal/domain/sites"
	"github.com/davidmovas/postulator/internal/domain/topics"
	"github.com/davidmovas/postulator/internal/infra/ai"
	"github.com/davidmovas/postulator/internal/infra/wp"
	"github.com/davidmovas/postulator/pkg/errors"
	"github.com/davidmovas/postulator/pkg/logger"
)

var _ Service = (*service)(nil)

type service struct {
	wp              wp.Client
	repo            Repository
	siteService     sites.Service
	providerService providers.Service
	promptService   prompts.Service
	topicService    topics.Service
	logger          *logger.Logger
}

func NewService(
	repo Repository,
	siteService sites.Service,
	providerService providers.Service,
	promptService prompts.Service,
	topicService topics.Service,
	wp wp.Client,
	logger *logger.Logger,
) Service {
	return &service{
		repo:            repo,
		siteService:     siteService,
		providerService: providerService,
		promptService:   promptService,
		topicService:    topicService,
		wp:              wp,
		logger:          logger.WithScope("service").WithScope("articles"),
	}
}

func (s *service) CreateArticle(ctx context.Context, article *entities.Article) error {
	if err := s.validateArticle(article); err != nil {
		return err
	}

	now := time.Now()
	// Preserve CreatedAt if already set (e.g., from WordPress import)
	if article.CreatedAt.IsZero() {
		article.CreatedAt = now
	}
	article.UpdatedAt = now

	if article.WordCount == nil {
		count := s.calculateWordCount(article.Content)
		article.WordCount = &count
	}

	if article.Source == "" {
		article.Source = entities.SourceManual
	}

	if article.Status == "" {
		article.Status = entities.StatusDraft
	}

	if err := s.repo.Create(ctx, article); err != nil {
		s.logger.ErrorWithErr(err, "Failed to create article")
		return err
	}

	s.logger.Info("Article created successfully")
	return nil
}

func (s *service) CreateAndPublishArticle(ctx context.Context, article *entities.Article) (*entities.Article, error) {
	// First create the article locally
	if err := s.CreateArticle(ctx, article); err != nil {
		return nil, err
	}

	// Then publish to WordPress
	if err := s.PublishToWordPress(ctx, article); err != nil {
		s.logger.ErrorWithErr(err, "Article created locally but failed to publish to WordPress")
		return nil, errors.WordPress("publish", err)
	}

	// Fetch the updated article with WP info
	updatedArticle, err := s.repo.GetByID(ctx, article.ID)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get updated article after publish")
		return article, nil // Return original article if we can't fetch the updated one
	}

	s.logger.Info("Article created and published successfully")
	return updatedArticle, nil
}

func (s *service) UpdateAndSyncArticle(ctx context.Context, article *entities.Article) (*entities.Article, error) {
	// First update the article locally
	if err := s.UpdateArticle(ctx, article); err != nil {
		return nil, err
	}

	// Check if the article is already published to WordPress
	existingArticle, err := s.repo.GetByID(ctx, article.ID)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get article for sync")
		return nil, err
	}

	// If already published, sync to WordPress; otherwise publish
	if existingArticle.WPPostID > 0 {
		if err := s.UpdateInWordPress(ctx, existingArticle); err != nil {
			s.logger.ErrorWithErr(err, "Article updated locally but failed to sync to WordPress")
			return nil, errors.WordPress("sync", err)
		}
	} else {
		if err := s.PublishToWordPress(ctx, existingArticle); err != nil {
			s.logger.ErrorWithErr(err, "Article updated locally but failed to publish to WordPress")
			return nil, errors.WordPress("publish", err)
		}
	}

	// Fetch the updated article with WP info
	updatedArticle, err := s.repo.GetByID(ctx, article.ID)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get updated article after sync")
		return existingArticle, nil
	}

	s.logger.Info("Article updated and synced successfully")
	return updatedArticle, nil
}

func (s *service) GetArticle(ctx context.Context, id int64) (*entities.Article, error) {
	article, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get article")
		return nil, err
	}

	return article, nil
}

func (s *service) ListArticles(ctx context.Context, filter *ListFilter) (*ListResult, error) {
	if filter.Limit <= 0 {
		filter.Limit = 25
	}

	result, err := s.repo.List(ctx, filter)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to list articles")
		return nil, err
	}

	return result, nil
}

func (s *service) UpdateArticle(ctx context.Context, article *entities.Article) error {
	if err := s.validateArticle(article); err != nil {
		return err
	}

	existingArticle, err := s.repo.GetByID(ctx, article.ID)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get article for update")
		return err
	}

	// Preserve original title if not set
	if article.OriginalTitle == "" {
		article.OriginalTitle = existingArticle.OriginalTitle
	}

	// Preserve WordPress fields if not set (critical for update vs create logic)
	if article.WPPostID == 0 {
		article.WPPostID = existingArticle.WPPostID
	}
	if article.WPPostURL == "" {
		article.WPPostURL = existingArticle.WPPostURL
	}
	if len(article.WPCategoryIDs) == 0 {
		article.WPCategoryIDs = existingArticle.WPCategoryIDs
	}
	if len(article.WPTagIDs) == 0 {
		article.WPTagIDs = existingArticle.WPTagIDs
	}
	if article.Source == "" {
		article.Source = existingArticle.Source
	}
	if article.Slug == nil && existingArticle.Slug != nil {
		article.Slug = existingArticle.Slug
	}
	if article.Author == nil && existingArticle.Author != nil {
		article.Author = existingArticle.Author
	}
	// Note: FeaturedMediaID and FeaturedMediaURL are NOT preserved from existing article
	// This allows users to clear the featured image by setting it to nil/empty
	if article.PublishedAt == nil && existingArticle.PublishedAt != nil {
		article.PublishedAt = existingArticle.PublishedAt
	}
	if article.LastSyncedAt == nil && existingArticle.LastSyncedAt != nil {
		article.LastSyncedAt = existingArticle.LastSyncedAt
	}
	// Preserve CreatedAt - it should never change
	article.CreatedAt = existingArticle.CreatedAt

	// Mark as edited if content changed
	if article.Title != existingArticle.Title || article.Content != existingArticle.Content {
		article.IsEdited = true
	}

	// Recalculate word count
	count := s.calculateWordCount(article.Content)
	article.WordCount = &count

	article.UpdatedAt = time.Now()

	if err = s.repo.Update(ctx, article); err != nil {
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

func (s *service) BulkDeleteArticles(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}

	if err := s.repo.BulkDelete(ctx, ids); err != nil {
		s.logger.ErrorWithErr(err, "Failed to bulk delete articles")
		return err
	}

	s.logger.Infof("Bulk deleted %d articles", len(ids))
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

	// Articles to create (exist in remote but not in local)
	var articlesToCreate []*entities.Article
	for wpID, remoteArticle := range remoteMap {
		if _, exists := localMap[wpID]; !exists {
			remoteArticle.SiteID = siteID
			articlesToCreate = append(articlesToCreate, remoteArticle)
		}
	}

	// Articles to delete (exist in local but not in remote)
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

	// Update existing articles (sync content)
	var articlesToUpdate []*entities.Article
	for wpID, localArticle := range localMap {
		now := time.Now()
		if remoteArticle, exists := remoteMap[wpID]; exists {
			// Check if update is needed
			if s.needUpdate(localArticle, remoteArticle) {
				localArticle.Title = remoteArticle.Title
				localArticle.Content = remoteArticle.Content
				localArticle.Excerpt = remoteArticle.Excerpt
				localArticle.WPCategoryIDs = remoteArticle.WPCategoryIDs
				localArticle.WPTagIDs = remoteArticle.WPTagIDs
				localArticle.Slug = remoteArticle.Slug
				localArticle.FeaturedMediaID = remoteArticle.FeaturedMediaID
				localArticle.Author = remoteArticle.Author
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

	// Upload featured image if URL is provided but no media ID
	if article.FeaturedMediaURL != nil && *article.FeaturedMediaURL != "" &&
		(article.FeaturedMediaID == nil || *article.FeaturedMediaID == 0) {
		s.logger.Info("Uploading featured image from URL")
		mediaResult, uploadErr := s.wp.UploadMediaFromURL(ctx, site, *article.FeaturedMediaURL, "", article.Title)
		if uploadErr != nil {
			s.logger.ErrorWithErr(uploadErr, "Failed to upload featured image, continuing without it")
		} else {
			article.FeaturedMediaID = &mediaResult.ID
			article.FeaturedMediaURL = &mediaResult.SourceURL
			s.logger.Infof("Featured image uploaded, media ID: %d", mediaResult.ID)
		}
	}

	wpPostID, err := s.wp.CreatePost(ctx, site, article, &wp.PostOptions{Status: "publish"})
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

	s.logger.Info("Article published to WordPress successfully")
	return nil
}

func (s *service) UpdateInWordPress(ctx context.Context, article *entities.Article) error {
	site, err := s.siteService.GetSiteWithPassword(ctx, article.SiteID)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get site for update")
		return err
	}

	// Upload featured image if URL is provided but no media ID
	if article.FeaturedMediaURL != nil && *article.FeaturedMediaURL != "" &&
		(article.FeaturedMediaID == nil || *article.FeaturedMediaID == 0) {
		s.logger.Info("Uploading featured image from URL")
		mediaResult, uploadErr := s.wp.UploadMediaFromURL(ctx, site, *article.FeaturedMediaURL, "", article.Title)
		if uploadErr != nil {
			s.logger.ErrorWithErr(uploadErr, "Failed to upload featured image, continuing without it")
		} else {
			article.FeaturedMediaID = &mediaResult.ID
			article.FeaturedMediaURL = &mediaResult.SourceURL
			s.logger.Infof("Featured image uploaded, media ID: %d", mediaResult.ID)
		}
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

	if err = s.repo.Update(ctx, article); err != nil {
		s.logger.ErrorWithErr(err, "Failed to update article after WordPress deletion")
		return err
	}

	s.logger.Info("Article deleted from WordPress successfully")
	return nil
}

func (s *service) BulkPublishToWordPress(ctx context.Context, ids []int64) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}

	var publishedCount int
	for _, id := range ids {
		article, err := s.repo.GetByID(ctx, id)
		if err != nil {
			s.logger.ErrorWithErr(err, "Failed to get article for bulk publish")
			continue
		}

		if err = s.PublishToWordPress(ctx, article); err != nil {
			s.logger.ErrorWithErr(err, "Failed to publish article in bulk")
			continue
		}

		publishedCount++
	}

	s.logger.Infof("Bulk published %d articles to WordPress", publishedCount)
	return publishedCount, nil
}

func (s *service) BulkDeleteFromWordPress(ctx context.Context, ids []int64) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}

	var deletedCount int
	for _, id := range ids {
		if err := s.DeleteFromWordPress(ctx, id); err != nil {
			s.logger.ErrorWithErr(err, "Failed to delete article from WordPress in bulk")
			continue
		}

		deletedCount++
	}

	s.logger.Infof("Bulk deleted %d articles from WordPress", deletedCount)
	return deletedCount, nil
}

func (s *service) CreateDraft(ctx context.Context, exec *entities.Execution, title, content string) (*entities.Article, error) {
	wordCount := s.calculateWordCount(content)

	article := &entities.Article{
		JobID:         &exec.JobID,
		SiteID:        exec.SiteID,
		TopicID:       &exec.TopicID,
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

	return nil
}

func (s *service) calculateWordCount(content string) int {
	words := strings.Fields(content)
	return len(words)
}

func (s *service) needUpdate(local, remote *entities.Article) bool {
	return (local.Title != remote.Title) && (local.WPPostID != remote.WPPostID)
}

func (s *service) GenerateContent(ctx context.Context, input *GenerateContentInput) (*GenerateContentResult, error) {
	// Validate input
	if input.SiteID <= 0 {
		return nil, errors.Validation("Site ID is required")
	}
	if input.ProviderID <= 0 {
		return nil, errors.Validation("Provider ID is required")
	}
	if input.PromptID <= 0 {
		return nil, errors.Validation("Prompt ID is required")
	}

	// Get site info
	site, err := s.siteService.GetSite(ctx, input.SiteID)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get site for content generation")
		return nil, err
	}

	// Get provider
	provider, err := s.providerService.GetProvider(ctx, input.ProviderID)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get provider for content generation")
		return nil, err
	}

	// Determine topic title
	var topicTitle string
	var topicID *int64

	if input.TopicID != nil {
		// Use existing topic
		topic, err := s.topicService.GetTopic(ctx, *input.TopicID)
		if err != nil {
			s.logger.ErrorWithErr(err, "Failed to get topic for content generation")
			return nil, err
		}
		topicTitle = topic.Title
		topicID = &topic.ID
	} else if strings.TrimSpace(input.CustomTopicTitle) != "" {
		// Use custom topic - create it and assign to site
		topicTitle = strings.TrimSpace(input.CustomTopicTitle)

		newTopic := &entities.Topic{
			Title:     topicTitle,
			CreatedAt: time.Now(),
		}

		result, err := s.topicService.CreateAndAssignToSite(ctx, input.SiteID, newTopic)
		if err != nil {
			s.logger.ErrorWithErr(err, "Failed to create and assign custom topic")
			return nil, err
		}

		// Get the created topic ID
		if result.TotalAdded > 0 {
			// Fetch the topic by title to get its ID
			topics, err := s.topicService.GetByTitles(ctx, []string{topicTitle})
			if err == nil && len(topics) > 0 {
				topicID = &topics[0].ID
			}
		}
	} else {
		return nil, errors.Validation("Either topic ID or custom topic title is required")
	}

	// Build placeholders map
	placeholders := make(map[string]string)

	// Add standard placeholders
	placeholders["title"] = topicTitle
	placeholders["topic"] = topicTitle
	placeholders["siteName"] = site.Name
	placeholders["siteUrl"] = site.URL

	// Add custom placeholders from input
	for k, v := range input.PlaceholderValues {
		placeholders[k] = v
	}

	// Render prompts
	systemPrompt, userPrompt, err := s.promptService.RenderPrompt(ctx, input.PromptID, placeholders)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to render prompts for content generation")
		return nil, err
	}

	// Create AI client
	aiClient, err := ai.CreateClient(provider)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to create AI client for content generation")
		return nil, err
	}

	// Generate article
	s.logger.Info("Starting AI content generation")
	aiResult, err := aiClient.GenerateArticle(ctx, systemPrompt, userPrompt)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to generate article content")
		return nil, errors.Internal(err)
	}

	// Mark topic as used if we have a topic ID
	if topicID != nil {
		if err := s.topicService.MarkTopicUsed(ctx, input.SiteID, *topicID); err != nil {
			s.logger.ErrorWithErr(err, "Failed to mark topic as used")
			// Don't fail the whole operation for this
		}
	}

	// Build result
	result := &GenerateContentResult{
		Title:           aiResult.Title,
		Content:         aiResult.Content,
		Excerpt:         aiResult.Excerpt,
		MetaDescription: s.generateMetaDescription(aiResult.Excerpt, aiResult.Content),
		TopicID:         topicID,
	}

	s.logger.Info("AI content generation completed successfully")
	return result, nil
}

func (s *service) generateMetaDescription(excerpt, content string) string {
	// Use excerpt if available and not too long
	if len(excerpt) > 0 && len(excerpt) <= 160 {
		return excerpt
	}

	// Otherwise, extract from content
	// Strip HTML tags and take first 160 characters
	text := strings.TrimSpace(stripHTMLTags(content))
	if len(text) > 160 {
		// Find last space before 160 to avoid cutting words
		text = text[:160]
		lastSpace := strings.LastIndex(text, " ")
		if lastSpace > 100 {
			text = text[:lastSpace]
		}
		text = strings.TrimSuffix(text, ".") + "..."
	}

	return text
}

func stripHTMLTags(s string) string {
	var result strings.Builder
	inTag := false

	for _, r := range s {
		if r == '<' {
			inTag = true
			continue
		}
		if r == '>' {
			inTag = false
			result.WriteRune(' ')
			continue
		}
		if !inTag {
			result.WriteRune(r)
		}
	}

	// Clean up multiple spaces
	text := result.String()
	for strings.Contains(text, "  ") {
		text = strings.ReplaceAll(text, "  ", " ")
	}

	return strings.TrimSpace(text)
}
