package wp

import (
	"context"
	"fmt"

	"github.com/davidmovas/postulator/internal/domain/entities"
	"github.com/davidmovas/postulator/pkg/errors"
)

func (c *restyClient) GetPost(ctx context.Context, s *entities.Site, postID int) (*entities.Article, error) {
	var post wpPost

	resp, err := c.resty.R().
		SetContext(ctx).
		SetBasicAuth(s.WPUsername, s.WPPassword).
		SetResult(&post).
		Get(c.getAPIURL(s.URL, fmt.Sprintf("posts/%d", postID)))

	if err != nil {
		return nil, errors.WordPress("failed to make request", err)
	}

	if resp.StatusCode() != 200 {
		return nil, errors.WordPress(fmt.Sprintf("wordpress API returned status %d: %s", resp.StatusCode(), resp.String()), nil)
	}

	return c.convertWPPostToArticle(post, s.ID), nil
}

func (c *restyClient) GetPosts(ctx context.Context, s *entities.Site) ([]*entities.Article, error) {
	var wpPosts []wpPost

	resp, err := c.resty.R().
		SetContext(ctx).
		SetBasicAuth(s.WPUsername, s.WPPassword).
		SetQueryParams(map[string]string{
			"per_page": "100",
			"orderby":  "date",
			"order":    "desc",
		}).
		SetResult(&wpPosts).
		Get(c.getAPIURL(s.URL, "posts"))

	if err != nil {
		return nil, errors.WordPress("failed to make request", err)
	}

	if resp.StatusCode() != 200 {
		return nil, errors.WordPress(fmt.Sprintf("wordpress API returned status %d: %s", resp.StatusCode(), resp.String()), nil)
	}

	articles := make([]*entities.Article, 0, len(wpPosts))
	for _, post := range wpPosts {
		articles = append(articles, c.convertWPPostToArticle(post, s.ID))
	}

	return articles, nil
}

func (c *restyClient) CreatePost(ctx context.Context, s *entities.Site, article *entities.Article) (int, error) {
	postData := map[string]interface{}{
		"title":   map[string]string{"raw": article.Title},
		"content": map[string]string{"raw": article.Content},
		"status":  "publish",
	}

	if article.Excerpt != nil && *article.Excerpt != "" {
		postData["excerpt"] = map[string]string{"raw": *article.Excerpt}
	}

	if len(article.WPCategoryIDs) > 0 {
		postData["categories"] = article.WPCategoryIDs
	}

	var createdPost wpPost

	resp, err := c.resty.R().
		SetContext(ctx).
		SetBasicAuth(s.WPUsername, s.WPPassword).
		SetBody(postData).
		SetResult(&createdPost).
		Post(c.getAPIURL(s.URL, "posts"))

	if err != nil {
		return 0, errors.WordPress("failed to make request", err)
	}

	if resp.StatusCode() != 201 {
		return 0, errors.WordPress(fmt.Sprintf("wordpress API returned status %d: %s", resp.StatusCode(), resp.String()), nil)
	}

	article.WPPostID = createdPost.ID
	article.WPPostURL = createdPost.Link

	return createdPost.ID, nil
}

func (c *restyClient) UpdatePost(ctx context.Context, s *entities.Site, article *entities.Article) error {
	postData := map[string]interface{}{}

	if article.Title != "" {
		postData["title"] = map[string]string{"raw": article.Title}
	}

	if article.Content != "" {
		postData["content"] = map[string]string{"raw": article.Content}
	}

	if article.Excerpt != nil {
		postData["excerpt"] = map[string]string{"raw": *article.Excerpt}
	}

	if len(article.WPCategoryIDs) > 0 {
		postData["categories"] = article.WPCategoryIDs
	}

	var updatedPost wpPost

	resp, err := c.resty.R().
		SetContext(ctx).
		SetBasicAuth(s.WPUsername, s.WPPassword).
		SetBody(postData).
		SetResult(&updatedPost).
		Post(c.getAPIURL(s.URL, fmt.Sprintf("posts/%d", article.WPPostID)))

	if err != nil {
		return errors.WordPress("failed to make request", err)
	}

	if resp.StatusCode() != 200 {
		return errors.WordPress(fmt.Sprintf("wordpress API returned status %d: %s", resp.StatusCode(), resp.String()), nil)
	}

	article.WPPostURL = updatedPost.Link

	return nil
}

func (c *restyClient) DeletePost(ctx context.Context, s *entities.Site, postID int) error {
	resp, err := c.resty.R().
		SetContext(ctx).
		SetBasicAuth(s.WPUsername, s.WPPassword).
		SetQueryParam("force", "true").
		Delete(c.getAPIURL(s.URL, fmt.Sprintf("posts/%d", postID)))

	if err != nil {
		return errors.WordPress("failed to make request", err)
	}

	if resp.StatusCode() != 200 && resp.StatusCode() != 204 {
		return errors.WordPress(fmt.Sprintf("wordpress API returned status %d: %s", resp.StatusCode(), resp.String()), nil)
	}

	return nil
}

func (c *restyClient) convertWPPostToArticle(wpPost wpPost, siteID int64) *entities.Article {
	wordCount := c.calculateWordCount(wpPost.Content.Rendered)

	var status entities.ArticleStatus
	switch wpPost.Status {
	case "publish":
		status = entities.StatusPublished
	case "draft":
		status = entities.StatusDraft
	default:
		status = entities.StatusUnknown
	}

	return &entities.Article{
		SiteID:        siteID,
		Title:         wpPost.Title.Rendered,
		OriginalTitle: wpPost.Title.Rendered,
		Content:       wpPost.Content.Rendered,
		Excerpt:       &wpPost.Excerpt.Rendered,
		WPPostID:      wpPost.ID,
		WPPostURL:     wpPost.Link,
		WPCategoryIDs: wpPost.Categories,
		Status:        status,
		Source:        entities.SourceImported,
		WordCount:     &wordCount,
		PublishedAt:   &wpPost.Date,
		CreatedAt:     wpPost.Date,
		UpdatedAt:     wpPost.Modified,
		LastSyncedAt:  &wpPost.Modified,
	}
}

func (c *restyClient) calculateWordCount(content string) int {
	words := 0
	inWord := false

	for _, r := range content {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			if inWord {
				words++
				inWord = false
			}
		} else {
			inWord = true
		}
	}

	if inWord {
		words++
	}

	return words
}
