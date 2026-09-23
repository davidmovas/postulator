package images

import (
	"path/filepath"
	"strings"

	"github.com/davidmovas/postulator/internal/domain/llm"
)

type Image struct {
	Filename    string    `json:"filename"`
	ContentType string    `json:"contentType"`
	Alt         string    `json:"alt"`
	URL         string    `json:"url"`
	Bytes       []byte    `json:"-"`
	Usage       llm.Usage `json:"-"`
	WPID        int64     `json:"wpId"`
}

type Prompt struct {
	SiteID  string
	RunID   string
	ItemID  string
	Step    string
	Subject string
	Context string
	Alt     string
	Size    string
}

type Query struct {
	SiteID string
	Term   string
	Limit  int
}

var contentTypes = map[string]string{
	".png":  "image/png",
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".gif":  "image/gif",
	".webp": "image/webp",
	".avif": "image/avif",
}

func ContentType(name string) string {
	return contentTypes[strings.ToLower(filepath.Ext(name))]
}

func Supported(name string) bool {
	return ContentType(name) != ""
}
