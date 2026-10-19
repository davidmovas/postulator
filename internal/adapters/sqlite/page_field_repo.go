package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	fieldColumns = `page_id, field_key, current_value, current_hash, base_value, base_hash, observed_at, written_at, written_run_id`
	observeField = `INSERT INTO page_fields (page_id, field_key, current_value, current_hash, base_value, base_hash, observed_at) VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (page_id, field_key) DO UPDATE SET current_value = excluded.current_value, current_hash = excluded.current_hash, observed_at = excluded.observed_at`
	recordField = `INSERT INTO page_fields (` + fieldColumns + `) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (page_id, field_key) DO UPDATE SET current_value = excluded.current_value, current_hash = excluded.current_hash,
		base_value = excluded.base_value, base_hash = excluded.base_hash, written_at = excluded.written_at, written_run_id = excluded.written_run_id`
	agreeField          = `UPDATE page_fields SET base_value = current_value, base_hash = current_hash, observed_at = ? WHERE page_id = ? AND field_key = ?`
	selectFieldsForPage = `SELECT ` + fieldColumns + ` FROM page_fields WHERE page_id = ? ORDER BY field_key`
	selectFieldsBySite  = `SELECT f.page_id, f.field_key, f.current_value, f.current_hash, f.base_value, f.base_hash, f.observed_at, f.written_at, f.written_run_id
		FROM page_fields AS f JOIN pages AS p ON p.id = f.page_id WHERE p.site_id = ? ORDER BY f.page_id, f.field_key`
	selectFieldsForPages = `SELECT ` + fieldColumns + ` FROM page_fields WHERE page_id IN `
	orderFields          = ` ORDER BY page_id, field_key`
)

type PageFieldRepo struct {
	store *Store
}

func NewPageFieldRepo(store *Store) *PageFieldRepo {
	return &PageFieldRepo{store: store}
}

func (r *PageFieldRepo) Observe(ctx context.Context, pageID string, values []field.Value) error {
	for i := range values {
		if values[i].PageID != pageID {
			return errors.New(errors.Invalid, "every field value must belong to the page being observed").
				WithDetail("pageId", pageID).WithDetail("fieldKey", string(values[i].Key))
		}
	}
	return r.store.Do(ctx, func(c context.Context) error {
		for i := range values {
			value := &values[i]
			if _, err := execWrite(c, r.store.writeFrom(c), observeField, []any{
				pageID, string(value.Key), value.Current, value.CurrentHash, value.Current, value.CurrentHash, formatTime(value.ObservedAt),
			}, nil, "observe a page field"); err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *PageFieldRepo) Record(ctx context.Context, pageID string, key field.Key, value, runID string, at time.Time) error {
	hash := field.Hash(value)
	_, err := execWrite(ctx, r.store.writeFrom(ctx), recordField, []any{
		pageID, string(key), value, hash, value, hash, formatTime(at), formatTime(at), runID,
	}, nil, "record a written page field")
	return err
}

func fieldNotFound(pageID string, key field.Key) *errors.Error {
	return errors.New(errors.NotFound, "the page has no observed value for this field").
		WithDetail("pageId", pageID).WithDetail("fieldKey", string(key))
}

func (r *PageFieldRepo) Agree(ctx context.Context, pageID string, key field.Key, at time.Time) error {
	affected, err := execWrite(ctx, r.store.writeFrom(ctx), agreeField, []any{formatTime(at), pageID, string(key)}, nil, "agree the page field")
	return requireAffected(affected, err, fieldNotFound(pageID, key))
}

func (r *PageFieldRepo) ListForPage(ctx context.Context, pageID string) ([]field.Value, error) {
	return selectAll(ctx, r.store.execFrom(ctx), selectFieldsForPage, []any{pageID}, scanFieldValue, "list the page fields")
}

func (r *PageFieldRepo) ListBySite(ctx context.Context, siteID string) (map[string][]field.Value, error) {
	values, err := selectAll(ctx, r.store.execFrom(ctx), selectFieldsBySite, []any{siteID}, scanFieldValue, "list the site fields")
	if err != nil {
		return nil, err
	}
	return fieldsByPage(values), nil
}

func (r *PageFieldRepo) ListForPages(ctx context.Context, pageIDs []string) (map[string][]field.Value, error) {
	values, err := selectByIDs(ctx, r.store.execFrom(ctx), selectFieldsForPages, orderFields, pageIDs, scanFieldValue, "list the fields of the pages")
	if err != nil {
		return nil, err
	}
	return fieldsByPage(values), nil
}

func fieldsByPage(values []field.Value) map[string][]field.Value {
	byPage := make(map[string][]field.Value)
	for i := range values {
		byPage[values[i].PageID] = append(byPage[values[i].PageID], values[i])
	}
	return byPage
}

func scanFieldValue(rows *sql.Rows) (field.Value, error) {
	var (
		value        field.Value
		key          string
		observedAt   string
		writtenAt    sql.NullString
		writtenRunID sql.NullString
	)
	if err := rows.Scan(&value.PageID, &key, &value.Current, &value.CurrentHash, &value.Base, &value.BaseHash,
		&observedAt, &writtenAt, &writtenRunID); err != nil {
		return field.Value{}, err
	}
	value.Key = field.Key(key)
	value.WrittenRunID = optString(writtenRunID)

	var err error
	if value.ObservedAt, err = parseTime(observedAt); err != nil {
		return field.Value{}, err
	}
	if value.WrittenAt, err = parseNullTime(writtenAt); err != nil {
		return field.Value{}, err
	}
	return value, nil
}
