package observe

import (
	"context"
	"slices"

	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/site"
)

const (
	typeAttachment = "attachment"
	typeVariation  = "product_variation"
	typeTerm       = string(pagemap.WPProductCategory)
)

func (s *Service) inventory(ctx context.Context, state *State, readers Readers) error {
	tracked, err := s.findings(ctx, state.SiteID, finding.PluginOutdated)
	if err != nil {
		return err
	}
	switch {
	case readers.Plugin != nil && readers.Manifest.Outdated:
		err = tracked.raise(ctx, finding.PluginOutdated, nil, finding.SubjectSite, map[string]string{
			"version": readers.Manifest.Version, "required": readers.Manifest.Required,
		})
	default:
		err = tracked.clear(ctx, finding.PluginOutdated, finding.SubjectSite)
	}
	if err != nil {
		return err
	}

	types, err := observedTypes(ctx, state.Source, readers)
	if err != nil {
		return err
	}
	state.Types = types

	technical, err := technicalPages(ctx, readers)
	if err != nil {
		return err
	}
	state.Technical = technical
	return nil
}

func observedTypes(ctx context.Context, source string, readers Readers) ([]string, error) {
	postTypes, err := readers.Core.PostTypes(ctx)
	if err != nil {
		return nil, err
	}
	types := make([]string, 0, len(postTypes)+1)
	for _, postType := range postTypes {
		wpType := pagemap.WPType(postType.Slug)
		switch {
		case !postType.Viewable, postType.Slug == typeAttachment, postType.Slug == typeVariation, !wpType.Valid():
			continue
		case source == SourceCore && wpType.StoreAddressed():
			continue
		case source == SourcePlugin && len(readers.Manifest.PostTypes) > 0 && !contains(readers.Manifest.PostTypes, postType.Slug):
			continue
		}
		types = append(types, postType.Slug)
	}
	if source == SourcePlugin && (len(readers.Manifest.Taxonomies) == 0 || contains(readers.Manifest.Taxonomies, typeTerm)) {
		types = append(types, typeTerm)
	}
	return types, nil
}

func technicalPages(ctx context.Context, readers Readers) (site.TechnicalPages, error) {
	if readers.plugin(capabilitySettings) {
		return readers.Plugin.Settings(ctx)
	}
	settings, err := readers.Core.SiteSettings(ctx)
	if err != nil {
		return site.TechnicalPages{}, err
	}
	technical := site.TechnicalPages{}
	if settings.ShowOnFront == "page" {
		technical.Front = settings.FrontPageID
		technical.Posts = settings.PostsPageID
	}
	if readers.Store == nil {
		return technical, nil
	}
	store, err := readers.Store.StorePages(ctx)
	if err != nil {
		return site.TechnicalPages{}, err
	}
	technical.Shop, technical.Cart, technical.Checkout, technical.Account, technical.Terms =
		store.Shop, store.Cart, store.Checkout, store.Account, store.Terms
	return technical, nil
}

func contains(list []string, value string) bool {
	return slices.Contains(list, value)
}
