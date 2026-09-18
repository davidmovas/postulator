package wp

import (
	"context"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type Media struct {
	Filename    string
	ContentType string
	Bytes       []byte
	Alt         string
	Title       string
}

type MediaItem struct {
	SourceURL string
	Alt       string
	Title     string
	MimeType  string
	ID        int64
}

type mediaPayload struct {
	Title     renderedText `json:"title"`
	SourceURL string       `json:"source_url"`
	AltText   string       `json:"alt_text"`
	MimeType  string       `json:"mime_type"`
	ID        int64        `json:"id"`
}

func (p mediaPayload) mediaItem() MediaItem {
	return MediaItem{
		ID:        p.ID,
		SourceURL: p.SourceURL,
		Alt:       p.AltText,
		Title:     p.Title.value(),
		MimeType:  p.MimeType,
	}
}

func (c *Client) UploadMedia(ctx context.Context, media Media) (MediaItem, error) {
	filename := safeFilename(media.Filename)
	if filename == "" || len(media.Bytes) == 0 {
		return MediaItem{}, errors.New(errors.Invalid, "the upload needs a file name and file bytes")
	}

	contentType := media.ContentType
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	header := http.Header{}
	header.Set("Content-Disposition", `attachment; filename="`+filename+`"`)

	_, body, err := c.do(ctx, request{
		method:      http.MethodPost,
		namespace:   coreNamespace,
		path:        "/media",
		body:        media.Bytes,
		contentType: contentType,
		header:      header,
	})
	if err != nil {
		return MediaItem{}, err
	}

	var uploaded mediaPayload
	if err := decodeJSON(body, &uploaded); err != nil {
		return MediaItem{}, err
	}

	attributes := make(map[string]any, 2)
	if media.Alt != "" {
		attributes["alt_text"] = media.Alt
	}
	if media.Title != "" {
		attributes["title"] = media.Title
	}
	if len(attributes) == 0 {
		return uploaded.mediaItem(), nil
	}

	payload, err := encodeJSON(attributes)
	if err != nil {
		return MediaItem{}, err
	}

	_, body, err = c.do(ctx, request{
		method:      http.MethodPost,
		namespace:   coreNamespace,
		path:        resourcePath("/media", uploaded.ID),
		body:        payload,
		contentType: contentTypeJSON,
	})
	if err != nil {
		return MediaItem{}, err
	}

	var described mediaPayload
	if err := decodeJSON(body, &described); err != nil {
		return MediaItem{}, err
	}
	return described.mediaItem(), nil
}

func safeFilename(name string) string {
	base := filepath.Base(strings.TrimSpace(name))
	if base == "." || base == string(filepath.Separator) {
		return ""
	}
	return strings.NewReplacer(`"`, "", "\r", "", "\n", "").Replace(base)
}
