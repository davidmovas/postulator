package wp

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/davidmovas/postulator/internal/domain/entities"
	"github.com/davidmovas/postulator/pkg/errors"
)

func (c *restyClient) UploadMedia(ctx context.Context, s *entities.Site, filename string, data []byte, altText string) (*MediaResult, error) {
	var uploadedMedia wpMedia

	// Determine content type from filename extension
	contentType := getContentType(filename)

	resp, err := c.resty.R().
		SetContext(ctx).
		SetBasicAuth(s.WPUsername, s.WPPassword).
		SetHeader("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename)).
		SetHeader("Content-Type", contentType).
		SetBody(data).
		SetResult(&uploadedMedia).
		Post(c.getAPIURL(s.URL, "media"))
	if err != nil {
		return nil, errors.WordPress("failed to upload media", err)
	}

	if resp.StatusCode() != 201 {
		return nil, errors.WordPress(fmt.Sprintf("wordpress API returned status %d: %s", resp.StatusCode(), resp.String()), nil)
	}

	// Update alt text if provided
	if altText != "" {
		if err := c.updateMediaAltText(ctx, s, uploadedMedia.ID, altText); err != nil {
			// Log but don't fail - media was uploaded successfully
		}
	}

	return &MediaResult{
		ID:        uploadedMedia.ID,
		SourceURL: uploadedMedia.SourceURL,
		AltText:   altText,
	}, nil
}

func (c *restyClient) UploadMediaFromURL(ctx context.Context, s *entities.Site, imageURL, filename, altText string) (*MediaResult, error) {
	// Download the image first
	httpResp, err := http.Get(imageURL)
	if err != nil {
		return nil, errors.WordPress("failed to download image from URL", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		return nil, errors.WordPress(fmt.Sprintf("failed to download image, status: %d", httpResp.StatusCode), nil)
	}

	data, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, errors.WordPress("failed to read image data", err)
	}

	// If filename is empty, try to extract from URL
	if filename == "" {
		filename = extractFilenameFromURL(imageURL)
	}

	return c.UploadMedia(ctx, s, filename, data, altText)
}

func (c *restyClient) GetMedia(ctx context.Context, s *entities.Site, mediaID int) (*MediaResult, error) {
	var media wpMedia

	resp, err := c.resty.R().
		SetContext(ctx).
		SetBasicAuth(s.WPUsername, s.WPPassword).
		SetResult(&media).
		Get(c.getAPIURL(s.URL, fmt.Sprintf("media/%d", mediaID)))
	if err != nil {
		return nil, errors.WordPress("failed to get media", err)
	}

	if resp.StatusCode() != 200 {
		return nil, errors.WordPress(fmt.Sprintf("wordpress API returned status %d: %s", resp.StatusCode(), resp.String()), nil)
	}

	return &MediaResult{
		ID:        media.ID,
		SourceURL: media.SourceURL,
		AltText:   media.AltText,
	}, nil
}

func (c *restyClient) DeleteMedia(ctx context.Context, s *entities.Site, mediaID int) error {
	resp, err := c.resty.R().
		SetContext(ctx).
		SetBasicAuth(s.WPUsername, s.WPPassword).
		SetQueryParam("force", "true").
		Delete(c.getAPIURL(s.URL, fmt.Sprintf("media/%d", mediaID)))
	if err != nil {
		return errors.WordPress("failed to delete media", err)
	}

	if resp.StatusCode() != 200 && resp.StatusCode() != 204 {
		return errors.WordPress(fmt.Sprintf("wordpress API returned status %d: %s", resp.StatusCode(), resp.String()), nil)
	}

	return nil
}

func (c *restyClient) updateMediaAltText(ctx context.Context, s *entities.Site, mediaID int, altText string) error {
	resp, err := c.resty.R().
		SetContext(ctx).
		SetBasicAuth(s.WPUsername, s.WPPassword).
		SetBody(map[string]interface{}{
			"alt_text": altText,
		}).
		Post(c.getAPIURL(s.URL, fmt.Sprintf("media/%d", mediaID)))
	if err != nil {
		return errors.WordPress("failed to update media alt text", err)
	}

	if resp.StatusCode() != 200 {
		return errors.WordPress(fmt.Sprintf("wordpress API returned status %d: %s", resp.StatusCode(), resp.String()), nil)
	}

	return nil
}

func getContentType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".svg":
		return "image/svg+xml"
	case ".bmp":
		return "image/bmp"
	case ".ico":
		return "image/x-icon"
	case ".tiff", ".tif":
		return "image/tiff"
	case ".pdf":
		return "application/pdf"
	case ".mp4":
		return "video/mp4"
	case ".webm":
		return "video/webm"
	case ".mp3":
		return "audio/mpeg"
	case ".wav":
		return "audio/wav"
	case ".ogg":
		return "audio/ogg"
	default:
		return "application/octet-stream"
	}
}

func extractFilenameFromURL(url string) string {
	// Get the last path segment
	parts := strings.Split(url, "/")
	if len(parts) == 0 {
		return "image.jpg"
	}

	filename := parts[len(parts)-1]

	// Remove query parameters
	if idx := strings.Index(filename, "?"); idx != -1 {
		filename = filename[:idx]
	}

	// If no extension, add .jpg as default
	if !strings.Contains(filename, ".") {
		filename += ".jpg"
	}

	return filename
}
