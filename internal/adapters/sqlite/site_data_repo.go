package sqlite

import (
	"context"
	"database/sql"

	"github.com/davidmovas/postulator/internal/domain/site"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	selectSiteHeld = `SELECT
    (SELECT count(*) FROM pages WHERE site_id = ?1),
    (SELECT count(*) FROM findings WHERE site_id = ?1),
    (SELECT count(*) FROM entities WHERE site_id = ?1),
    (SELECT count(*) FROM edges WHERE site_id = ?1),
    (SELECT count(*) FROM runs WHERE site_id = ?1),
    (SELECT count(*) FROM runs WHERE site_id = ?1 AND status IN ('waiting', 'paused')),
    (SELECT count(*) FROM conversations WHERE site_id = ?1)`
	selectSiteBusy = `SELECT id FROM runs WHERE site_id = ? AND status IN ('pending', 'running') ORDER BY created_at, id`

	deleteSiteRuns          = `DELETE FROM runs WHERE site_id = ?`
	deleteSiteConversations = `DELETE FROM conversations WHERE site_id = ?`
	deleteSiteFindings      = `DELETE FROM findings WHERE site_id = ?`
	deleteSitePages         = `DELETE FROM pages WHERE site_id = ?`
	deleteSiteTopicBuilds   = `DELETE FROM topic_builds WHERE site_id = ?`
)

type SiteDataRepo struct {
	store *Store
}

func NewSiteDataRepo(store *Store) *SiteDataRepo {
	return &SiteDataRepo{store: store}
}

func scanHeld(rows *sql.Rows) (site.Held, error) {
	var held site.Held
	err := rows.Scan(&held.Pages, &held.Findings, &held.Entities, &held.Edges, &held.Runs, &held.Unfinished, &held.Chats)
	return held, err
}

func scanRunID(rows *sql.Rows) (string, error) {
	var runID string
	err := rows.Scan(&runID)
	return runID, err
}

func (r *SiteDataRepo) Counts(ctx context.Context, siteID string) (site.Held, error) {
	return selectOne(ctx, r.store.execFrom(ctx), selectSiteHeld, []any{siteID}, scanHeld,
		errors.New(errors.NotFound, "the site's data could not be counted").WithDetail("siteId", siteID), "count the site's data")
}

func (r *SiteDataRepo) Busy(ctx context.Context, siteID string) ([]string, error) {
	return selectAll(ctx, r.store.execFrom(ctx), selectSiteBusy, []any{siteID}, scanRunID, "list the site's runs under way")
}

func (r *SiteDataRepo) Clear(ctx context.Context, siteID string, what site.Clearing) error {
	return r.store.Do(ctx, func(c context.Context) error {
		exec := r.store.writeFrom(c)
		statements := make([]string, 0, 5)
		if what.History {
			statements = append(statements, deleteSiteRuns, deleteSiteConversations)
		}
		if what.Model {
			statements = append(statements, deleteSiteFindings, deleteSitePages)
		}
		if what.Graph {
			statements = append(statements, deleteSiteTopicBuilds)
		}
		for _, statement := range statements {
			if _, err := execWrite(c, exec, statement, []any{siteID}, nil, "clear the site's data"); err != nil {
				return err
			}
		}
		if !what.Graph {
			return nil
		}
		return r.store.deleteEntitiesLeavesFirst(c, siteID)
	})
}
