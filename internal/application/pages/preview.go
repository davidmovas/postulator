package pages

import (
	"context"
	"strings"
	"time"

	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/kernel/dto"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type PreviewKind string

const (
	PreviewPublic PreviewKind = "public"
	PreviewIssued PreviewKind = "preview"
)

type IssuedPreview struct {
	ExpiresAt time.Time
	URL       string
}

type previewIssuer interface {
	IssuePreview(ctx context.Context, siteID string, wpID int64) (IssuedPreview, error)
	TrashItem(ctx context.Context, siteID string, wpID int64, wpType string) error
}

func (s *Service) PreviewLink(ctx context.Context, req PreviewLinkRequest) (PreviewLinkResponse, error) {
	pageID := strings.TrimSpace(req.PageID)
	if pageID == "" {
		return PreviewLinkResponse{}, errors.New(errors.Invalid, "a preview needs a page").WithDetail("field", "pageId")
	}

	page, err := s.pages.Get(ctx, pageID)
	if err != nil {
		return PreviewLinkResponse{}, err
	}
	if page.Status == pagemap.StatusArchived {
		return PreviewLinkResponse{}, errors.New(errors.Invalid, "the page is no longer on the site").
			WithDetail("field", "status").WithDetail("pageId", page.ID)
	}
	if page.WPID == nil {
		return PreviewLinkResponse{}, errors.New(errors.Invalid, "the page has not been written to the site yet").
			WithDetail("field", "wpId").WithDetail("pageId", page.ID).WithDetail("status", string(page.Status))
	}

	owner, err := s.sites.Get(ctx, page.SiteID)
	if err != nil {
		return PreviewLinkResponse{}, err
	}
	if page.Status == pagemap.StatusPublished {
		return PreviewLinkResponse{
			URL: pagemap.NewSite(owner.BaseURL).URL(page.Path), Kind: string(PreviewPublic),
		}, nil
	}

	issued, err := s.preview.IssuePreview(ctx, page.SiteID, *page.WPID)
	if err != nil {
		return PreviewLinkResponse{}, err
	}
	return PreviewLinkResponse{URL: issued.URL, ExpiresAt: dto.NewTime(issued.ExpiresAt), Kind: string(PreviewIssued)}, nil
}
