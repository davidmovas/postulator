package sqlite

import (
	"context"
	"database/sql"

	"github.com/davidmovas/postulator/internal/domain/importmap"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	importMappingColumns = `id, site_id, name, columns, options, created_at, updated_at`
	upsertImportMapping  = `INSERT INTO import_mappings (` + importMappingColumns + `) VALUES (?, ?, ?, ?, ?, ?, ?) ` +
		`ON CONFLICT (id) DO UPDATE SET name = excluded.name, columns = excluded.columns, options = excluded.options, updated_at = excluded.updated_at`
	deleteImportMapping        = `DELETE FROM import_mappings WHERE id = ?`
	selectImportMapping        = `SELECT ` + importMappingColumns + ` FROM import_mappings WHERE id = ?`
	selectImportMappingsBySite = `SELECT ` + importMappingColumns + ` FROM import_mappings WHERE site_id = ? ORDER BY name, id`
)

type ImportMappingRepo struct {
	store *Store
}

func NewImportMappingRepo(store *Store) *ImportMappingRepo {
	return &ImportMappingRepo{store: store}
}

func importMappingNotFound(id string) *errors.Error {
	return errors.New(errors.NotFound, "import mapping not found").WithDetail("mappingId", id)
}

func importMappingConflict(name string) *errors.Error {
	return errors.New(errors.Conflict, "an import mapping with this name already exists in the site").WithDetail("name", name)
}

func (r *ImportMappingRepo) Upsert(ctx context.Context, m importmap.Mapping) error {
	columns, err := encodeJSON(textColumns(m.Columns))
	if err != nil {
		return err
	}
	options, err := encodeJSON(m.Options)
	if err != nil {
		return err
	}
	_, err = execWrite(ctx, r.store.writeFrom(ctx), upsertImportMapping, []any{
		m.ID, m.SiteID, m.Name, columns, options, formatTime(m.CreatedAt), formatTime(m.UpdatedAt),
	}, importMappingConflict(m.Name), "save the import mapping")
	return err
}

func (r *ImportMappingRepo) Delete(ctx context.Context, id string) error {
	affected, err := execWrite(ctx, r.store.writeFrom(ctx), deleteImportMapping, []any{id}, nil, "delete the import mapping")
	return requireAffected(affected, err, importMappingNotFound(id))
}

func (r *ImportMappingRepo) Get(ctx context.Context, id string) (importmap.Mapping, error) {
	return selectOne(ctx, r.store.execFrom(ctx), selectImportMapping, []any{id}, scanImportMapping, importMappingNotFound(id), "read the import mapping")
}

func (r *ImportMappingRepo) ListBySite(ctx context.Context, siteID string) ([]importmap.Mapping, error) {
	return selectAll(ctx, r.store.execFrom(ctx), selectImportMappingsBySite, []any{siteID}, scanImportMapping, "list the import mappings")
}

func textColumns(columns map[importmap.Field]string) map[string]string {
	out := make(map[string]string, len(columns))
	for field, column := range columns {
		out[string(field)] = column
	}
	return out
}

func scanImportMapping(rows *sql.Rows) (importmap.Mapping, error) {
	var (
		m                    importmap.Mapping
		columns, options     string
		createdAt, updatedAt string
	)
	if err := rows.Scan(&m.ID, &m.SiteID, &m.Name, &columns, &options, &createdAt, &updatedAt); err != nil {
		return importmap.Mapping{}, err
	}

	decoded := make(map[string]string)
	if err := decodeJSON(columns, &decoded, "decode the import mapping columns"); err != nil {
		return importmap.Mapping{}, err
	}
	m.Columns = make(map[importmap.Field]string, len(decoded))
	for field, column := range decoded {
		m.Columns[importmap.Field(field)] = column
	}
	if err := decodeJSON(options, &m.Options, "decode the import mapping options"); err != nil {
		return importmap.Mapping{}, err
	}

	var err error
	if m.CreatedAt, err = parseTime(createdAt); err != nil {
		return importmap.Mapping{}, err
	}
	if m.UpdatedAt, err = parseTime(updatedAt); err != nil {
		return importmap.Mapping{}, err
	}
	return m, nil
}
