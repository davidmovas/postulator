package sqlite

import (
	"context"
	"database/sql"

	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	identityColumns = `page_id, url, status, wp_type, wp_parent_id, modified_gmt, observed_at`
	upsertIdentity  = `INSERT INTO page_identity (` + identityColumns + `) VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (page_id) DO UPDATE SET url = excluded.url, status = excluded.status, wp_type = excluded.wp_type,
		wp_parent_id = excluded.wp_parent_id, modified_gmt = excluded.modified_gmt, observed_at = excluded.observed_at`
	selectIdentity         = `SELECT ` + identityColumns + ` FROM page_identity WHERE page_id = ?`
	selectIdentitiesBySite = `SELECT i.page_id, i.url, i.status, i.wp_type, i.wp_parent_id, i.modified_gmt, i.observed_at
		FROM page_identity AS i JOIN pages AS p ON p.id = i.page_id WHERE p.site_id = ? ORDER BY i.page_id`
	selectIdentitiesFor = `SELECT ` + identityColumns + ` FROM page_identity WHERE page_id IN `
	orderIdentities     = ` ORDER BY page_id`

	termColumns        = `taxonomy, term_id, name, slug, parent_term_id`
	deleteTermsForPage = `DELETE FROM page_terms WHERE page_id = ?`
	insertTerm         = `INSERT INTO page_terms (page_id, ` + termColumns + `) VALUES (?, ?, ?, ?, ?, ?)`
	selectTermsForPage = `SELECT ` + termColumns + ` FROM page_terms WHERE page_id = ? ORDER BY taxonomy, term_id`
	selectTermsBySite  = `SELECT t.page_id, t.taxonomy, t.term_id, t.name, t.slug, t.parent_term_id
		FROM page_terms AS t JOIN pages AS p ON p.id = t.page_id WHERE p.site_id = ? ORDER BY t.page_id, t.taxonomy, t.term_id`
)

type PageIdentityRepo struct {
	store *Store
}

func NewPageIdentityRepo(store *Store) *PageIdentityRepo {
	return &PageIdentityRepo{store: store}
}

func identityNotFound(pageID string) *errors.Error {
	return errors.New(errors.NotFound, "the page has no observed identity").WithDetail("pageId", pageID)
}

func (r *PageIdentityRepo) Put(ctx context.Context, identity pagemap.Identity) error {
	_, err := execWrite(ctx, r.store.writeFrom(ctx), upsertIdentity, []any{
		identity.PageID, identity.URL, identity.Status, string(identity.WPType), nullInt(identity.WPParentID),
		nullTime(identity.ModifiedGMT), formatTime(identity.ObservedAt),
	}, nil, "store the page identity")
	return err
}

func (r *PageIdentityRepo) Get(ctx context.Context, pageID string) (pagemap.Identity, error) {
	return selectOne(ctx, r.store.execFrom(ctx), selectIdentity, []any{pageID}, scanIdentity, identityNotFound(pageID), "read the page identity")
}

func (r *PageIdentityRepo) ListBySite(ctx context.Context, siteID string) ([]pagemap.Identity, error) {
	return selectAll(ctx, r.store.execFrom(ctx), selectIdentitiesBySite, []any{siteID}, scanIdentity, "list the page identities")
}

func (r *PageIdentityRepo) ListForPages(ctx context.Context, pageIDs []string) ([]pagemap.Identity, error) {
	return selectByIDs(ctx, r.store.execFrom(ctx), selectIdentitiesFor, orderIdentities, pageIDs, scanIdentity, "list the identities of the pages")
}

func (r *PageIdentityRepo) ReplaceTerms(ctx context.Context, pageID string, terms []pagemap.Term) error {
	return r.store.Do(ctx, func(c context.Context) error {
		if _, err := execWrite(c, r.store.writeFrom(c), deleteTermsForPage, []any{pageID}, nil, "clear the page terms"); err != nil {
			return err
		}
		for i := range terms {
			term := &terms[i]
			if _, err := execWrite(c, r.store.writeFrom(c), insertTerm, []any{
				pageID, term.Taxonomy, term.TermID, term.Name, term.Slug, term.ParentTermID,
			}, termConflict(pageID, term), "insert a page term"); err != nil {
				return err
			}
		}
		return nil
	})
}

func termConflict(pageID string, term *pagemap.Term) *errors.Error {
	return errors.New(errors.Conflict, "the page carries this term once").
		WithDetail("pageId", pageID).WithDetail("taxonomy", term.Taxonomy).WithDetail("termId", term.TermID)
}

func (r *PageIdentityRepo) ListTerms(ctx context.Context, pageID string) ([]pagemap.Term, error) {
	return selectAll(ctx, r.store.execFrom(ctx), selectTermsForPage, []any{pageID}, scanTerm, "list the page terms")
}

func (r *PageIdentityRepo) ListTermsBySite(ctx context.Context, siteID string) (map[string][]pagemap.Term, error) {
	rows, err := selectAll(ctx, r.store.execFrom(ctx), selectTermsBySite, []any{siteID}, scanPageTerm, "list the site terms")
	if err != nil {
		return nil, err
	}
	byPage := make(map[string][]pagemap.Term)
	for i := range rows {
		byPage[rows[i].pageID] = append(byPage[rows[i].pageID], rows[i].term)
	}
	return byPage, nil
}

type pageTerm struct {
	pageID string
	term   pagemap.Term
}

func scanIdentity(rows *sql.Rows) (pagemap.Identity, error) {
	var (
		identity    pagemap.Identity
		wpType      string
		wpParentID  sql.NullInt64
		modifiedGMT sql.NullString
		observedAt  string
	)
	if err := rows.Scan(&identity.PageID, &identity.URL, &identity.Status, &wpType, &wpParentID, &modifiedGMT, &observedAt); err != nil {
		return pagemap.Identity{}, err
	}
	identity.WPType = pagemap.WPType(wpType)
	identity.WPParentID = optInt(wpParentID)

	var err error
	if identity.ModifiedGMT, err = parseNullTime(modifiedGMT); err != nil {
		return pagemap.Identity{}, err
	}
	if identity.ObservedAt, err = parseTime(observedAt); err != nil {
		return pagemap.Identity{}, err
	}
	return identity, nil
}

func scanTerm(rows *sql.Rows) (pagemap.Term, error) {
	var term pagemap.Term
	if err := rows.Scan(&term.Taxonomy, &term.TermID, &term.Name, &term.Slug, &term.ParentTermID); err != nil {
		return pagemap.Term{}, err
	}
	return term, nil
}

func scanPageTerm(rows *sql.Rows) (pageTerm, error) {
	var row pageTerm
	if err := rows.Scan(&row.pageID, &row.term.Taxonomy, &row.term.TermID, &row.term.Name, &row.term.Slug, &row.term.ParentTermID); err != nil {
		return pageTerm{}, err
	}
	return row, nil
}
