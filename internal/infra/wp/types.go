package wp

import (
	"strings"
	"time"
)

// WPTime handles WordPress date format without timezone suffix
type WPTime struct {
	time.Time
}

func (t *WPTime) UnmarshalJSON(data []byte) error {
	s := strings.Trim(string(data), `"`)
	if s == "" || s == "null" {
		t.Time = time.Time{}
		return nil
	}

	// Try RFC3339 first (with timezone)
	parsed, err := time.Parse(time.RFC3339, s)
	if err == nil {
		t.Time = parsed
		return nil
	}

	// WordPress format without timezone - assume UTC
	parsed, err = time.Parse("2006-01-02T15:04:05", s)
	if err != nil {
		return err
	}
	t.Time = parsed.UTC()
	return nil
}

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

type wpTag struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	Count       int    `json:"count"`
}

type wpMedia struct {
	ID           int    `json:"id"`
	SourceURL    string `json:"source_url"`
	MediaDetails struct {
		Width  int `json:"width"`
		Height int `json:"height"`
		Sizes  map[string]struct {
			SourceURL string `json:"source_url"`
			Width     int    `json:"width"`
			Height    int    `json:"height"`
		} `json:"sizes"`
	} `json:"media_details"`
	AltText string `json:"alt_text"`
	Title   struct {
		Rendered string `json:"rendered"`
	} `json:"title"`
}

type wpPost struct {
	ID          int    `json:"id"`
	Date        WPTime `json:"date"`
	DateGMT     WPTime `json:"date_gmt"`
	Modified    WPTime `json:"modified"`
	ModifiedGMT WPTime `json:"modified_gmt"`
	Slug        string    `json:"slug"`
	Status      string    `json:"status"`
	Type        string    `json:"type"`
	Title       struct {
		Rendered string `json:"rendered"`
	} `json:"title"`
	Content struct {
		Rendered  string `json:"rendered"`
		Protected bool   `json:"protected"`
	} `json:"content"`
	Excerpt struct {
		Rendered  string `json:"rendered"`
		Protected bool   `json:"protected"`
	} `json:"excerpt"`
	Author        int    `json:"author"`
	FeaturedMedia int    `json:"featured_media"`
	Categories    []int  `json:"categories"`
	Tags          []int  `json:"tags"`
	Link          string `json:"link"`
	Meta          struct {
		Description string `json:"_yoast_wpseo_metadesc,omitempty"`
	} `json:"meta,omitempty"`
}

type wpPage struct {
	ID          int    `json:"id"`
	Date        WPTime `json:"date"`
	DateGMT     WPTime `json:"date_gmt"`
	Modified    WPTime `json:"modified"`
	ModifiedGMT WPTime `json:"modified_gmt"`
	Slug        string `json:"slug"`
	Status      string `json:"status"`
	Type        string `json:"type"`
	Parent      int    `json:"parent"`
	Title       struct {
		Rendered string `json:"rendered"`
	} `json:"title"`
	Content struct {
		Rendered  string `json:"rendered"`
		Protected bool   `json:"protected"`
	} `json:"content"`
	Excerpt struct {
		Rendered  string `json:"rendered"`
		Protected bool   `json:"protected"`
	} `json:"excerpt"`
	Author        int    `json:"author"`
	FeaturedMedia int    `json:"featured_media"`
	MenuOrder     int    `json:"menu_order"`
	Link          string `json:"link"`
	Template      string `json:"template"`
}
