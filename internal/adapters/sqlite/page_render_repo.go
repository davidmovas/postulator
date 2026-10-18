package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/render"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	renderColumns = `page_id, render_hash, content_root, regions, jsonld, trail, headings, word_count_content, word_count_generated, builder, url_form, robots, final_path, extracted_at`
	upsertRender  = `INSERT INTO page_renders (` + renderColumns + `) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (page_id) DO UPDATE SET render_hash = excluded.render_hash, content_root = excluded.content_root, regions = excluded.regions,
		jsonld = excluded.jsonld, trail = excluded.trail, headings = excluded.headings, word_count_content = excluded.word_count_content,
		word_count_generated = excluded.word_count_generated, builder = excluded.builder, url_form = excluded.url_form, robots = excluded.robots,
		final_path = excluded.final_path, extracted_at = excluded.extracted_at`
	selectRender        = `SELECT ` + renderColumns + ` FROM page_renders WHERE page_id = ?`
	selectRendersBySite = `SELECT r.page_id, r.render_hash, r.content_root, r.regions, r.jsonld, r.trail, r.headings, r.word_count_content,
		r.word_count_generated, r.builder, r.url_form, r.robots, r.final_path, r.extracted_at
		FROM page_renders AS r JOIN pages AS p ON p.id = r.page_id WHERE p.site_id = ? ORDER BY r.page_id`
)

type PageRenderRepo struct {
	store *Store
}

func NewPageRenderRepo(store *Store) *PageRenderRepo {
	return &PageRenderRepo{store: store}
}

func renderNotFound(pageID string) *errors.Error {
	return errors.New(errors.NotFound, "the page has no extracted render").WithDetail("pageId", pageID)
}

type renderJSON struct {
	regions, jsonld, trails, headings string
}

func encodeRender(e *render.Extraction) (renderJSON, error) {
	regions := e.Regions
	if regions == nil {
		regions = []render.Region{}
	}
	jsonld := e.JSONLD
	if jsonld == nil {
		jsonld = map[string][]json.RawMessage{}
	}
	trails := e.Trails
	if trails == nil {
		trails = []pagemap.Trail{}
	}
	headings := e.Headings
	if headings == nil {
		headings = []render.Heading{}
	}

	var (
		out renderJSON
		err error
	)
	if out.regions, err = encodeJSON(regions); err != nil {
		return renderJSON{}, err
	}
	if out.jsonld, err = encodeJSON(jsonld); err != nil {
		return renderJSON{}, err
	}
	if out.trails, err = encodeJSON(trails); err != nil {
		return renderJSON{}, err
	}
	if out.headings, err = encodeJSON(headings); err != nil {
		return renderJSON{}, err
	}
	return out, nil
}

func (r *PageRenderRepo) Put(ctx context.Context, record render.Record) error {
	encoded, err := encodeRender(&record.Extraction)
	if err != nil {
		return err
	}
	_, err = execWrite(ctx, r.store.writeFrom(ctx), upsertRender, []any{
		record.PageID, record.Hash, string(record.Root), encoded.regions, encoded.jsonld, encoded.trails, encoded.headings,
		record.WordsContent, record.WordsGenerated, record.Builder, record.URLForm, record.Robots, record.FinalPath, formatTime(record.ExtractedAt),
	}, nil, "store the page render")
	return err
}

func (r *PageRenderRepo) Get(ctx context.Context, pageID string) (render.Record, error) {
	return selectOne(ctx, r.store.execFrom(ctx), selectRender, []any{pageID}, scanRender, renderNotFound(pageID), "read the page render")
}

func (r *PageRenderRepo) ListBySite(ctx context.Context, siteID string) ([]render.Record, error) {
	return selectAll(ctx, r.store.execFrom(ctx), selectRendersBySite, []any{siteID}, scanRender, "list the site renders")
}

func scanRender(rows *sql.Rows) (render.Record, error) {
	var (
		record                            render.Record
		root                              string
		regions, jsonld, trails, headings string
		extractedAt                       string
	)
	if err := rows.Scan(&record.PageID, &record.Hash, &root, &regions, &jsonld, &trails, &headings,
		&record.WordsContent, &record.WordsGenerated, &record.Builder, &record.URLForm, &record.Robots, &record.FinalPath, &extractedAt); err != nil {
		return render.Record{}, err
	}
	record.Root = render.RootState(root)
	for _, column := range []struct {
		raw     string
		into    any
		message string
	}{
		{raw: regions, into: &record.Regions, message: "decode the render regions"},
		{raw: jsonld, into: &record.JSONLD, message: "decode the render json-ld"},
		{raw: trails, into: &record.Trails, message: "decode the render trails"},
		{raw: headings, into: &record.Headings, message: "decode the render headings"},
	} {
		if err := decodeJSON(column.raw, column.into, column.message); err != nil {
			return render.Record{}, err
		}
	}

	var err error
	if record.ExtractedAt, err = parseTime(extractedAt); err != nil {
		return render.Record{}, err
	}
	return record, nil
}
