package wp

import (
	"context"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
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
	Parent    int64
}

type mediaPayload struct {
	Title     renderedText `json:"title"`
	SourceURL string       `json:"source_url"`
	AltText   string       `json:"alt_text"`
	MimeType  string       `json:"mime_type"`
	ID        int64        `json:"id"`
	Post      int64        `json:"post"`
}

func (p mediaPayload) mediaItem() MediaItem {
	return MediaItem{
		ID:        p.ID,
		SourceURL: p.SourceURL,
		Alt:       p.AltText,
		Title:     p.Title.value(),
		MimeType:  p.MimeType,
		Parent:    p.Post,
	}
}

type UpdateMedia struct {
	Alt   *string
	Title *string
}

func (c *Client) GetMedia(ctx context.Context, id int64) (MediaItem, error) {
	_, body, err := c.do(ctx, request{
		method:    http.MethodGet,
		namespace: coreNamespace,
		path:      resourcePath("/media", id),
		query:     url.Values{"context": {"edit"}},
	})
	if err != nil {
		return MediaItem{}, err
	}
	var held mediaPayload
	if err := decodeJSON(body, &held); err != nil {
		return MediaItem{}, err
	}
	return held.mediaItem(), nil
}

func (c *Client) UpdateMedia(ctx context.Context, id int64, in UpdateMedia) (MediaItem, error) {
	fields := make(map[string]any, 2)
	if in.Alt != nil {
		fields["alt_text"] = *in.Alt
	}
	if in.Title != nil {
		fields["title"] = *in.Title
	}
	if len(fields) == 0 {
		return MediaItem{}, errors.New(errors.Invalid, "the media update carries no fields")
	}
	payload, err := encodeJSON(fields)
	if err != nil {
		return MediaItem{}, err
	}
	_, body, err := c.do(ctx, request{
		method:      http.MethodPost,
		namespace:   coreNamespace,
		path:        resourcePath("/media", id),
		query:       url.Values{"context": {"edit"}},
		body:        payload,
		contentType: contentTypeJSON,
	})
	if err != nil {
		return MediaItem{}, err
	}
	var written mediaPayload
	if err := decodeJSON(body, &written); err != nil {
		return MediaItem{}, err
	}
	return written.mediaItem(), nil
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

type MediaQuery struct {
	Search  string
	Page    int
	PerPage int
}

func (q MediaQuery) values() url.Values {
	query := url.Values{}
	query.Set("context", "edit")
	query.Set("page", strconv.Itoa(pageNumber(q.Page)))
	query.Set("per_page", strconv.Itoa(perPageSize(q.PerPage)))
	query.Set("orderby", "id")
	query.Set("order", "asc")
	if q.Search != "" {
		query.Set("search", q.Search)
	}
	return query
}

type MediaPage = Page[MediaItem]

func (c *Client) ListMedia(ctx context.Context, query MediaQuery) (MediaPage, error) {
	resp, body, err := c.do(ctx, request{
		method:    http.MethodGet,
		namespace: coreNamespace,
		path:      "/media",
		query:     query.values(),
	})
	if err != nil {
		if endOfList(err) {
			return MediaPage{Page: pageNumber(query.Page)}, nil
		}
		return MediaPage{}, err
	}

	var payload []mediaPayload
	if decodeErr := decodeJSON(body, &payload); decodeErr != nil {
		return MediaPage{}, decodeErr
	}

	items := make([]MediaItem, 0, len(payload))
	for index := range payload {
		items = append(items, payload[index].mediaItem())
	}
	return newPage(resp, pageNumber(query.Page), items), nil
}
