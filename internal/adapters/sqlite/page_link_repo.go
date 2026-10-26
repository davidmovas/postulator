package sqlite

import (
	"context"
	"database/sql"

	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	linkColumns        = `id, site_id, from_page_id, to_page_id, to_url, anchor_text, type, region, source, position, ours, block_hash, absolute, observed_at`
	insertLink         = `INSERT INTO page_links (` + linkColumns + `) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	deleteLinksForPage = `DELETE FROM page_links WHERE from_page_id = ? AND source = ?`
	selectLinksForPage = `SELECT ` + linkColumns + ` FROM page_links WHERE from_page_id = ? ORDER BY observed_at, id`
	selectLinksBySite  = `SELECT ` + linkColumns + ` FROM page_links WHERE site_id = ? ORDER BY from_page_id, id`
	selectLinksFrom    = `SELECT ` + linkColumns + ` FROM page_links WHERE from_page_id IN `
	orderLinksFrom     = ` ORDER BY from_page_id, observed_at, id`
	selectLinksToPage  = `SELECT ` + linkColumns + ` FROM page_links WHERE to_page_id = ? ORDER BY from_page_id, id`
	selectContextual   = `SELECT ` + linkColumns + ` FROM page_links WHERE site_id = ? AND type = 'contextual' AND source = 'raw'
		ORDER BY from_page_id, id`
	countContextualIn  = `SELECT to_page_id, count(*) FROM page_links WHERE type = 'contextual' AND from_page_id <> to_page_id AND to_page_id IN `
	groupContextualIn  = ` GROUP BY to_page_id`
	countContextualOut = `SELECT from_page_id, count(*) FROM page_links WHERE type = 'contextual' AND from_page_id IN `
	groupContextualOut = ` GROUP BY from_page_id`
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
				link.ID, link.SiteID, link.FromPageID, nullString(link.ToPageID), link.ToURL, link.AnchorText,
				string(link.Type), link.Region, string(link.Source), nullPosition(link.Position), boolInt(link.Ours),
				nullText(link.BlockHash), boolInt(link.Absolute), formatTime(link.ObservedAt),
			}, nil, "insert a page link"); err != nil {
				return err
			}
		}
		return nil
	})
}

const selectListedProducts = `SELECT l.from_page_id, count(DISTINCT l.to_page_id) FROM page_links AS l
		JOIN pages AS p ON p.id = l.to_page_id
		WHERE l.site_id = ? AND l.type = 'listing' AND p.wp_type = 'product' GROUP BY l.from_page_id ORDER BY l.from_page_id`

type listedRow struct {
	pageID   string
	products int
}

func (r *PageLinkRepo) ListedProductsBySite(ctx context.Context, siteID string) (map[string]int, error) {
	rows, err := selectAll(ctx, r.store.execFrom(ctx), selectListedProducts, []any{siteID}, func(rows *sql.Rows) (listedRow, error) {
		var row listedRow
		err := rows.Scan(&row.pageID, &row.products)
		return row, err
	}, "count the products each page lists")
	if err != nil {
		return nil, err
	}
	out := make(map[string]int, len(rows))
	for _, row := range rows {
		out[row.pageID] = row.products
	}
	return out, nil
}

func (r *PageLinkRepo) CountContextual(ctx context.Context, pageIDs []string) (inbound, outbound map[string]int, err error) {
	if inbound, err = countByIDs(ctx, r.store.execFrom(ctx), countContextualIn, groupContextualIn, pageIDs, "count the links into the pages"); err != nil {
		return nil, nil, err
	}
	outbound, err = countByIDs(ctx, r.store.execFrom(ctx), countContextualOut, groupContextualOut, pageIDs, "count the links out of the pages")
	return inbound, outbound, err
}

func (r *PageLinkRepo) ListForPage(ctx context.Context, pageID string) ([]pagemap.PageLink, error) {
	return selectAll(ctx, r.store.execFrom(ctx), selectLinksForPage, []any{pageID}, scanPageLink, "list the page links")
}

func (r *PageLinkRepo) ListBySite(ctx context.Context, siteID string) ([]pagemap.PageLink, error) {
	return selectAll(ctx, r.store.execFrom(ctx), selectLinksBySite, []any{siteID}, scanPageLink, "list the site links")
}

func (r *PageLinkRepo) ListFromPages(ctx context.Context, pageIDs []string) ([]pagemap.PageLink, error) {
	return selectByIDs(ctx, r.store.execFrom(ctx), selectLinksFrom, orderLinksFrom, pageIDs, scanPageLink, "list the links of the pages")
}

func (r *PageLinkRepo) ListContextual(ctx context.Context, siteID string) ([]pagemap.PageLink, error) {
	return selectAll(ctx, r.store.execFrom(ctx), selectContextual, []any{siteID}, scanPageLink, "list the site's contextual links")
}

func (r *PageLinkRepo) ListToPage(ctx context.Context, pageID string) ([]pagemap.PageLink, error) {
	return selectAll(ctx, r.store.execFrom(ctx), selectLinksToPage, []any{pageID}, scanPageLink, "list the links to the page")
}

func scanPageLink(rows *sql.Rows) (pagemap.PageLink, error) {
	var (
		l          pagemap.PageLink
		toPageID   sql.NullString
		linkType   string
		source     string
		position   sql.NullInt64
		ours       int64
		blockHash  sql.NullString
		absolute   int64
		observedAt string
	)
	if err := rows.Scan(&l.ID, &l.SiteID, &l.FromPageID, &toPageID, &l.ToURL, &l.AnchorText,
		&linkType, &l.Region, &source, &position, &ours, &blockHash, &absolute, &observedAt); err != nil {
		return pagemap.PageLink{}, err
	}
	l.BlockHash = blockHash.String
	l.Absolute = absolute == 1
	l.ToPageID = optString(toPageID)
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
