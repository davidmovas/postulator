package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/davidmovas/postulator/internal/domain/block"
)

const (
	bodyColumns = `page_id, side, raw, hash, kind, observed_at`
	upsertBody  = `INSERT INTO page_bodies (` + bodyColumns + `) VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT (page_id, side) DO UPDATE SET raw = excluded.raw, hash = excluded.hash, kind = excluded.kind, observed_at = excluded.observed_at`
	insertBodyIfAbsent = `INSERT INTO page_bodies (` + bodyColumns + `) VALUES (?, ?, ?, ?, ?, ?) ON CONFLICT (page_id, side) DO NOTHING`
	selectBodies       = `SELECT ` + bodyColumns + ` FROM page_bodies WHERE page_id = ?`
	selectBodiesBySite = `SELECT b.page_id, b.side, b.raw, b.hash, b.kind, b.observed_at
		FROM page_bodies AS b JOIN pages AS p ON p.id = b.page_id WHERE p.site_id = ? ORDER BY b.page_id, b.side`
	selectCurrentBodies = `SELECT ` + bodyColumns + ` FROM page_bodies WHERE side = 'current' AND page_id IN `
	orderBodies         = ` ORDER BY page_id`
	selectDrifted       = `SELECT c.page_id FROM page_bodies AS c
		JOIN page_bodies AS b ON b.page_id = c.page_id AND b.side = 'base'
		JOIN pages AS p ON p.id = c.page_id
		WHERE c.side = 'current' AND p.site_id = ? AND c.hash <> b.hash ORDER BY c.page_id`

	deleteWritten = `DELETE FROM page_blocks_written WHERE page_id = ?`
	insertWritten = `INSERT INTO page_blocks_written (page_id, block_hash, run_id, written_at) VALUES (?, ?, ?, ?)
		ON CONFLICT (page_id, block_hash) DO NOTHING`
	selectWritten    = `SELECT page_id, block_hash, run_id, written_at FROM page_blocks_written WHERE page_id = ? ORDER BY block_hash`
	selectWrittenFor = `SELECT page_id, block_hash, run_id, written_at FROM page_blocks_written WHERE page_id IN `
	orderWritten     = ` ORDER BY page_id, block_hash`
)

type PageBodyRepo struct {
	store *Store
}

func NewPageBodyRepo(store *Store) *PageBodyRepo {
	return &PageBodyRepo{store: store}
}

func (r *PageBodyRepo) Observe(ctx context.Context, snapshot block.Snapshot) error {
	return r.store.Do(ctx, func(c context.Context) error {
		if err := r.write(c, upsertBody, snapshot, block.Current, "store the current body"); err != nil {
			return err
		}
		return r.write(c, insertBodyIfAbsent, snapshot, block.Base, "store the base body")
	})
}

func (r *PageBodyRepo) Record(ctx context.Context, pageID, raw string, at time.Time) error {
	snapshot := block.Snapshot{PageID: pageID, Raw: raw, Hash: block.BodyHash(raw), Kind: block.Parse(raw).Kind, ObservedAt: at}
	return r.store.Do(ctx, func(c context.Context) error {
		if err := r.write(c, upsertBody, snapshot, block.Current, "record the written body"); err != nil {
			return err
		}
		return r.write(c, upsertBody, snapshot, block.Base, "record the written body as the base")
	})
}

func (r *PageBodyRepo) write(ctx context.Context, query string, snapshot block.Snapshot, side block.Side, message string) error {
	_, err := execWrite(ctx, r.store.writeFrom(ctx), query, []any{
		snapshot.PageID, string(side), snapshot.Raw, snapshot.Hash, string(snapshot.Kind), formatTime(snapshot.ObservedAt),
	}, nil, message)
	return err
}

func (r *PageBodyRepo) Get(ctx context.Context, pageID string) (current, base block.Snapshot, found bool, err error) {
	snapshots, err := selectAll(ctx, r.store.execFrom(ctx), selectBodies, []any{pageID}, scanSnapshot, "read the page body")
	if err != nil {
		return block.Snapshot{}, block.Snapshot{}, false, err
	}
	for i := range snapshots {
		switch snapshots[i].Side {
		case block.Current:
			current, found = snapshots[i], true
		case block.Base:
			base = snapshots[i]
		}
	}
	return current, base, found, nil
}

func (r *PageBodyRepo) ListBySite(ctx context.Context, siteID string) ([]block.Snapshot, error) {
	return selectAll(ctx, r.store.execFrom(ctx), selectBodiesBySite, []any{siteID}, scanSnapshot, "list the site bodies")
}

func (r *PageBodyRepo) ListCurrent(ctx context.Context, pageIDs []string) ([]block.Snapshot, error) {
	return selectByIDs(ctx, r.store.execFrom(ctx), selectCurrentBodies, orderBodies, pageIDs, scanSnapshot, "list the current bodies of the pages")
}

func (r *PageBodyRepo) Drifted(ctx context.Context, siteID string) ([]string, error) {
	return selectAll(ctx, r.store.execFrom(ctx), selectDrifted, []any{siteID}, scanPageID, "list the bodies that moved from their base")
}

func scanPageID(rows *sql.Rows) (string, error) {
	var pageID string
	err := rows.Scan(&pageID)
	return pageID, err
}

func (r *PageBodyRepo) ReplaceWritten(ctx context.Context, pageID string, hashes []string, runID string, at time.Time) error {
	return r.store.Do(ctx, func(c context.Context) error {
		if _, err := execWrite(c, r.store.writeFrom(c), deleteWritten, []any{pageID}, nil, "clear the written blocks"); err != nil {
			return err
		}
		for _, hash := range hashes {
			if _, err := execWrite(c, r.store.writeFrom(c), insertWritten, []any{pageID, hash, runID, formatTime(at)}, nil, "record a written block"); err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *PageBodyRepo) ListWritten(ctx context.Context, pageID string) ([]block.Written, error) {
	return selectAll(ctx, r.store.execFrom(ctx), selectWritten, []any{pageID}, scanWritten, "list the written blocks")
}

func (r *PageBodyRepo) ListWrittenFor(ctx context.Context, pageIDs []string) ([]block.Written, error) {
	return selectByIDs(ctx, r.store.execFrom(ctx), selectWrittenFor, orderWritten, pageIDs, scanWritten, "list the written blocks of the pages")
}

func scanSnapshot(rows *sql.Rows) (block.Snapshot, error) {
	var (
		snapshot   block.Snapshot
		side, kind string
		observedAt string
	)
	if err := rows.Scan(&snapshot.PageID, &side, &snapshot.Raw, &snapshot.Hash, &kind, &observedAt); err != nil {
		return block.Snapshot{}, err
	}
	snapshot.Side = block.Side(side)
	snapshot.Kind = block.Kind(kind)

	var err error
	if snapshot.ObservedAt, err = parseTime(observedAt); err != nil {
		return block.Snapshot{}, err
	}
	return snapshot, nil
}

func scanWritten(rows *sql.Rows) (block.Written, error) {
	var (
		written   block.Written
		writtenAt string
	)
	if err := rows.Scan(&written.PageID, &written.BlockHash, &written.RunID, &writtenAt); err != nil {
		return block.Written{}, err
	}

	var err error
	if written.WrittenAt, err = parseTime(writtenAt); err != nil {
		return block.Written{}, err
	}
	return written, nil
}
