package dto

import "github.com/davidmovas/postulator/internal/domain/entities"

type Category struct {
	ID           int64   `json:"id"`
	SiteID       int64   `json:"siteId"`
	WPCategoryID int     `json:"wpCategoryId"`
	Name         string  `json:"name"`
	Slug         *string `json:"slug"`
	Description  *string `json:"description"`
	Count        int     `json:"count"`
	CreatedAt    string  `json:"createdAt"`
	UpdatedAt    string  `json:"updatedAt"`
}

func NewCategory(entity *entities.Category) *Category {
	c := &Category{}
	return c.FromEntity(entity)
}

func (d *Category) ToEntity() (*entities.Category, error) {
	createdAt, err := StringToTime(d.CreatedAt)
	if err != nil {
		return nil, err
	}

	updatedAt, err := StringToTime(d.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return &entities.Category{
		ID:           d.ID,
		SiteID:       d.SiteID,
		WPCategoryID: d.WPCategoryID,
		Name:         d.Name,
		Slug:         d.Slug,
		Description:  d.Description,
		Count:        d.Count,
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
	}, nil
}

func (d *Category) FromEntity(entity *entities.Category) *Category {
	d.ID = entity.ID
	d.SiteID = entity.SiteID
	d.WPCategoryID = entity.WPCategoryID
	d.Name = entity.Name
	d.Slug = entity.Slug
	d.Description = entity.Description
	d.Count = entity.Count
	d.CreatedAt = TimeToString(entity.CreatedAt)
	d.UpdatedAt = TimeToString(entity.UpdatedAt)
	return d
}

type Statistics struct {
	CategoryID        int64  `json:"categoryId"`
	Date              string `json:"date"`
	ArticlesPublished int    `json:"articlesPublished"`
	TotalWords        int    `json:"totalWords"`
}

func NewStatistics(entity *entities.Statistics) *Statistics {
	s := &Statistics{}
	return s.FromEntity(entity)
}

func (d *Statistics) ToEntity() (*entities.Statistics, error) {
	date, err := StringToTime(d.Date)
	if err != nil {
		return nil, err
	}

	return &entities.Statistics{
		CategoryID:        d.CategoryID,
		Date:              date,
		ArticlesPublished: d.ArticlesPublished,
		TotalWords:        d.TotalWords,
	}, nil
}

func (d *Statistics) FromEntity(entity *entities.Statistics) *Statistics {
	d.CategoryID = entity.CategoryID
	d.Date = TimeToString(entity.Date)
	d.ArticlesPublished = entity.ArticlesPublished
	d.TotalWords = entity.TotalWords
	return d
}
