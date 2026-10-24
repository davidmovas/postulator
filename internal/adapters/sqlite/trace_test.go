package sqlite

import (
	"context"
	"path/filepath"
	"slices"
	"sync"
	"testing"
)

type tally struct {
	mu         sync.Mutex
	statements []string
	rows       int
}

func (t *tally) Statement(sql string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.statements = append(t.statements, sql)
}

func (t *tally) Row() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.rows++
}

func (t *tally) reset() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.statements, t.rows = nil, 0
}

func (t *tally) counted(sql string) (times, rows int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	for _, seen := range t.statements {
		if seen == sql {
			times++
		}
	}
	return times, t.rows
}

func TestATracedStoreCountsTheStatementsAndRowsItRuns(t *testing.T) {
	t.Parallel()

	const (
		read  = `SELECT value FROM json_each('[1,2,3]')`
		write = `INSERT INTO app_meta (key, value, created_at) VALUES (?, ?, ?)`
	)
	for _, key := range [][]byte{nil, testKey()} {
		t.Run(map[bool]string{true: "plain", false: "adiantum"}[key == nil], func(t *testing.T) {
			t.Parallel()

			traced := &tally{}
			store, err := Open(Config{Path: filepath.Join(t.TempDir(), "postulator.db"), Key: key, Trace: traced})
			if err != nil {
				t.Fatalf("open: %v", err)
			}
			t.Cleanup(func() {
				if closeErr := store.Close(); closeErr != nil {
					t.Errorf("close: %v", closeErr)
				}
			})

			if err = store.Read(t.Context(), func(context.Context) error { return nil }); err != nil {
				t.Fatalf("open a reader: %v", err)
			}
			traced.reset()
			values := 0
			err = store.Read(t.Context(), func(c context.Context) error {
				rows, queryErr := store.execFrom(c).QueryContext(c, read)
				if queryErr != nil {
					return queryErr
				}
				defer func() {
					if closeErr := rows.Close(); closeErr != nil {
						t.Errorf("close the rows: %v", closeErr)
					}
				}()
				for rows.Next() {
					values++
				}
				return rows.Err()
			})
			if err != nil {
				t.Fatalf("read: %v", err)
			}
			if times, rows := traced.counted(read); times != 1 || rows != values || values != 3 {
				t.Errorf("the read ran %d times over %d traced rows of %d, want once over three", times, rows, values)
			}

			traced.reset()
			if err = store.Do(t.Context(), func(c context.Context) error {
				_, execErr := store.writeFrom(c).ExecContext(c, write, "traced", "yes", "2026-10-24T00:00:00Z")
				return execErr
			}); err != nil {
				t.Fatalf("write: %v", err)
			}
			if times, rows := traced.counted(write); times != 1 || rows != 0 {
				t.Errorf("the write ran %d times over %d rows, want once over none", times, rows)
			}
			if !slices.Contains(traced.statements, write) {
				t.Errorf("the writer's statements %v miss the write", traced.statements)
			}
		})
	}
}
