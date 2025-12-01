package dto

// MediaResult represents the result of a media upload operation
type MediaResult struct {
	ID        int    `json:"id"`
	SourceURL string `json:"sourceUrl"`
	AltText   string `json:"altText"`
}

// MediaUploadInput represents input for uploading media
type MediaUploadInput struct {
	SiteID   int64  `json:"siteId"`
	Filename string `json:"filename"`
	FileData string `json:"fileData"` // Base64 encoded file content
	AltText  string `json:"altText"`
}
