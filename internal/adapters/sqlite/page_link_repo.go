package sqlite

import (
	"context"
	"database/sql"

	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	linkColumns        = `id, site_id, from_page_id, to_page_id, to_url, anchor_text, origin, type, region, source, position, ours, observed_at`
	insertLink         = `INSERT INTO page_links (` + linkColumns + `) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	deleteLinksForPage = `DELETE FROM page_links WHERE from_page_id = ? AND source = ?`
	selectLinksForPage = `SELECT ` + linkColumns + ` FROM page_links WHERE from_page_id = ? ORDER BY observed_at, id`
	selectLinksBySite  = `SELECT ` + linkColumns + ` FROM page_links WHERE site_id = ? ORDER BY from_page_id, id`
)

type PageLinkRepo struct {
	store *Store
}

func NewPageLinkRepo(store *Store) *PageLinkRepo {
	return &PageLinkRepo{store: store}
}

func (r *PageLinkRepo) ReplaceForPage(ctx context.Context, pageID string, source pagemap.LinkSource, links []pagemap.PageLink) error {
	if !source.Valid() {
		return errors.New(errors.Invalid, "link source is not recognized").WithDetail("source", string(source))
	}
	for i := range links {
		if links[i].FromPageID != pageID {
			return errors.New(errors.Invalid, "every link must start from the page being replaced").
				WithDetail("pageId", pageID).WithDetail("linkId", links[i].ID)
		}
		if links[i].Source != source {
			return errors.New(errors.Invalid, "every link must come from the source being replaced").
				WithDetail("source", string(source)).WithDetail("linkId", links[i].ID)
		}
	}
	return r.store.Do(ctx, func(c context.Context) error {
		if _, err := execWrite(c, r.store.writeFrom(c), deleteLinksForPage, []any{pageID, string(source)}, nil, "clear the page links"); err != nil {
			return err
		}
		for i := range links {
			link := &links[i]
			if _, err := execWrite(c, r.store.writeFrom(c), insertLink, []any{
				link.ID, link.SiteID, link.FromPageID, nullString(link.ToPageID), link.ToURL, link.AnchorText, string(link.Origin),
				string(link.Type), link.Region, string(link.Source), nullPosition(link.Position), boolInt(link.Ours), formatTime(link.ObservedAt),
			}, nil, "insert a page link"); err != nil {
				return err
			}
		}
		return nil
	})
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
		linkType   string
		source     string
		position   sql.NullInt64
		ours       int64
		observedAt string
	)
	if err := rows.Scan(&l.ID, &l.SiteID, &l.FromPageID, &toPageID, &l.ToURL, &l.AnchorText, &origin,
		&linkType, &l.Region, &source, &position, &ours, &observedAt); err != nil {
		return pagemap.PageLink{}, err
	}
	l.ToPageID = optString(toPageID)
	l.Origin = pagemap.LinkOrigin(origin)
	l.Type = pagemap.LinkType(linkType)
	l.Source = pagemap.LinkSource(source)
	l.Position = optPosition(position)
	l.Ours = ours == 1

	var err error
	if l.ObservedAt, err = parseTime(observedAt); err != nil {
		return pagemap.PageLink{}, err
	}
	return l, nil
}

func nullPosition(position *int) any {
	if position == nil {
		return nil
	}
	return int64(*position)
}

func optPosition(raw sql.NullInt64) *int {
	if !raw.Valid {
		return nil
	}
	value := int(raw.Int64)
	return &value
}
