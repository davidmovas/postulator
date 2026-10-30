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
	if err := s.deps.Types.Seed(ctx, state.SiteID); err != nil {
		return err
	}
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
	types := make([]string, 0, len(postTypes)+len(readers.Manifest.PostTypes)+1)
	for _, postType := range postTypes {
		switch {
		case !postType.Viewable, !listable(postType.Slug):
			continue
		case source == SourceCore && pagemap.WPType(postType.Slug).StoreAddressed():
			continue
		case source == SourcePlugin && len(readers.Manifest.PostTypes) > 0 && !contains(readers.Manifest.PostTypes, postType.Slug):
			continue
		}
		types = append(types, postType.Slug)
	}
	if source != SourcePlugin {
		return types, nil
	}
	for _, slug := range readers.Manifest.PostTypes {
		if listable(slug) && !contains(types, slug) {
			types = append(types, slug)
		}
	}
	if len(readers.Manifest.Taxonomies) == 0 || contains(readers.Manifest.Taxonomies, typeTerm) {
		types = append(types, typeTerm)
	}
	return types, nil
}

func listable(slug string) bool {
	return slug != typeAttachment && slug != typeVariation && pagemap.WPType(slug).Valid()
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
