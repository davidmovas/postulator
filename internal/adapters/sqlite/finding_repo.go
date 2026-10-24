package sqlite

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/Masterminds/squirrel"

	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

const (
	findingColumns  = `id, site_id, page_id, entity_id, code, severity, subject_key, subject, options, status, resolution, resolved_by, created_at, updated_at`
	insertFinding   = `INSERT INTO findings (` + findingColumns + `) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	selectRaised    = `SELECT id, status, resolved_by, subject FROM findings WHERE site_id = ? AND code = ? AND subject_key = ?`
	raiseAgain      = `UPDATE findings SET severity = ?, subject = ?, options = ?, status = ?, resolution = '', resolved_by = '', updated_at = ? WHERE id = ?`
	clearFinding    = `UPDATE findings SET status = ?, resolution = ?, resolved_by = ?, updated_at = ? WHERE site_id = ? AND code = ? AND subject_key = ? AND status = ?`
	settleFinding   = `UPDATE findings SET status = ?, resolution = ?, resolved_by = ?, updated_at = ? WHERE id = ? AND status = ?`
	moveFinding     = `UPDATE findings SET page_id = ?, updated_at = ? WHERE id = ? AND EXISTS (SELECT 1 FROM pages WHERE pages.id = ? AND pages.site_id = findings.site_id)`
	selectFinding   = `SELECT ` + findingColumns + ` FROM findings WHERE id = ?`
	findingOrder    = ` ORDER BY created_at, id`
	selectFindings  = `SELECT ` + findingColumns + ` FROM findings`
	selectOpenOfOne = selectFindings + ` WHERE site_id = ? AND code = ? AND status = ?` + findingOrder
	selectOfSite    = selectFindings + ` WHERE site_id = ?` + findingOrder
	selectOfStatus  = selectFindings + ` WHERE site_id = ? AND status = ?` + findingOrder
	selectOfPage    = selectFindings + ` WHERE page_id = ?` + findingOrder
	countOpenOf     = `SELECT page_id, count(*) FROM findings WHERE status = 'open' AND page_id IN `
	groupOpenOf     = ` GROUP BY page_id`

	qualifiedFindingColumns = `f.id, f.site_id, f.page_id, f.entity_id, f.code, f.severity, f.subject_key, f.subject, f.options, f.status,
		f.resolution, f.resolved_by, f.created_at, f.updated_at`
)

type FindingRepo struct {
	store *Store
}

func NewFindingRepo(store *Store) *FindingRepo {
	return &FindingRepo{store: store}
}

type raised struct {
	id         string
	status     finding.Status
	resolvedBy string
	subject    string
}

func (r *FindingRepo) Upsert(ctx context.Context, f finding.Finding) error {
	if !f.Code.Valid() {
		return errors.New(errors.Invalid, "finding code is not recognized").WithDetail("code", string(f.Code))
	}
	subject := subjectText(f.Subject)
	options := f.Options
	if options == nil {
		options = []finding.Option{}
	}
	encodedOptions, err := encodeJSON(options)
	if err != nil {
		return err
	}
	return r.store.Do(ctx, func(c context.Context) error {
		rows, selectErr := selectAll(c, r.store.execFrom(c), selectRaised, []any{f.SiteID, string(f.Code), f.SubjectKey}, scanRaised, "read the raised finding")
		if selectErr != nil {
			return selectErr
		}
		if len(rows) == 0 {
			_, insertErr := execWrite(c, r.store.writeFrom(c), insertFinding, []any{
				f.ID, f.SiteID, nullString(f.PageID), nullString(f.EntityID), string(f.Code), string(f.Severity), f.SubjectKey,
				subject, encodedOptions, string(finding.Open), "", "", formatTime(f.CreatedAt), formatTime(f.UpdatedAt),
			}, nil, "raise the finding")
			return insertErr
		}
		existing := rows[0]
		if finding.Settled(existing.status, existing.resolvedBy) && sameJSON(existing.subject, subject) {
			return nil
		}
		_, updateErr := execWrite(c, r.store.writeFrom(c), raiseAgain, []any{
			string(f.Severity), subject, encodedOptions, string(finding.Open), formatTime(f.UpdatedAt), existing.id,
		}, nil, "raise the finding again")
		return updateErr
	})
}

func (r *FindingRepo) Clear(ctx context.Context, siteID string, code finding.Code, subjectKey, resolvedBy, resolution string, at time.Time) error {
	_, err := execWrite(ctx, r.store.writeFrom(ctx), clearFinding, []any{
		string(finding.Resolved), resolution, resolvedBy, formatTime(at), siteID, string(code), subjectKey, string(finding.Open),
	}, nil, "clear the finding")
	return err
}

func (r *FindingRepo) Get(ctx context.Context, id string) (finding.Finding, error) {
	return selectOne(ctx, r.store.execFrom(ctx), selectFinding, []any{id}, scanFinding,
		errors.New(errors.NotFound, "finding not found").WithDetail("findingId", id), "read the finding")
}

func (r *FindingRepo) Settle(ctx context.Context, id string, status finding.Status, resolution, resolvedBy string, at time.Time) error {
	if status != finding.Resolved && status != finding.Dismissed {
		return errors.New(errors.Invalid, "a finding is settled as resolved or dismissed").WithDetail("status", string(status))
	}
	return r.store.Do(ctx, func(c context.Context) error {
		affected, err := execWrite(c, r.store.writeFrom(c), settleFinding, []any{
			string(status), resolution, resolvedBy, formatTime(at), id, string(finding.Open),
		}, nil, "settle the finding")
		if err != nil || affected > 0 {
			return err
		}
		if _, getErr := r.Get(c, id); getErr != nil {
			return getErr
		}
		return errors.New(errors.Conflict, "the finding is no longer open").WithDetail("findingId", id)
	})
}

func (r *FindingRepo) MoveToPage(ctx context.Context, id, pageID string, at time.Time) error {
	affected, err := execWrite(ctx, r.store.writeFrom(ctx), moveFinding, []any{pageID, formatTime(at), id, pageID}, nil,
		"move the finding onto the page")
	return requireAffected(affected, err,
		errors.New(errors.NotFound, "no finding of the page's site has this id").WithDetail("findingId", id).WithDetail("pageId", pageID))
}

func (r *FindingRepo) OpenBySite(ctx context.Context, siteID string, code finding.Code) ([]finding.Finding, error) {
	return selectAll(ctx, r.store.execFrom(ctx), selectOpenOfOne, []any{siteID, string(code), string(finding.Open)}, scanFinding, "list the open findings")
}

func (r *FindingRepo) ListBySite(ctx context.Context, siteID string, status finding.Status) ([]finding.Finding, error) {
	if status == "" {
		return selectAll(ctx, r.store.execFrom(ctx), selectOfSite, []any{siteID}, scanFinding, "list the site findings")
	}
	if !status.Valid() {
		return nil, errors.New(errors.Invalid, "finding status is not recognized").WithDetail("status", string(status))
	}
	return selectAll(ctx, r.store.execFrom(ctx), selectOfStatus, []any{siteID, string(status)}, scanFinding, "list the site findings")
}

func findingKeyset() paging.Keyset[finding.Finding] {
	return paging.Keyset[finding.Finding]{
		IDColumn: "f.id",
		ID:       func(f finding.Finding) string { return f.ID },
		Keys: []paging.SortKey[finding.Finding]{
			paging.TimeKey[finding.Finding]("createdAt", "f.created_at", func(f finding.Finding) any { return f.CreatedAt }),
		},
	}
}

func findingFilter(builder squirrel.SelectBuilder, q finding.Query) squirrel.SelectBuilder {
	builder = builder.From("findings AS f").Where(squirrel.Eq{"f.site_id": q.SiteID})
	if q.Placed() {
		builder = builder.Join("page_map AS m ON m.page_id = f.page_id")
	}
	if q.Status != "" {
		builder = builder.Where(squirrel.Eq{"f.status": string(q.Status)})
	}
	if q.Code != "" {
		builder = builder.Where(squirrel.Eq{"f.code": string(q.Code)})
	}
	if q.PageID != "" {
		builder = builder.Where(squirrel.Eq{"f.page_id": q.PageID})
	}
	if q.PageTypeID != "" {
		builder = builder.Where(squirrel.Eq{"m.type_id": q.PageTypeID})
	}
	if q.Under != "" {
		builder = builder.Where("m.trail >= ? AND m.trail < ?", q.Under, q.Under[:len(q.Under)-1]+trailCeiling)
	}
	return builder
}

func (r *FindingRepo) List(ctx context.Context, q finding.Query, page paging.Request) (paging.List[finding.Finding], error) {
	builder := findingFilter(squirrel.Select(qualifiedFindingColumns), q)
	return selectKeyed(ctx, r.store.execFrom(ctx), builder, findingKeyset(), page, scanFinding, "findings")
}

func (r *FindingRepo) CountByCode(ctx context.Context, q finding.Query) (map[finding.Code]int, error) {
	query, args, err := buildQuery(findingFilter(squirrel.Select("f.code", "count(*)"), q).GroupBy("f.code"), "finding counts")
	if err != nil {
		return nil, err
	}
	listed, err := selectAll(ctx, r.store.execFrom(ctx), query, args, scanCounted, "count the findings")
	if err != nil {
		return nil, err
	}
	out := make(map[finding.Code]int, len(listed))
	for i := range listed {
		out[finding.Code(listed[i].id)] = listed[i].count
	}
	return out, nil
}

func (r *FindingRepo) CountOpen(ctx context.Context, pageIDs []string) (map[string]int, error) {
	return countByIDs(ctx, r.store.execFrom(ctx), countOpenOf, groupOpenOf, pageIDs, "count the open findings of the pages")
}

func (r *FindingRepo) ListForPage(ctx context.Context, pageID string) ([]finding.Finding, error) {
	return selectAll(ctx, r.store.execFrom(ctx), selectOfPage, []any{pageID}, scanFinding, "list the page findings")
}

func subjectText(subject json.RawMessage) string {
	if len(subject) == 0 {
		return "{}"
	}
	return string(subject)
}

func sameJSON(stored, raised string) bool {
	var left, right bytes.Buffer
	if json.Compact(&left, []byte(stored)) != nil || json.Compact(&right, []byte(raised)) != nil {
		return stored == raised
	}
	return bytes.Equal(left.Bytes(), right.Bytes())
}

func scanRaised(rows *sql.Rows) (raised, error) {
	var (
		row    raised
		status string
	)
	if err := rows.Scan(&row.id, &status, &row.resolvedBy, &row.subject); err != nil {
		return raised{}, err
	}
	row.status = finding.Status(status)
	return row, nil
}

func scanFinding(rows *sql.Rows) (finding.Finding, error) {
	var (
		f                    finding.Finding
		pageID, entityID     sql.NullString
		code, severity       string
		subject, options     string
		status               string
		createdAt, updatedAt string
	)
	if err := rows.Scan(&f.ID, &f.SiteID, &pageID, &entityID, &code, &severity, &f.SubjectKey, &subject, &options,
		&status, &f.Resolution, &f.ResolvedBy, &createdAt, &updatedAt); err != nil {
		return finding.Finding{}, err
	}
	f.PageID = optString(pageID)
	f.EntityID = optString(entityID)
	f.Code = finding.Code(code)
	f.Severity = finding.Severity(severity)
	f.Subject = json.RawMessage(subject)
	f.Status = finding.Status(status)
	if err := decodeJSON(options, &f.Options, "decode the finding options"); err != nil {
		return finding.Finding{}, err
	}

	var err error
	if f.CreatedAt, err = parseTime(createdAt); err != nil {
		return finding.Finding{}, err
	}
	if f.UpdatedAt, err = parseTime(updatedAt); err != nil {
		return finding.Finding{}, err
	}
	return f, nil
}
