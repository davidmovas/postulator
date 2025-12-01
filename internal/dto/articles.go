package dto

import (
	"time"

	"github.com/davidmovas/postulator/internal/domain/entities"
)

type Article struct {
	ID            int64   `json:"id"`
	SiteID        int64   `json:"siteId"`
	JobID         *int64  `json:"jobId"`
	TopicID       *int64  `json:"topicId"`
	Title         string  `json:"title"`
	OriginalTitle string  `json:"originalTitle"`
	Content       string  `json:"content"`
	Excerpt       *string `json:"excerpt"`
	WPPostID      int     `json:"wpPostId"`
	WPPostURL     string  `json:"wpPostUrl"`
	WPCategoryIDs []int   `json:"wpCategoryIds"`
	WPTagIDs      []int   `json:"wpTagIds"`
	Status        string  `json:"status"`
	WordCount     *int    `json:"wordCount"`
	Source        string  `json:"source"`
	IsEdited      bool    `json:"isEdited"`
	CreatedAt     string  `json:"createdAt"`
	PublishedAt   *string `json:"publishedAt"`
	UpdatedAt     string  `json:"updatedAt"`
	LastSyncedAt  *string `json:"lastSyncedAt"`

	// SEO & WordPress fields
	Slug             *string `json:"slug"`
	FeaturedMediaID  *int    `json:"featuredMediaId"`
	FeaturedMediaURL *string `json:"featuredMediaUrl"`
	MetaDescription  *string `json:"metaDescription"`
	Author           *int    `json:"author"`
}

func NewArticle(entity *entities.Article) *Article {
	a := &Article{}
	return a.FromEntity(entity)
}

func (d *Article) ToEntity() (*entities.Article, error) {
	createdAt, err := StringToTime(d.CreatedAt)
	if err != nil {
		return nil, err
	}

	updatedAt, err := StringToTime(d.UpdatedAt)
	if err != nil {
		return nil, err
	}

	var publishedAt, lastSyncedAt *time.Time
	if d.PublishedAt != nil {
		var publishedAtTime time.Time
		publishedAtTime, err = StringToTime(*d.PublishedAt)
		if err != nil {
			return nil, err
		}
		publishedAt = &publishedAtTime
	}

	if d.LastSyncedAt != nil {
		var lastSyncedAtTime time.Time
		lastSyncedAtTime, err = StringToTime(*d.LastSyncedAt)
		if err != nil {
			return nil, err
		}
		lastSyncedAt = &lastSyncedAtTime
	}

	return &entities.Article{
		ID:               d.ID,
		SiteID:           d.SiteID,
		JobID:            d.JobID,
		TopicID:          d.TopicID,
		Title:            d.Title,
		OriginalTitle:    d.OriginalTitle,
		Content:          d.Content,
		Excerpt:          d.Excerpt,
		WPPostID:         d.WPPostID,
		WPPostURL:        d.WPPostURL,
		WPCategoryIDs:    d.WPCategoryIDs,
		WPTagIDs:         d.WPTagIDs,
		Status:           entities.ArticleStatus(d.Status),
		WordCount:        d.WordCount,
		Source:           entities.Source(d.Source),
		IsEdited:         d.IsEdited,
		CreatedAt:        createdAt,
		PublishedAt:      publishedAt,
		UpdatedAt:        updatedAt,
		LastSyncedAt:     lastSyncedAt,
		Slug:             d.Slug,
		FeaturedMediaID:  d.FeaturedMediaID,
		FeaturedMediaURL: d.FeaturedMediaURL,
		MetaDescription:  d.MetaDescription,
		Author:           d.Author,
	}, nil
}

func (d *Article) FromEntity(entity *entities.Article) *Article {
	d.ID = entity.ID
	d.SiteID = entity.SiteID
	d.JobID = entity.JobID
	d.TopicID = entity.TopicID
	d.Title = entity.Title
	d.OriginalTitle = entity.OriginalTitle
	d.Content = entity.Content
	d.Excerpt = entity.Excerpt
	d.WPPostID = entity.WPPostID
	d.WPPostURL = entity.WPPostURL
	d.WPCategoryIDs = entity.WPCategoryIDs
	d.WPTagIDs = entity.WPTagIDs
	d.Status = string(entity.Status)
	d.WordCount = entity.WordCount
	d.Source = string(entity.Source)
	d.IsEdited = entity.IsEdited
	d.CreatedAt = TimeToString(entity.CreatedAt)
	d.UpdatedAt = TimeToString(entity.UpdatedAt)

	// SEO & WordPress fields
	d.Slug = entity.Slug
	d.FeaturedMediaID = entity.FeaturedMediaID
	d.FeaturedMediaURL = entity.FeaturedMediaURL
	d.MetaDescription = entity.MetaDescription
	d.Author = entity.Author

	if entity.PublishedAt != nil {
		publishedAt := TimeToString(*entity.PublishedAt)
		d.PublishedAt = &publishedAt
	} else {
		d.PublishedAt = nil
	}

	if entity.LastSyncedAt != nil {
		lastSyncedAt := TimeToString(*entity.LastSyncedAt)
		d.LastSyncedAt = &lastSyncedAt
	} else {
		d.LastSyncedAt = nil
	}

	return d
}

type ArticleListFilter struct {
	SiteID     int64   `json:"siteId"`
	Status     *string `json:"status"`
	Source     *string `json:"source"`
	CategoryID *int    `json:"categoryId"`
	Search     *string `json:"search"`
	SortBy     string  `json:"sortBy"`
	SortOrder  string  `json:"sortOrder"`
	Limit      int     `json:"limit"`
	Offset     int     `json:"offset"`
}

type ArticleListResult struct {
	Articles []*Article `json:"articles"`
	Total    int        `json:"total"`
}

type WPInfoUpdate struct {
	ID          int64   `json:"id"`
	WPPostID    int     `json:"wpPostId"`
	WPPostURL   string  `json:"wpPostUrl"`
	Status      string  `json:"status"`
	PublishedAt *string `json:"publishedAt"`
}

func NewWPInfoUpdate(entity *entities.WPInfoUpdate) *WPInfoUpdate {
	w := &WPInfoUpdate{}
	return w.FromEntity(entity)
}

func (d *WPInfoUpdate) ToEntity() (*entities.WPInfoUpdate, error) {
	var publishedAt *time.Time
	if d.PublishedAt != nil {
		publishedAtTime, err := StringToTime(*d.PublishedAt)
		if err != nil {
			return nil, err
		}
		publishedAt = &publishedAtTime
	}

	return &entities.WPInfoUpdate{
		ID:          d.ID,
		WPPostID:    d.WPPostID,
		WPPostURL:   d.WPPostURL,
		Status:      entities.Status(d.Status),
		PublishedAt: publishedAt,
	}, nil
}

func (d *WPInfoUpdate) FromEntity(entity *entities.WPInfoUpdate) *WPInfoUpdate {
	d.ID = entity.ID
	d.WPPostID = entity.WPPostID
	d.WPPostURL = entity.WPPostURL
	d.Status = string(entity.Status)

	if entity.PublishedAt != nil {
		publishedAt := TimeToString(*entity.PublishedAt)
		d.PublishedAt = &publishedAt
	} else {
		d.PublishedAt = nil
	}

	return d
}
