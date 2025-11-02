package wp

import "time"

type wpError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Status int `json:"status"`
	} `json:"data"`
}

type wpCategory struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	Count       int    `json:"count"`
}

type wpPost struct {
	ID          int       `json:"id"`
	Date        time.Time `json:"date"`
	DateGMT     time.Time `json:"date_gmt"`
	Modified    time.Time `json:"modified"`
	ModifiedGMT time.Time `json:"modified_gmt"`
	Slug        string    `json:"slug"`
	Status      string    `json:"status"`
	Type        string    `json:"type"`
	Title       struct {
		Rendered string `json:"rendered"`
	} `json:"title"`
	Content struct {
		Rendered string `json:"rendered"`
	} `json:"content"`
	Excerpt struct {
		Rendered string `json:"rendered"`
	} `json:"excerpt"`
	Author        int    `json:"author"`
	FeaturedMedia int    `json:"featured_media"`
	Categories    []int  `json:"categories"`
	Tags          []int  `json:"tags"`
	Link          string `json:"link"`
}
