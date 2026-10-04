package steps

import (
	"context"

	"github.com/davidmovas/postulator/internal/domain/pagemap"
)

func archiveAbsent(ctx context.Context, deps Deps, siteID string, state *SiteSyncResult) error {
	pages, err := deps.Pages.ListBySite(ctx, siteID)
	if err != nil {
		return err
	}

	now := deps.now()
	stale := make([]pagemap.Page, 0)
	for i := range pages {
		page := pages[i]
		if page.WPID == nil || page.Status == pagemap.StatusArchived || !covered(state.Source, page.WPType) {
			continue
		}
		if page.LastSyncedAt != nil && !page.LastSyncedAt.Before(state.StartedAt) {
			continue
		}
		page.Status = pagemap.StatusArchived
		page.UpdatedAt = now
		stale = append(stale, page)
	}
	if len(stale) == 0 {
		return nil
	}

	if updateErr := updateAll(ctx, deps, stale); updateErr != nil {
		return updateErr
	}
	state.Archived += len(stale)
	return nil
}

func covered(source string, wpType pagemap.WPType) bool {
	if source == SourcePlugin {
		return true
	}
	for i := range coreTypes {
		if string(coreTypes[i]) == string(wpType) {
			return true
		}
	}
	return false
}
