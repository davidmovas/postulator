package sqlite

import (
	"context"
	"database/sql"

	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	linkColumns        = `id, site_id, from_page_id, to_page_id, to_url, anchor_text, origin, observed_at`
	insertLink         = `INSERT INTO page_links (` + linkColumns + `) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	deleteLinksForPage = `DELETE FROM page_links WHERE from_page_id = ?`
	selectLinksForPage = `SELECT ` + linkColumns + ` FROM page_links WHERE from_page_id = ? ORDER BY observed_at, id`
	selectLinksBySite  = `SELECT ` + linkColumns + ` FROM page_links WHERE site_id = ? ORDER BY from_page_id, id`
)

type PageLinkRepo struct {
	store *Store
}

func NewPageLinkRepo(store *Store) *PageLinkRepo {
	return &PageLinkRepo{store: store}
}

func (r *PageLinkRepo) ReplaceForPage(ctx context.Context, pageID string, links []pagemap.PageLink) error {
	for i := range links {
		if links[i].FromPageID != pageID {
			return errors.New(errors.Invalid, "every link must start from the page being replaced").
				WithDetail("pageId", pageID).WithDetail("linkId", links[i].ID)
		}
	}
	if _, err := execWrite(ctx, r.store.writeFrom(ctx), deleteLinksForPage, []any{pageID}, nil, "clear the page links"); err != nil {
		return err
	}
	for i := range links {
		link := &links[i]
		if _, err := execWrite(ctx, r.store.writeFrom(ctx), insertLink, []any{
			link.ID, link.SiteID, link.FromPageID, nullString(link.ToPageID), link.ToURL, link.AnchorText, string(link.Origin), formatTime(link.ObservedAt),
		}, nil, "insert a page link"); err != nil {
			return err
		}
	}
	return nil
}

func (r *PageLinkRepo) ListForPage(ctx context.Context, pageID string) ([]pagemap.PageLink, error) {
	return selectAll(ctx, r.store.execFrom(ctx), selectLinksForPage, []any{pageID}, scanPageLink, "list the page links")
}

func (r *PageLinkRepo) ListBySite(ctx context.Context, siteID string) ([]pagemap.PageLink, error) {
	return selectAll(ctx, r.store.execFrom(ctx), selectLinksBySite, []any{siteID}, scanPageLink, "list the site links")
}

func scanPageLink(rows *sql.Rows) (pagemap.PageLink, error) {
	var (
		l          pagemap.PageLink
		toPageID   sql.NullString
		origin     string
		observedAt string
	)
	if err := rows.Scan(&l.ID, &l.SiteID, &l.FromPageID, &toPageID, &l.ToURL, &l.AnchorText, &origin, &observedAt); err != nil {
		return pagemap.PageLink{}, err
	}
	l.ToPageID = optString(toPageID)
	l.Origin = pagemap.LinkOrigin(origin)

	var err error
	if l.ObservedAt, err = parseTime(observedAt); err != nil {
		return pagemap.PageLink{}, err
	}
	return l, nil
}
