package wpmedia

import (
	"context"

	"github.com/davidmovas/postulator/internal/adapters/images"
	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const defaultLimit = 3

type siteClients interface {
	Client(ctx context.Context, siteID string) (*wp.Client, error)
}

type Source struct {
	sites siteClients
}

func New(sites siteClients) *Source {
	return &Source{sites: sites}
}

func (s *Source) Pick(ctx context.Context, query images.Query) ([]images.Image, error) {
	if query.SiteID == "" {
		return nil, errors.New(errors.Invalid, "a media search needs a site")
	}

	client, err := s.sites.Client(ctx, query.SiteID)
	if err != nil {
		return nil, err
	}

	limit := query.Limit
	if limit <= 0 {
		limit = defaultLimit
	}

	listed, err := client.ListMedia(ctx, wp.MediaQuery{Search: query.Term, PerPage: limit})
	if err != nil {
		return nil, err
	}

	picked := make([]images.Image, 0, len(listed.Items))
	for _, item := range listed.Items {
		alt := item.Alt
		if alt == "" {
			alt = item.Title
		}
		picked = append(picked, images.Image{
			Filename:    item.Title,
			ContentType: item.MimeType,
			Alt:         alt,
			URL:         item.SourceURL,
			WPID:        item.ID,
		})
	}
	return picked, nil
}
