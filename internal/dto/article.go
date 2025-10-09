package dto

import "Postulator/internal/domain/entities"

type Article struct {
	ID            int64   `json:"id"`
	SiteID        int64   `json:"siteId"`
	JobID         *int64  `json:"jobId,omitempty"`
	TopicID       int64   `json:"topicId"`
	Title         string  `json:"title"`
	OriginalTitle string  `json:"originalTitle"`
	Content       string  `json:"content"`
	Excerpt       *string `json:"excerpt,omitempty"`
	WPPostID      int     `json:"wpPostId"`
	WPPostURL     string  `json:"wpPostUrl"`
	WPCategoryID  int     `json:"wpCategoryId"`
	Status        string  `json:"status"`
	WordCount     *int    `json:"wordCount,omitempty"`
	CreatedAt     string  `json:"createdAt"`
	PublishedAt   *string `json:"publishedAt,omitempty"`
}

func FromArticle(e *entities.Article) *Article {
	if e == nil {
		return nil
	}
	var published *string
	if e.PublishedAt != nil {
		v := e.PublishedAt.UTC().Format(timeLayout)
		published = &v
	}
	return &Article{
		ID:            e.ID,
		SiteID:        e.SiteID,
		JobID:         e.JobID,
		TopicID:       e.TopicID,
		Title:         e.Title,
		OriginalTitle: e.OriginalTitle,
		Content:       e.Content,
		Excerpt:       e.Excerpt,
		WPPostID:      e.WPPostID,
		WPPostURL:     e.WPPostURL,
		WPCategoryID:  e.WPCategoryID,
		Status:        string(e.Status),
		WordCount:     e.WordCount,
		CreatedAt:     e.CreatedAt.UTC().Format(timeLayout),
		PublishedAt:   published,
	}
}

func FromArticles(items []*entities.Article) []*Article {
	out := make([]*Article, 0, len(items))
	for _, it := range items {
		out = append(out, FromArticle(it))
	}
	return out
}
