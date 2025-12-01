package handlers

import (
	"encoding/base64"
	"path/filepath"
	"strings"

	"github.com/davidmovas/postulator/internal/domain/sites"
	"github.com/davidmovas/postulator/internal/dto"
	"github.com/davidmovas/postulator/internal/infra/wp"
	"github.com/davidmovas/postulator/pkg/ctx"
	"github.com/davidmovas/postulator/pkg/errors"
)

type MediaHandler struct {
	wpClient     wp.Client
	sitesService sites.Service
}

func NewMediaHandler(wpClient wp.Client, sitesService sites.Service) *MediaHandler {
	return &MediaHandler{
		wpClient:     wpClient,
		sitesService: sitesService,
	}
}

// UploadMedia uploads a file to WordPress media library
// fileData should be base64 encoded file content
func (h *MediaHandler) UploadMedia(siteID int64, filename string, fileData string, altText string) *dto.Response[*dto.MediaResult] {
	if filename == "" {
		return fail[*dto.MediaResult](errors.Validation("filename is required"))
	}

	if fileData == "" {
		return fail[*dto.MediaResult](errors.Validation("file data is required"))
	}

	// Validate file extension
	ext := strings.ToLower(filepath.Ext(filename))
	allowedExts := map[string]bool{
		".jpg": true, ".jpeg": true, ".png": true, ".gif": true,
		".webp": true, ".svg": true, ".bmp": true, ".ico": true,
	}
	if !allowedExts[ext] {
		return fail[*dto.MediaResult](errors.Validation("unsupported file type. Allowed: jpg, jpeg, png, gif, webp, svg, bmp, ico"))
	}

	// Decode base64 data
	data, err := base64.StdEncoding.DecodeString(fileData)
	if err != nil {
		return fail[*dto.MediaResult](errors.Validation("invalid base64 file data"))
	}

	// Get site with password
	site, err := h.sitesService.GetSiteWithPassword(ctx.FastCtx(), siteID)
	if err != nil {
		return fail[*dto.MediaResult](err)
	}

	// Upload to WordPress
	result, err := h.wpClient.UploadMedia(ctx.FastCtx(), site, filename, data, altText)
	if err != nil {
		return fail[*dto.MediaResult](err)
	}

	return ok(&dto.MediaResult{
		ID:        result.ID,
		SourceURL: result.SourceURL,
		AltText:   result.AltText,
	})
}

// UploadMediaFromURL downloads an image from URL and uploads to WordPress
func (h *MediaHandler) UploadMediaFromURL(siteID int64, imageURL string, altText string) *dto.Response[*dto.MediaResult] {
	if imageURL == "" {
		return fail[*dto.MediaResult](errors.Validation("image URL is required"))
	}

	// Get site with password
	site, err := h.sitesService.GetSiteWithPassword(ctx.FastCtx(), siteID)
	if err != nil {
		return fail[*dto.MediaResult](err)
	}

	// Upload from URL to WordPress
	result, err := h.wpClient.UploadMediaFromURL(ctx.FastCtx(), site, imageURL, "", altText)
	if err != nil {
		return fail[*dto.MediaResult](err)
	}

	return ok(&dto.MediaResult{
		ID:        result.ID,
		SourceURL: result.SourceURL,
		AltText:   result.AltText,
	})
}

// GetMedia retrieves media information from WordPress
func (h *MediaHandler) GetMedia(siteID int64, mediaID int) *dto.Response[*dto.MediaResult] {
	site, err := h.sitesService.GetSiteWithPassword(ctx.FastCtx(), siteID)
	if err != nil {
		return fail[*dto.MediaResult](err)
	}

	result, err := h.wpClient.GetMedia(ctx.FastCtx(), site, mediaID)
	if err != nil {
		return fail[*dto.MediaResult](err)
	}

	return ok(&dto.MediaResult{
		ID:        result.ID,
		SourceURL: result.SourceURL,
		AltText:   result.AltText,
	})
}

// DeleteMedia deletes media from WordPress
func (h *MediaHandler) DeleteMedia(siteID int64, mediaID int) *dto.Response[string] {
	site, err := h.sitesService.GetSiteWithPassword(ctx.FastCtx(), siteID)
	if err != nil {
		return fail[string](err)
	}

	if err = h.wpClient.DeleteMedia(ctx.FastCtx(), site, mediaID); err != nil {
		return fail[string](err)
	}

	return ok("Media deleted successfully")
}
