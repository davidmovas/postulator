package sqlitetest

import (
	"path/filepath"
	"sync"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
)

type Tally struct {
	Statements int
	Rows       int
}

type Counter struct {
	mu         sync.Mutex
	statements int
	rows       int
	seen       map[string]struct{}
	distinct   []string
}

func NewCounter() *Counter {
	return &Counter{seen: map[string]struct{}{}}
}

func (c *Counter) Statement(sql string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.statements++
	if _, held := c.seen[sql]; !held {
		c.seen[sql] = struct{}{}
		c.distinct = append(c.distinct, sql)
	}
}

func (c *Counter) Row() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.rows++
}

func (c *Counter) Tally() Tally {
	c.mu.Lock()
	defer c.mu.Unlock()
	return Tally{Statements: c.statements, Rows: c.rows}
}

func (c *Counter) Distinct() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]string(nil), c.distinct...)
}

func (c *Counter) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.statements, c.rows, c.seen, c.distinct = 0, 0, map[string]struct{}{}, nil
}

func (c *Counter) Measure(fn func() error) (Tally, error) {
	c.Reset()
	err := fn()
	return c.Tally(), err
}

func (c *Counter) Open(cfg sqlite.Config) (*sqlite.Store, error) {
	cfg.Trace = c
	return sqlite.Open(cfg)
}

func OpenCounted(t testing.TB) (*sqlite.Store, *Counter) {
	t.Helper()

	counter := NewCounter()
	store, err := counter.Open(sqlite.Config{Path: filepath.Join(t.TempDir(), fileName)})
	if err != nil {
		t.Fatalf("open the counted store: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := store.Close(); closeErr != nil {
			t.Errorf("close the counted store: %v", closeErr)
		}
	})
	return store, counter
}
